package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/gorilla/websocket"
	log "github.com/s00500/env_logger"
)

// Message envelopes
type WSMessageToServer struct {
	MsgsFromPanel []*rwp.OutboundMessage `json:"msgsFromPanel,omitempty"` // raw json we receive from client
	Auth          *AuthCredentials       `json:"auth,omitempty"`          // optional auth field
}

// AuthCredentials holds username/password login info
type AuthCredentials struct {
	ClientID     string `json:"client_id"`     // "Username" for authentication
	ClientSecret string `json:"client_secret"` // "Password" for authentication
}

type WSMessageFromServer struct {
	MsgsToPanel []*rwp.InboundMessage `json:"msgsToPanel,omitempty"` // raw json passed back to client
	Error       string                `json:"error,omitempty"`       // optional error message
	Message     string                `json:"message,omitempty"`     // optional success message
	Status      string                `json:"status,omitempty"`      // connection status (e.g., "ready", "auth_required")
}

type WSConnection struct {
	conn        *websocket.Conn
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	writeMu     sync.Mutex
	sendChan    chan []byte // Optional: async outgoing messages
	demoManager *DemoManager
}

func NewWSConnection(conn *websocket.Conn) *WSConnection {
	ctx, cancel := context.WithCancel(context.Background())

	ws := &WSConnection{
		conn:     conn,
		ctx:      ctx,
		cancel:   cancel,
		sendChan: make(chan []byte, 100),
	}

	// Enable debug output if the flag is set
	if debugWS {
		log.Println("WebSocket debug mode enabled")
	}

	// Standard Raw Panel initialization message
	// This message activates the panel, requests info and topology, sets heartbeat timer, etc.
	ws.sendJSON(WSMessageFromServer{
		MsgsToPanel: []*rwp.InboundMessage{
			{
				Command: &rwp.Command{
					ActivatePanel:         true,
					SendPanelInfo:         true,
					SendPanelTopology:     true,
					ReportHWCavailability: true,
					ClearAll:              true, // Clear all previous messages
					SetHeartBeatTimer: &rwp.HeartBeatTimer{
						Value: 3000, // Heartbeat in milliseconds. Panel will send a ping every 3 seconds and we must respond with an ack or another message.
					},
				},
			},
		},
	})

	ws.demoManager = NewDemoManager(ws.sendDemoMessage)
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.demoManager.run(ws)
	}()

	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.readLoop()
	}()

	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.pingLoop()
	}()

	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.writeLoop()
	}()

	return ws
}

// sendDemoMessage sends a demo message to the panel.
func (ws *WSConnection) sendDemoMessage(msg *rwp.InboundMessage) {
	ws.sendJSON(WSMessageFromServer{
		MsgsToPanel: []*rwp.InboundMessage{msg},
	})
}

// readLoop continuously reads messages from the WebSocket connection.
func (ws *WSConnection) readLoop() {

	// Set a read deadline to avoid blocking indefinitely.
	ws.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	ws.conn.SetPongHandler(func(string) error {
		ws.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		return nil
	})

	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
			_, msg, err := ws.conn.ReadMessage()
			if err != nil {
				log.Println("readLoop: error reading message:", err)
				ws.cancel()
				return
			}

			if debugWS {
				log.Printf("<<< Received WS message:\n%s\n", msg)
			}

			// Handle the received message
			ws.handleMessage(msg)
		}
	}
}

// handleMessage processes incoming messages from the WebSocket connection.
// It unmarshals the JSON message and processes each panel message accordingly.
// If the message is a ping, it sends an acknowledgment back to the panel.
// If the message contains events, it forwards them to the demo manager for processing.
// If the message contains a topology, it updates the demo manager with the new topology.
func (ws *WSConnection) handleMessage(msg []byte) {
	var envelope WSMessageToServer
	if err := json.Unmarshal(msg, &envelope); err != nil {
		log.Println("Invalid JSON:", err)
		return
	}

	if len(envelope.MsgsFromPanel) > 0 {
		for _, msg := range envelope.MsgsFromPanel {
			if msg.FlowMessage == 1 { // Ping:
				log.Debugln("Received ping message, sending ack")
				rwpmsg := WSMessageFromServer{
					MsgsToPanel: []*rwp.InboundMessage{
						{
							FlowMessage: 2,
						},
					},
				}
				ws.sendJSON(rwpmsg)
			}

			if msg.PanelTopology != nil {
				ws.demoManager.OnTopology(msg.PanelTopology.Json)
			}

			for _, evt := range msg.Events {
				ws.demoManager.OnEvent(evt)
			}
		}
	}
}

// pingLoop sends periodic pings to the WebSocket connection.
// It sends both a protocol-level WebSocket ping and an application-level FlowMessage ping.
func (ws *WSConnection) pingLoop() {

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ws.ctx.Done():
			return

		case <-ticker.C:

			// --- Send protocol-level WebSocket ping ---
			ws.writeMu.Lock()
			err := ws.conn.WriteControl(
				websocket.PingMessage,
				[]byte("ping"),                // Optional payload
				time.Now().Add(1*time.Second), // Deadline for control frame
			)
			ws.writeMu.Unlock()

			if err != nil {
				log.Println("pingLoop: WebSocket protocol-level ping failed:", err)
				ws.cancel() // Trigger shutdown
				return
			}

			// --- Send app-level FlowMessage ping ---
			appPing := WSMessageFromServer{
				MsgsToPanel: []*rwp.InboundMessage{
					{FlowMessage: 1},
				},
			}
			ws.sendJSON(appPing)
		}
	}
}

// sendJSON marshals the given value to JSON and sends it over the WebSocket connection.
// If the send channel is full, it logs an error and drops the message.
func (ws *WSConnection) sendJSON(v interface{}) {
	msg, err := json.Marshal(v)
	if err != nil {
		log.Println("sendJSON: marshal error:", err)
		return
	}

	if debugWS {
		log.Printf(">>> Sending WS message:\n%s\n", msg)
	}

	select {
	case ws.sendChan <- msg:
	default:
		log.Println("sendJSON: sendChan full, dropping message")
	}
}

// writeLoop continuously writes messages from the send channel to the WebSocket connection.
// It locks the write mutex to ensure thread-safe writes.
func (ws *WSConnection) writeLoop() {
	for {
		select {
		case <-ws.ctx.Done():
			return
		case msg, ok := <-ws.sendChan:
			if !ok {
				return // channel closed
			}
			ws.writeMu.Lock()
			err := ws.conn.WriteMessage(websocket.TextMessage, msg)
			ws.writeMu.Unlock()
			if err != nil {
				log.Println("writeLoop: error writing message:", err)
				ws.cancel()
				return
			}
		}
	}
}

// Shutdown gracefully closes the WebSocket connection and waits for all goroutines to finish.
// It cancels the context, waits for the read, write, and ping loops to finish, and then closes the connection.
func (ws *WSConnection) Shutdown() {
	ws.cancel()
	ws.wg.Wait()
	ws.conn.Close()
	log.Println("WSConnection shutdown complete")
}

// authenticateWebSocket handles the authentication process for the WebSocket connection.
func authenticateWebSocket(conn *websocket.Conn) bool {

	statusMsg := WSMessageFromServer{
		Status:  "auth_required",
		Message: "Please authenticate with Client ID/Client Secret.",
	}
	if data, err := json.Marshal(statusMsg); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Set a deadline for the first message to allow for authentication timeout
	defer conn.SetReadDeadline(time.Time{})               // Clear deadline after auth

	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("WebSocket read error (likely timeout or invalid message):", err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"authentication timeout or failure"}`))
		return false
	}

	var envelope WSMessageToServer
	if err := json.Unmarshal(msg, &envelope); err != nil {
		log.Println("Invalid JSON in first message:", err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid json format"}`))
		return false
	}

	if envelope.Auth == nil {
		log.Println("Missing authentication in first message")
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"authentication required"}`))
		return false
	}
	if envelope.Auth.ClientID != authClientId || envelope.Auth.ClientSecret != authClientSecret {
		log.Println("Invalid credentials")
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid credentials"}`))
		return false
	}

	log.Println("Client authenticated successfully")

	return true
}

// sendReadyMessage sends a "ready" message to the client after successful authentication.
// This message indicates that the connection is established and possibly authenticated.
func sendReadyMessage(conn *websocket.Conn) {
	readyMsg := WSMessageFromServer{
		Status:  "ready",
		Message: "Connection established and authenticated.",
	}
	if data, err := json.Marshal(readyMsg); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}
