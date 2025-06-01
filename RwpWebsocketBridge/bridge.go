package main

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"

	"github.com/gorilla/websocket"
)

// BridgeConnection encapsulates all state and channels related to a single raw panel connection through websocket.
type BridgeConnection struct {
	// PanelAddr is the IP:Port address of the panel device.
	PanelAddr string

	// ServerEndpoint is the WebSocket endpoint address for this panel.
	ServerEndpoint string

	// Incoming is the channel for messages being sent TO the panel.
	Incoming chan []*rwp.InboundMessage

	// Outgoing is the channel for messages being received FROM the panel.
	Outgoing chan []*rwp.OutboundMessage

	// Ctx controls cancellation of this panel's connection and goroutines.
	Ctx context.Context

	// Cancel cancels the context and shuts down background routines.
	Cancel context.CancelFunc

	// Wg tracks all goroutines launched for this panel, for clean shutdown.
	Wg sync.WaitGroup

	ClientID     string // "Username" for authentication
	ClientSecret string // "Password" for authentication

	WSConn         *websocket.Conn
	WSMutex        sync.Mutex // To safely send on WSConn
	PanelConnected atomic.Bool
	WSConnected    bool
	wsReady        atomic.Bool
}

// Message envelopes
type WSMessageToServer struct {
	MsgsFromPanel []*rwp.OutboundMessage `json:"msgsFromPanel,omitempty"` // raw json passed back to client
	Auth          *AuthCredentials       `json:"auth,omitempty"`          // optional auth field
}

// AuthCredentials holds username/password login info
type AuthCredentials struct {
	ClientID     string `json:"client_id"`     // "Username" for authentication
	ClientSecret string `json:"client_secret"` // "Password" for authentication
}

type WSMessageFromServer struct {
	MsgsToPanel []*rwp.InboundMessage `json:"msgsToPanel,omitempty"` // raw json we receive from client
	Error       string                `json:"error,omitempty"`       // optional error message
	Message     string                `json:"message,omitempty"`     // optional success message
	Status      string                `json:"status,omitempty"`      // connection status (e.g., "ready", "auth required")
}

// NewBridgeManager initializes a new BridgeConnection instance with the given parameters.
func NewBridgeManager(addr string, wsEndpoint string, clientID, clientSecret string) *BridgeConnection {

	// Create a cancellable context used to control this connection's lifecycle.
	ctx, cancel := context.WithCancel(context.Background())

	// Return a fully initialized struct instance.
	return &BridgeConnection{
		PanelAddr:      addr,                                   // Panel's network address
		ServerEndpoint: wsEndpoint,                             // WebSocket endpoint for this panel
		Incoming:       make(chan []*rwp.InboundMessage, 100),  // Buffered channel for messages to panel
		Outgoing:       make(chan []*rwp.OutboundMessage, 100), // Buffered channel for messages from panel
		Ctx:            ctx,                                    // Context for cancellation
		Cancel:         cancel,                                 // Cancel function
		Wg:             sync.WaitGroup{},                       // WaitGroup to track goroutines
		ClientID:       clientID,                               // Client ID ("username") for authentication
		ClientSecret:   clientSecret,                           // Client Secret ("password") for authentication
	}
}

// Start initializes the connection to the panel and starts the WebSocket client.
func (bc *BridgeConnection) Start() {

	onconnect := func(msg string, binary bool, conn net.Conn) {
		log.Infof("[%s] Panel Connected", bc.PanelAddr)
		bc.PanelConnected.Store(true)
	}

	ondisconnect := func(wasConnected bool) {
		log.Infof("[%s] Panel Disconnected", bc.PanelAddr)
		bc.PanelConnected.Store(false)
		bc.setWSConnected(false)
	}

	// Start the main connection goroutine.
	// The WaitGroup tracks its lifecycle so we can block on shutdown.
	bc.Wg.Add(1)
	go func() {
		defer bc.Wg.Done()

		// This handles the TCP connection to the raw panel device.
		// It uses the supplied onconnect/ondisconnect handlers.
		helpers.ConnectToPanel(
			bc.PanelAddr, // IP:Port to connect to
			bc.Incoming,  // Channel to send messages to panel
			bc.Outgoing,  // Channel to receive messages from panel
			bc.Ctx,       // Context to cancel the connection
			&bc.Wg,       // WaitGroup to track background goroutines
			onconnect,    // Callback when connected
			ondisconnect, // Callback when disconnected
			nil,          // Optional config (nil = use default)
		)
	}()

	// Start the WebSocket connection goroutine.
	// This goroutine connects to the WebSocket server and handles incoming messages.
	bc.Wg.Add(1)
	go func() {
		defer bc.Wg.Done()
		bc.connectWebSocket()
	}()

	// Start a message-handling loop in a separate goroutine.
	// This loop processes all incoming panel messages.
	bc.Wg.Add(1)
	go func() {
		defer bc.Wg.Done()
		bc.messageLoop()
	}()
}

// connectWebSocket establishes a WebSocket connection to the server endpoint and handles incoming messages.
// It will automatically reconnect if the connection is lost.
// It also handles authentication if client ID and secret are provided.
// This function runs in its own goroutine and will exit when the context is cancelled.
func (bc *BridgeConnection) connectWebSocket() {
	for {
		select {
		case <-bc.Ctx.Done():
			log.Debugf("[%s] connectWebSocket() exiting due to shutdown before connection", bc.PanelAddr)
			return
		default:
		}

		if !bc.PanelConnected.Load() {
			select {
			case <-time.After(1 * time.Second):
			case <-bc.Ctx.Done():
				log.Debugf("[%s] connectWebSocket() exiting due to shutdown while waiting for panel", bc.PanelAddr)
				return
			}
			continue
		}

		bc.wsReady.Store(false) // Add this before each connect attempt
		c, _, err := websocket.DefaultDialer.Dial(bc.ServerEndpoint, nil)
		if err != nil {
			log.Errorf("[%s] Failed to connect to WS: %v", bc.PanelAddr, err)
			select {
			case <-time.After(5 * time.Second):
			case <-bc.Ctx.Done():
				log.Debugf("[%s] connectWebSocket() exiting due to shutdown during reconnect delay", bc.PanelAddr)
				return
			}
			continue
		}

		bc.setWSConnected(true)
		log.Infof("[%s] WebSocket connected", bc.PanelAddr)

		bc.WSMutex.Lock()
		bc.WSConn = c
		bc.WSMutex.Unlock()

		// Signal when read loop exits
		wsDone := make(chan struct{})

		bc.Wg.Add(1)
		go func() {
			defer bc.Wg.Done()
			bc.readWebSocketMessages()
			log.Infof("[%s] WebSocket read loop exited", bc.PanelAddr)
			close(wsDone)
		}()

		// Authentication
		if bc.ClientID != "" || bc.ClientSecret != "" {
			envelope := WSMessageToServer{
				Auth: &AuthCredentials{
					ClientID:     bc.ClientID,
					ClientSecret: bc.ClientSecret,
				},
			}
			payload, err := json.Marshal(envelope)
			if err == nil {
				bc.WSMutex.Lock()
				if bc.WSConn != nil {
					bc.WSConn.WriteMessage(websocket.TextMessage, payload)
				}
				bc.WSMutex.Unlock()
			} else {
				log.Warnf("[%s] Failed to encode panel message for WS: %v", bc.PanelAddr, err)
			}
		}

		// Wait for disconnect, panel drop, or shutdown
		select {
		case <-bc.Ctx.Done():
			log.Infof("[%s] connectWebSocket() exiting due to shutdown", bc.PanelAddr)
			bc.setWSConnected(false)
			return

		case <-wsDone:
			log.Infof("[%s] WebSocket disconnected", bc.PanelAddr)
			bc.setWSConnected(false)

		}

		time.Sleep(3 * time.Second)
	}
}

// readWebSocketMessages continuously reads messages from the WebSocket connection.
// It processes incoming messages and forwards them to the Incoming channel.
func (bc *BridgeConnection) readWebSocketMessages() {
	for {
		bc.WSMutex.Lock()
		conn := bc.WSConn
		bc.WSMutex.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Warnf("[%s] Error reading from WS: %v", bc.PanelAddr, err)
			return
		}

		var envelope WSMessageFromServer
		if err := json.Unmarshal(message, &envelope); err != nil {
			log.Warnf("[%s] Invalid WS message format: %v", bc.PanelAddr, err)
			continue
		}

		if envelope.Status != "" {
			log.Infof("[%s] WS status: %s", bc.PanelAddr, envelope.Status)
			if envelope.Status == "ready" {
				log.Infof("[%s] WebSocket connection is ready", bc.PanelAddr)
				bc.wsReady.Store(true) // Set ready flag
			}
			if envelope.Status == "auth_required" {
				log.Warnf("[%s] WebSocket authentication required", bc.PanelAddr)
				continue
			}
		}

		if envelope.Error != "" {
			log.Warnf("[%s] WS error: %s", bc.PanelAddr, envelope.Error)
			continue
		}

		if envelope.Message != "" {
			log.Infof("[%s] WS message: %s", bc.PanelAddr, envelope.Message)
		}

		if len(envelope.MsgsToPanel) > 0 {
			//log.Println(log.Indent(envelope.MsgsToPanel))
			if bc.wsReady.Load() { // Only forward messages if "ready"
				select {
				case bc.Incoming <- envelope.MsgsToPanel:
				default:
					log.Warnf("[%s] Dropping message: panel not ready", bc.PanelAddr)
				}
			} else {
				log.Debugf("[%s] Ignoring MsgsToPanel: WS not ready", bc.PanelAddr)
			}
		}
	}
}

// setWSConnected updates the WebSocket connection state and closes the connection if disconnected.
func (bc *BridgeConnection) setWSConnected(connected bool) {
	bc.WSMutex.Lock()
	defer bc.WSMutex.Unlock()

	bc.WSConnected = connected

	if !bc.WSConnected && bc.WSConn != nil {
		_ = bc.WSConn.Close() // Best-effort close
		bc.WSConn = nil       // Clear the connection
	}
}

// messageLoop continuously processes messages received from the panel.
func (bc *BridgeConnection) messageLoop() {
	for {
		select {
		case <-bc.Ctx.Done():
			log.Infof("[%s] Stopping message loop", bc.PanelAddr)
			return

		case msgs := <-bc.Outgoing:
			if !bc.wsReady.Load() {
				log.Warnf("[%s] Skipping outgoing message: WebSocket not ready", bc.PanelAddr)
				continue
			}

			payload, err := json.Marshal(WSMessageToServer{MsgsFromPanel: msgs})
			if err != nil {
				log.Warnf("[%s] Failed to encode panel message for WS: %v", bc.PanelAddr, err)
				continue
			}

			bc.WSMutex.Lock()
			conn := bc.WSConn
			if !(bc.WSConnected && conn != nil) {
				log.Warnf("[%s] Not sending message: WebSocket not connected", bc.PanelAddr)
				bc.WSMutex.Unlock()
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Warnf("[%s] WebSocket write failed: %v", bc.PanelAddr, err)
			}
			bc.WSMutex.Unlock()
		}
	}
}

// Shutdown gracefully shuts down the connection and all associated goroutines.
// It cancels the context, waits for all goroutines to finish, and closes the WebSocket connection.
func (bc *BridgeConnection) Shutdown() {
	log.Infof("[%s] Shutting down connection", bc.PanelAddr)
	bc.Cancel()  // Cancel all goroutines
	bc.Wg.Wait() // Wait for all goroutines to finish
	log.Infof("[%s] Shutdown complete", bc.PanelAddr)
}
