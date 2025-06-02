package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"sync"
	"time"

	monogfx "github.com/SKAARHOJ/rawpanel-lib/ibeam_lib_monogfx"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
	rawpanelproc "github.com/SKAARHOJ/rawpanel-processors"

	"github.com/fogleman/gg"
	log "github.com/s00500/env_logger"
)

//go:embed pictures/*
var picturesFS embed.FS

// DemoManager manages the demo functionality for this Raw Panel WebSocket server reference implementation.
// It handles the panel topology, color cycling, and event triggers to display messages on the panels.
// It runs a ticker to periodically update the panel states and listens for events to trigger specific actions.
type DemoManager struct {
	mu               sync.Mutex
	topology         *topology.Topology
	currentColor     int
	paused           bool
	lastTrigger      time.Time
	triggerChan      chan *rwp.HWCEvent
	sendFunc         func(msg *rwp.InboundMessage)
	demoTicker       *time.Ticker
	colorBase        int
	textSnippets     []string
	textSnippetsUTF8 []string
	iconCodes        []string
	colorMax         int
	hwcList          []uint32
	resumeTimer      *time.Timer
	hwcDisplayMap    map[uint32]*topology.TopologyHWcTypeDef
}

// NewDemoManager creates a new DemoManager instance with the provided send function.
func NewDemoManager(sendFunc func(msg *rwp.InboundMessage)) *DemoManager {
	return &DemoManager{
		colorBase:    2,
		colorMax:     17,
		textSnippets: []string{"Hello", "World", "RawPanel", "SKAARHOJ", "Demo", "Trigger", "Event", "ColorCycle"},
		textSnippetsUTF8: []string{
			"Привет", "Мир", "RawPanel ÆØÅ", "SKÅRHØJ", "Демо", "Триггер", "Событие", "ЦиклЦвета",
			"こんにちは", "世界", "RawPanel", "SKAARHOJ", "デモ", "トリガー", "イベント", "カラーチャート",
			"مرحبا", "العالم", "RawPanel", "SKAARHOJ", "عرض", "حدث", "تفعيل", "دورة اللون"},
		iconCodes:     []string{"e056", "e8b6", "e88a", "e5d2", "e5cd", "e8b8", "e86c", "e838", "e627", "e92b", "e7ef", "e897", "e068", "e440", "f07f", "f0d1"}, // See complete list at https://fonts.google.com/icons, click on the icon and see it's "Code Point" in the right inspector (scroll down)
		sendFunc:      sendFunc,
		triggerChan:   make(chan *rwp.HWCEvent, 100),
		hwcDisplayMap: make(map[uint32]*topology.TopologyHWcTypeDef),
	}
}

// Run begins the demo manager's operation, starting the ticker and running the main loop.
func (dm *DemoManager) run(ws *WSConnection) {
	dm.demoTicker = time.NewTicker(100 * time.Millisecond)
	hwcIndex := 0

	for {
		select {
		case <-ws.ctx.Done():
			log.Debugln("DemoManager: Stopping tickers and timers")
			dm.demoTicker.Stop()
			if dm.resumeTimer != nil {
				dm.resumeTimer.Stop()
			}
			return

		case <-dm.demoTicker.C:
			dm.mu.Lock()
			if dm.paused || dm.topology == nil || len(dm.hwcList) == 0 {
				dm.mu.Unlock()
				continue
			}

			hwc := dm.hwcList[hwcIndex]
			hwcIndex = (hwcIndex + 1) % len(dm.hwcList) // Loop over list

			typeDef := dm.hwcDisplayMap[hwc] // May be nil if no display
			state := dm.generateDisplayContent(hwc, typeDef)

			msg := &rwp.InboundMessage{
				States: []*rwp.HWCState{state},
			}
			dm.sendFunc(msg)
			dm.mu.Unlock()

		case evt := <-dm.triggerChan:
			dm.mu.Lock()
			dm.paused = true
			dm.lastTrigger = time.Now()

			// --- create the display text, as updated previously ---
			displayText := &rwp.HWCText{
				Title:          fmt.Sprintf("HWC #%d", evt.HWCID),
				Formatting:     7,
				SolidHeaderBar: true,
				PairMode:       4,
			}

			switch {
			case evt.Binary != nil:
				displayText.Textline1 = "Binary"
				displayText.Textline2 = "Up"
				if evt.Binary.Pressed {
					displayText.Textline2 = "Down"
				}
				if evt.Binary.Edge != 0 {
					displayText.Textline2 += fmt.Sprintf(" %s", evt.Binary.Edge.String())
				}
			case evt.Pulsed != nil:
				displayText.Textline1 = "Pulse"
				displayText.Textline2 = fmt.Sprintf("Value: %d", evt.Pulsed.Value)
			case evt.Absolute != nil:
				displayText.Textline1 = "Analog"
				displayText.Textline2 = fmt.Sprintf("Val: %d", evt.Absolute.Value)
			case evt.Speed != nil:
				displayText.Textline1 = "Speed"
				displayText.Textline2 = fmt.Sprintf("Val: %d", evt.Speed.Value)
			default:
				displayText.Textline1 = "Unknown"
				displayText.Textline2 = "?"
			}

			dm.currentColor++
			color := (dm.currentColor % (dm.colorMax - dm.colorBase + 1)) + dm.colorBase

			msg := &rwp.InboundMessage{
				States: []*rwp.HWCState{
					{
						HWCIDs: dm.hwcList,
						HWCMode: &rwp.HWCMode{
							State: rwp.HWCMode_ON,
						},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: rwp.ColorIndex_Colors(color),
							},
						},
						HWCText: displayText,
					},
				},
			}
			dm.sendFunc(msg)

			// --- Reset or create the resumeTimer ---
			if dm.resumeTimer != nil {
				dm.resumeTimer.Stop()
			}
			dm.resumeTimer = time.AfterFunc(3*time.Second, func() {
				dm.mu.Lock()
				dm.paused = false
				dm.mu.Unlock()
			})

			dm.mu.Unlock()
		}
	}
}

// OnTopology processes the incoming topology JSON string, updates the internal state,
func (dm *DemoManager) OnTopology(jsonStr string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var topology topology.Topology
	if err := json.Unmarshal([]byte(jsonStr), &topology); err != nil {
		log.Errorln("Failed to parse topology:", err)
		return
	}
	dm.topology = &topology
	dm.hwcList = make([]uint32, 0)
	for _, hwc := range topology.HWc {
		if hwc.Type > 0 {
			dm.hwcList = append(dm.hwcList, hwc.Id)

			if typeDef, err := topology.GetHWCtype(hwc.Id); err == nil && typeDef.HasDisplay() {
				dm.hwcDisplayMap[hwc.Id] = typeDef
			}
		}
	}
	log.Println("DemoManager: Topology loaded with", len(dm.hwcList), "HWCs")
}

// OnEvent processes incoming hardware events, triggering the demo functionality.
func (dm *DemoManager) OnEvent(evt *rwp.HWCEvent) {
	dm.triggerChan <- evt
}

func (dm *DemoManager) generateDisplayContent(hwcID uint32, typeDef *topology.TopologyHWcTypeDef) *rwp.HWCState {
	color := dm.currentColor
	dm.currentColor++
	if dm.currentColor > dm.colorMax {
		dm.currentColor = dm.colorBase
	}

	state := &rwp.HWCState{
		HWCIDs: []uint32{hwcID},
		HWCMode: &rwp.HWCMode{
			State: rwp.HWCMode_ON,
		},
		HWCColor: &rwp.HWCColor{
			ColorIndex: &rwp.ColorIndex{
				Index: rwp.ColorIndex_Colors(color),
			},
		},
	}

	// Only add text if HWC has display
	if typeDef != nil && typeDef.HasDisplay() {

		disp := typeDef.DisplayInfo()
		if disp == nil {
			return state // fallback: no display info available
		}

		switch rand.Intn(7) { // Now includes HWCGfx case
		case 0:
			// HWCText
			state.HWCText = &rwp.HWCText{
				Formatting: 7,
				Title:      dm.textSnippets[rand.Intn(len(dm.textSnippets))],
				Textline1:  dm.textSnippets[rand.Intn(len(dm.textSnippets))],
			}
		case 1:
			// Audio Meter
			state.Processors = &rwp.Processors{
				AudioMeter: &rwp.ProcAudioMeter{
					W:     uint32(disp.W),
					H:     uint32(disp.H),
					Title: dm.textSnippets[rand.Intn(len(dm.textSnippets))],
					Mono:  false,
					Data1: uint32(rand.Intn(1001)),
					Peak1: uint32(rand.Intn(1001)),
					Data2: uint32(rand.Intn(1001)),
					Peak2: uint32(rand.Intn(1001)),
				},
			}
		case 2:
			// UniText Processor
			state.Processors = &rwp.Processors{
				UniText: &rwp.ProcUniText{
					W:              uint32(disp.W),
					H:              uint32(disp.H),
					Title:          maybeRandomText(dm.textSnippetsUTF8),
					Textline1:      maybeRandomText(dm.textSnippetsUTF8),
					Textline2:      maybeRandomText(dm.textSnippetsUTF8),
					Textline3:      maybeRandomText(dm.textSnippetsUTF8),
					Textline4:      maybeRandomText(dm.textSnippetsUTF8),
					SolidHeaderBar: rand.Intn(2) == 1,
					Align:          rwp.ProcUniText_AlignTypeE(rand.Intn(3)),
				},
			}

		case 3:
			// Static PNG image using processor
			img, err := createTestImage(disp.W, disp.H, disp.Type, fmt.Sprintf("%dx%d", disp.W, disp.H))
			if err == nil {
				var buf bytes.Buffer
				if err := png.Encode(&buf, img); err == nil {
					imageData := buf.Bytes()
					encoding := rwp.ProcGfxConverter_MONO
					switch disp.Type {
					case "color":
						encoding = rwp.ProcGfxConverter_RGB16bit
					case "gray":
						encoding = rwp.ProcGfxConverter_Gray4bit
					}
					state.Processors = &rwp.Processors{
						GfxConv: &rwp.ProcGfxConverter{
							W:         uint32(disp.W),
							H:         uint32(disp.H),
							ImageType: encoding,
							Scaling:   rwp.ProcGfxConverter_STRETCH,
							ImageData: imageData,
						},
					}
				}
			}
		case 4:
			// HWCGfx image directly (not using processors)
			inImg, err := createTestImage(disp.W, disp.H, disp.Type, fmt.Sprintf("%d*%d", disp.W, disp.H))
			if err == nil {
				img := rwp.HWCGfx{
					W: uint32(disp.W),
					H: uint32(disp.H),
				}

				monoImg := monogfx.MonoImg{}
				monoImg.NewImage(int(img.W), int(img.H))

				switch disp.Type {
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

				imgBounds := rawpanelproc.ImageBounds{X: 0, Y: 0, W: int(img.W), H: int(img.H)}
				rawpanelproc.RenderImageOnCanvas(&img, inImg, imgBounds, "", "", "")
				state.HWCGfx = &img

			}
		case 5:

			encoding := rwp.ProcIcon_MONO
			switch disp.Type {
			case "color":
				encoding = rwp.ProcIcon_RGB16bit
			case "gray":
				encoding = rwp.ProcIcon_Gray4bit
			}
			// Icon Processor
			state.Processors = &rwp.Processors{
				Icon: &rwp.ProcIcon{
					W:         uint32(disp.W),
					H:         uint32(disp.H),
					GlyphCode: dm.iconCodes[rand.Intn(len(dm.iconCodes))],
					//GlyphSize: ,
					Background:       rand.Intn(2) == 0, // Randomly enable or disable background
					BackgroundRadius: uint32(min(disp.W, disp.H) / 4),
					IconType:         encoding,
				},
			}
		case 6:
			// Choose a random file
			files := []string{"400x400.png", "400x400.webp", "400x400.jpg", "400x400.gif"}
			randomFile := files[rand.Intn(len(files))]

			f, err := picturesFS.Open("pictures/" + randomFile)
			if err != nil {
				log.Fatal(err)
			}

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, f); err != nil {
				log.Fatal(err)
			}
			f.Close()

			imageData := buf.Bytes()
			encoding := rwp.ProcGfxConverter_MONO
			switch disp.Type {
			case "color":
				encoding = rwp.ProcGfxConverter_RGB16bit
			case "gray":
				encoding = rwp.ProcGfxConverter_Gray4bit
			}
			state.Processors = &rwp.Processors{
				GfxConv: &rwp.ProcGfxConverter{
					W:         uint32(disp.W),
					H:         uint32(disp.H),
					ImageType: encoding,
					Scaling:   rwp.ProcGfxConverter_FIT,
					ImageData: imageData,
				},
			}
		}
	}

	return state
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

func maybeRandomText(snippets []string) string {
	if rand.Intn(2) == 0 {
		return ""
	}
	return snippets[rand.Intn(len(snippets))]
}
