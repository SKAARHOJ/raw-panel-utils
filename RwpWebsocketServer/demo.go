package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
	log "github.com/s00500/env_logger"
)

// DemoManager manages the demo functionality for this Raw Panel WebSocket server reference implementation.
// It handles the panel topology, color cycling, and event triggers to display messages on the panels.
// It runs a ticker to periodically update the panel states and listens for events to trigger specific actions.
type DemoManager struct {
	mu           sync.Mutex
	topology     *topology.Topology
	currentColor int
	paused       bool
	lastTrigger  time.Time
	triggerChan  chan *rwp.HWCEvent
	sendFunc     func(msg *rwp.InboundMessage)
	demoTicker   *time.Ticker
	colorBase    int
	textSnippets []string
	colorMax     int
	hwcList      []uint32
	resumeTimer  *time.Timer
}

// NewDemoManager creates a new DemoManager instance with the provided send function.
func NewDemoManager(sendFunc func(msg *rwp.InboundMessage)) *DemoManager {
	return &DemoManager{
		colorBase:    2,
		colorMax:     17,
		textSnippets: []string{"Hello", "World", "RawPanel", "SKAARHOJ", "Demo", "Trigger", "Event", "ColorCycle"},
		sendFunc:     sendFunc,
		triggerChan:  make(chan *rwp.HWCEvent, 100),
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
			if dm.paused || dm.topology == nil {
				dm.mu.Unlock()
				continue
			}

			if len(dm.hwcList) == 0 {
				dm.mu.Unlock()
				continue
			}

			color := dm.currentColor
			dm.currentColor++
			if dm.currentColor > dm.colorMax {
				dm.currentColor = dm.colorBase
			}

			hwc := dm.hwcList[hwcIndex]
			hwcIndex = (hwcIndex + 1) % len(dm.hwcList) // Loop over list

			msg := &rwp.InboundMessage{
				States: []*rwp.HWCState{
					{
						HWCIDs: []uint32{hwc},
						HWCMode: &rwp.HWCMode{
							State: rwp.HWCMode_ON,
						},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: rwp.ColorIndex_Colors(color),
							},
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      dm.textSnippets[rand.Intn(len(dm.textSnippets))],
							Textline1:  dm.textSnippets[rand.Intn(len(dm.textSnippets))],
						},
					},
				},
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
		}
	}
	log.Println("DemoManager: Topology loaded with", len(dm.hwcList), "HWCs")
}

// OnEvent processes incoming hardware events, triggering the demo functionality.
func (dm *DemoManager) OnEvent(evt *rwp.HWCEvent) {
	dm.triggerChan <- evt
}
