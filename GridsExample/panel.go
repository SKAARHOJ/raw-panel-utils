package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"sync"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
	"github.com/fogleman/gg"
	log "github.com/s00500/env_logger"
)

// PanelConnection encapsulates all state and channels related to a single raw panel connection.
type PanelConnection struct {
	// Addr is the IP:Port address of the panel device.
	Addr string

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

	// TopologyData holds the most recently received parsed topology.
	TopologyData *topology.Topology

	// HWCMap stores hardware component availability/status mappings.
	HWCMap map[uint32]uint32
}

// Start launches the panel connection process and begins listening for messages.
// It sets up the ConnectToPanel call with proper handlers and spawns goroutines for async operations.
func (p *PanelConnection) Start() {

	// Define what happens when the panel successfully connects.
	onconnect := func(msg string, binary bool, conn net.Conn) {
		log.Infof("[%s] Connected", p.Addr)

		// Once connected, immediately send a set of initial commands to activate and configure the panel.
		p.Incoming <- []*rwp.InboundMessage{
			{
				Command: &rwp.Command{
					ActivatePanel:         true,
					SendPanelInfo:         true,
					SendPanelTopology:     true,
					ReportHWCavailability: true,
					SetHeartBeatTimer: &rwp.HeartBeatTimer{
						Value: 3000, // Heartbeat in milliseconds
					},
				},
			},
		}
	}

	// Define what happens when the connection is lost or closed.
	ondisconnect := func(wasConnected bool) {
		log.Infof("[%s] Disconnected (was connected: %v)", p.Addr, wasConnected)
	}

	// Start the main connection goroutine.
	// The WaitGroup tracks its lifecycle so we can block on shutdown.
	p.Wg.Add(1)
	go func() {
		defer p.Wg.Done()

		// This handles the TCP connection to the raw panel device.
		// It uses the supplied onconnect/ondisconnect handlers.
		helpers.ConnectToPanel(
			p.Addr,       // IP:Port to connect to
			p.Incoming,   // Channel to send messages to panel
			p.Outgoing,   // Channel to receive messages from panel
			p.Ctx,        // Context to cancel the connection
			&p.Wg,        // WaitGroup to track background goroutines
			onconnect,    // Callback when connected
			ondisconnect, // Callback when disconnected
			nil,          // Optional config (nil = use default)
		)
	}()

	// Start a message-handling loop in a separate goroutine.
	// This loop processes all incoming panel messages.
	go p.messageLoop()
}

// messageLoop continuously processes messages received from the panel.
// It responds to protocol-specific signals (like ping) and can be extended to handle topology/state/etc.
func (p *PanelConnection) messageLoop() {
	// Set up a ticker to send ping every 10 seconds
	pingTicker := time.NewTicker(10 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		// Context was canceled — stop the loop
		case <-p.Ctx.Done():
			log.Infof("[%s] Stopping message loop", p.Addr)
			return

		// Send periodic ping to the panel
		case <-pingTicker.C:
			log.Debugf("[%s] Sending periodic ping", p.Addr)
			p.Incoming <- []*rwp.InboundMessage{{FlowMessage: 1}}

		// Handle incoming messages from panel
		case msgs := <-p.Outgoing:
			for _, msg := range msgs {
				log.Infof("[%s] Received message: %+v", p.Addr, msg)

				// Respond to panel's ping (ping = 1, pong = 2)
				if msg.FlowMessage == 1 {
					log.Debugf("[%s] Responding to ping with pong", p.Addr)
					p.Incoming <- []*rwp.InboundMessage{{FlowMessage: 2}}
				}

				// Handle panel info (model, serial, etc.)
				if msg.PanelInfo != nil {
					model := msg.PanelInfo.Model
					serial := msg.PanelInfo.Serial
					log.Infof("[%s] Panel Model: %s, Serial: %s", p.Addr, model, serial)
				}

				// Handle basic panel trigger events
				if msg.Events != nil {
					for _, event := range msg.Events {
						log.Infof("[%s] Triggered event: %s", p.Addr, event)
					}
				}

				// Handle topology JSON and unmarshal it into struct
				if msg.PanelTopology != nil && msg.PanelTopology.Json != "" {
					err := json.Unmarshal([]byte(msg.PanelTopology.Json), p.TopologyData)
					if err != nil {
						log.Errorf("[%s] Failed to unmarshal topology JSON: %v", p.Addr, err)
					} else {
						log.Infof("[%s] Successfully parsed topology JSON with %d elements", p.Addr, log.Indent(p.TopologyData))
						p.AnalyzeTopologyForGrids(p.TopologyData)
					}
				}
			}
		}
	}
}

// AnalyzeTopologyForGrids analyzes grid layout and type definitions for a given topology.
// It handles both uniform grids (MasterTypeIndex set) and heterogeneous ones (per-HWC typing),
// and includes display info for each grid element or the master type.
func (p *PanelConnection) AnalyzeTopologyForGrids(topo *topology.Topology) {
	log.Infof("[%s] Analyzing topology for grids...", p.Addr)

	for _, grid := range topo.Grids {
		log.Infof("[%s] Grid: %s, Size: %dx%d", p.Addr, grid.Title, grid.Cols, grid.Rows)

		// Case 1: Uniform grid (MasterTypeIndex applies to all HWCs)
		if grid.MasterTypeIndex != 0 {
			typeDef, ok := topo.TypeIndex[grid.MasterTypeIndex]
			if ok {
				log.Infof("[%s] Grid uses MasterTypeIndex %d: %s", p.Addr, grid.MasterTypeIndex, typeDef.Desc)
				log.Infof("[%s] - InputType: %s, OutputType: %s, Render: %s", p.Addr, typeDef.In, typeDef.Out, typeDef.Render)

				if typeDef.HasDisplay() {
					disp := typeDef.DisplayInfo()
					log.Infof("[%s] - Has Display: %dx%d, Type: %s, Shrink: %d, Border: %d", p.Addr, disp.W, disp.H, disp.Type, disp.Shrink, disp.Border)
				}
			} else {
				log.Warnf("[%s] Grid %s has invalid MasterTypeIndex %d", p.Addr, grid.Title, grid.MasterTypeIndex)
			}
		} else {
			// Case 2: Per-element types
			for rowIdx, row := range grid.HWcMap {
				for colIdx, elem := range row {
					hwcID := elem.Id
					typeDef, err := topo.GetHWCtype(hwcID)
					if err != nil {
						log.Warnf("[%s] Could not find type for HWC ID %d at [%d,%d]", p.Addr, hwcID, rowIdx, colIdx)
						continue
					}

					hwcLabel := topo.GetHWCtext(hwcID)
					log.Infof("[%s] HWC ID: %d (%s) at [%d,%d] → InputType: %s, Desc: %s",
						p.Addr, hwcID, hwcLabel, rowIdx, colIdx, typeDef.GetInputType(), typeDef.Desc)

					// If AltDisplayId is present, use that type for display info
					usedAlt := false
					altId := uint32(0)
					if elem.AltDisplayId != 0 {
						if altTypeDef, err := topo.GetHWCtype(elem.AltDisplayId); err == nil {
							typeDef = altTypeDef
							usedAlt = true
							altId = elem.AltDisplayId
						}
					}

					if typeDef.HasDisplay() {
						disp := typeDef.DisplayInfo()
						origin := ""
						if usedAlt {
							origin = fmt.Sprintf(" via AltDisplay=%d", altId)
						}
						log.Infof("[%s]   ↳ Has Display: %dx%d, Type: %s, Shrink: %d, Border: %d%s",
							p.Addr, disp.W, disp.H, disp.Type, disp.Shrink, disp.Border, origin)
					}
				}
			}
		}
	}
}

func createTestImage(W int, H int, imageType string, label string) ([]byte, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, W, H))
	dc := gg.NewContextForRGBA(canvas)

	// Fill background with gradient or solid
	grad := gg.NewLinearGradient(0, float64(H), float64(W), 0)
	if imageType == "color" {
		grad.AddColorStop(0, color.RGBA{0, 255, 0, 255})
		grad.AddColorStop(1, color.RGBA{0, 0, 255, 255})
		grad.AddColorStop(0.5, color.RGBA{255, 0, 0, 255})
		dc.SetFillStyle(grad)
	} else if imageType == "gray" {
		grad.AddColorStop(0, color.RGBA{0, 0, 0, 255})
		grad.AddColorStop(1, color.RGBA{255, 255, 255, 255})
		dc.SetFillStyle(grad)
	} else {
		dc.SetColor(color.Black)
	}

	dc.Clear()
	dc.SetColor(color.White)
	dc.DrawRectangle(0, 0, float64(W), float64(H))
	dc.Stroke()

	// Use built-in font and draw the label (e.g., "3,7")
	dc.SetColor(color.White)
	dc.DrawStringAnchored(label, float64(W)/2, float64(H)/2, 0.5, 0.5)

	// Encode image to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
