package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"net"
	"sync"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	monogfx "github.com/SKAARHOJ/rawpanel-lib/ibeam_lib_monogfx"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
	rawpanelproc "github.com/SKAARHOJ/rawpanel-processors"
	"github.com/fogleman/gg"
	log "github.com/s00500/env_logger"
)

const (
	HeartbeatInterval = 3000 // HeartbeatInterval is the time in milliseconds between heartbeats FROM the panel to us.
	PingInterval      = 10 * time.Second
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
						Value: HeartbeatInterval, // Heartbeat in milliseconds
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
	pingTicker := time.NewTicker(PingInterval)
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

		if grid.MasterTypeIndex != 0 {
			// Uniform type grid
			typeDef, ok := topo.TypeIndex[grid.MasterTypeIndex]
			if ok {
				log.Infof("[%s] Grid uses MasterTypeIndex %d: %s", p.Addr, grid.MasterTypeIndex, typeDef.Desc)
				log.Infof("[%s] - InputType: %s, OutputType: %s, Render: %s", p.Addr, typeDef.In, typeDef.Out, typeDef.Render)

				if typeDef.HasDisplay() {
					disp := typeDef.DisplayInfo()
					log.Infof("[%s] - Has Display: %dx%d, Type: %s, Shrink: %d, Border: %d",
						p.Addr, disp.W, disp.H, disp.Type, disp.Shrink, disp.Border)

					for _, row := range grid.HWcMap {
						for _, elem := range row {
							label := fmt.Sprintf("%dx%d", disp.W, disp.H)
							for _, hwcID := range elem.Ids {
								p.SendDisplayTestImage(hwcID, disp, label, disp.Type) // If there are multiple IDs, send to each although we don't know if they all support displays. But that is the blindness we choose with MasterTypeIndex.
							}
						}
					}
				}
			} else {
				log.Warnf("[%s] Grid %s has invalid MasterTypeIndex %d", p.Addr, grid.Title, grid.MasterTypeIndex)
			}
		} else {
			// Per-element type grid
			for rowIdx, row := range grid.HWcMap { // Iterate over each row in the HWC map grid
				for colIdx, elem := range row { // Iterate over each element in the current row
					for _, hwcID := range elem.Ids { // Iterate over all HWC IDs associated with this element
						typeDef, err := topo.GetHWCtype(hwcID) // Get the type definition for the current HWC ID
						if err != nil {
							log.Warnf("[%s] Could not find type for HWC ID %d at [%d,%d]", p.Addr, hwcID, rowIdx, colIdx)
							continue
						}

						hwcLabel := topo.GetHWCtext(hwcID)                                     // Get the label/text associated with the HWC ID
						log.Infof("[%s] HWC ID: %d (%s) at [%d,%d] → InputType: %s, Desc: %s", // Log detailed info about the HWC ID
							p.Addr, hwcID, hwcLabel, rowIdx, colIdx, typeDef.GetInputType(), typeDef.Desc)

						if typeDef.HasDisplay() { // If the HWC type has a display
							disp := typeDef.DisplayInfo()                                              // Get display configuration/info
							log.Infof("[%s]   ↳ Has Display: %dx%d, Type: %s, Shrink: %d, Border: %d", // Log display-related parameters
								p.Addr, disp.W, disp.H, disp.Type, disp.Shrink, disp.Border)

							label := fmt.Sprintf("%dx%d", disp.W, disp.H)         // Create label text representing display resolution
							p.SendDisplayTestImage(hwcID, disp, label, disp.Type) // Send a test image to the display based on type
						}
					}
				}
			}

		}
	}

	// Apply random LED color per grid
	p.SetRandomColorStatePerGrid(topo)
}

// SetRandomColorStatePerGrid applies a unique RGB color to each defined grid in the topology.
func (p *PanelConnection) SetRandomColorStatePerGrid(topo *topology.Topology) {
	if topo == nil || len(topo.Grids) == 0 {
		log.Warnf("[%s] No grid layout in topology", p.Addr)
		return
	}

	for _, grid := range topo.Grids {
		hwcIDs := make([]uint32, 0, grid.Cols*grid.Rows)
		for _, row := range grid.HWcMap {
			for _, elem := range row {
				hwcIDs = append(hwcIDs, elem.Ids...)
			}
		}

		if len(hwcIDs) == 0 {
			log.Infof("[%s] Grid %s contains no valid HWCs", p.Addr, grid.Title)
			continue
		}

		// Generate a unique random RGB color
		color := &rwp.HWCColor{
			ColorRGB: &rwp.ColorRGB{
				Red:   uint32(rand.Intn(256)),
				Green: uint32(rand.Intn(256)),
				Blue:  uint32(rand.Intn(256)),
			},
		}

		state := &rwp.HWCState{
			HWCIDs: hwcIDs,
			HWCMode: &rwp.HWCMode{
				State: rwp.HWCMode_ON,
			},
			HWCColor: color,
		}

		p.Incoming <- []*rwp.InboundMessage{
			{States: []*rwp.HWCState{state}},
		}

		log.Infof("[%s] Set random color to %d HWCs in grid '%s'", p.Addr, len(hwcIDs), grid.Title)
	}
}

func (p *PanelConnection) SendDisplayTestImage(hwcID uint32, disp *topology.TopologyHWcTypeDef_Display, label string, imageType string) {

	// Create a test image
	inImg, err := createTestImage(disp.W, disp.H, imageType, label)
	if err != nil {
		log.Errorf("[%s] Failed to create test image for HWC %d: %v", p.Addr, hwcID, err)
		return
	}

	// Initialize a raw panel graphics state:
	img := rwp.HWCGfx{
		W: uint32(disp.W),
		H: uint32(disp.H),
	}

	// Use monoImg to create a base:
	monoImg := monogfx.MonoImg{}
	monoImg.NewImage(int(img.W), int(img.H))

	// Set up image type:
	switch imageType {
	case "color":
		img.ImageType = rwp.HWCGfx_RGB16bit
		img.ImageData = monoImg.GetImgSliceRGB()
	case "gray":
		img.ImageType = rwp.HWCGfx_Gray4bit
		img.ImageData = monoImg.GetImgSliceGray()
	default:
		img.ImageType = rwp.HWCGfx_MONO
		img.ImageData = monoImg.GetImgSlice()
	}

	// Set up bounds:
	imgBounds := rawpanelproc.ImageBounds{X: 0, Y: 0, W: int(img.W), H: int(img.H)}

	// Map the image onto the canvas
	rawpanelproc.RenderImageOnCanvas(&img, inImg, imgBounds, "", "", "")

	// Prepare and send the state update with embedded image
	state := &rwp.HWCState{
		HWCIDs:  []uint32{hwcID},
		HWCGfx:  &img,
		HWCText: nil,
	}

	p.Incoming <- []*rwp.InboundMessage{
		{States: []*rwp.HWCState{state}},
	}
}

func (p *PanelConnection) SendDisplayTestImageAsPNG(hwcID uint32, disp *topology.TopologyHWcTypeDef_Display, label string, imageType string) {
	// Create the test image
	img, err := createTestImage(disp.W, disp.H, imageType, label)
	if err != nil {
		log.Errorf("[%s] Failed to create image for HWC %d: %v", p.Addr, hwcID, err)
		return
	}

	// Encode the image as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Errorf("[%s] Failed to encode image for HWC %d: %v", p.Addr, hwcID, err)
		return
	}
	imgData := buf.Bytes()

	// Determine encoding based on image type
	encoding := rwp.ProcGfxConverter_MONO
	switch imageType {
	case "color":
		encoding = rwp.ProcGfxConverter_RGB16bit
	case "gray":
		encoding = rwp.ProcGfxConverter_Gray4bit
	}

	// Prepare and send the state update with embedded image
	state := &rwp.HWCState{
		HWCIDs: []uint32{hwcID},
		Processors: &rwp.Processors{
			GfxConv: &rwp.ProcGfxConverter{
				W:         uint32(disp.W),
				H:         uint32(disp.H),
				ImageType: encoding,
				Scaling:   rwp.ProcGfxConverter_STRETCH,
				ImageData: imgData,
			},
		},
	}

	p.Incoming <- []*rwp.InboundMessage{
		{States: []*rwp.HWCState{state}},
	}
}

func createTestImage(W int, H int, imageType string, label string) (image.Image, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, W, H))
	dc := gg.NewContextForRGBA(canvas)

	// Define the background gradient
	if imageType == "color" || imageType == "gray" {
		grad := gg.NewLinearGradient(0, 0, float64(W), float64(H))
		if imageType == "color" {
			grad.AddColorStop(0, color.RGBA{255, 0, 0, 255})   // Red
			grad.AddColorStop(0.5, color.RGBA{0, 255, 0, 255}) // Green
			grad.AddColorStop(1, color.RGBA{0, 0, 255, 255})   // Blue
		} else {
			grad.AddColorStop(0, color.RGBA{0, 0, 0, 255})       // Black
			grad.AddColorStop(1, color.RGBA{255, 255, 255, 255}) // White
		}
		dc.SetFillStyle(grad)
		dc.DrawRectangle(0, 0, float64(W), float64(H))
		dc.Fill()
	} else {
		dc.SetColor(color.Black)
		dc.Clear()
	}

	// Border
	dc.SetColor(color.White)
	dc.DrawRectangle(0, 0, float64(W), float64(H))
	dc.Stroke()

	// Label
	dc.SetColor(color.White)
	dc.DrawStringAnchored(label, float64(W)/2, float64(H)/2, 0.5, 0.5)

	return canvas, nil
}
