package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
)

var demoHWCids atomic.Value
var demoRunning atomic.Bool
var demoStep atomic.Uint32
var demoSpeed atomic.Uint32
var demoCancel *context.CancelFunc

var demoSize = uint32(88)

func startDemo(hwcs []uint32) {
	if demoRunning.Load() {
		// Decrease speed on every re-click on demo
		if demoSpeed.Load() > 125 {
			demoSpeed.Store(demoSpeed.Load() / 2)
			log.Printf("Demo period: %dms\n", demoSpeed.Load())
		}
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		demoCancel = &cancel
		demoHWCids.Store(hwcs)
		demoSpeed.Store(2000)

		demoStep.Store(0)
		stepDemo(demoStep.Load())
		demoRunning.Store(true)

		go func() {

			timer := time.NewTimer(time.Millisecond * time.Duration(demoSpeed.Load()))
			for {
				select {
				case <-ctx.Done():
					timer.Stop()
					demoRunning.Store(false)
					demoCancel = nil
					return
				case <-timer.C:
					timer.Reset(time.Millisecond * time.Duration(demoSpeed.Load()))
					demoStep.Store((demoStep.Load() + 1) % demoSize)
					stepDemo(demoStep.Load())
				}
			}
		}()
	}
}

func stopDemo() {
	if demoRunning.Load() {
		if demoCancel != nil {
			(*demoCancel)()
		}
	}
}

func stepForward() {
	if demoCancel != nil {
		(*demoCancel)()
	}
	demoStep.Store((demoStep.Load() + 1) % demoSize)
	stepDemo(demoStep.Load())
}

func stepBackward() {
	if demoCancel != nil {
		(*demoCancel)()
	}
	demoStep.Store((demoSize + demoStep.Load() - 1) % demoSize)
	stepDemo(demoStep.Load())
}

func stepDemo(step uint32) {

	incomingMessages := []*rwp.InboundMessage{}
	StepDescription := fmt.Sprintf("Step %d", step)

	switch {
	case step == 0: // Start
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 4,
						},
						HWCExtended: &rwp.HWCExtended{},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: 0,
							},
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Textline1:  "DEMO!",
							Inverted:   true,
						},
					},
				},
			},
		}
	case step == 1: // Off
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 0,
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      fmt.Sprintf("Demo %d", step),
							Textline1:  "(Off)",
						},
					},
				},
			},
		}
	case step == 2: // Dimmed
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 5,
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      fmt.Sprintf("Demo %d", step),
							Textline1:  "Dimmed",
						},
					},
				},
			},
		}
	case step == 3: // Amber
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 1,
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      fmt.Sprintf("Demo %d", step),
							Textline1:  "Amber(1)",
						},
					},
				},
			},
		}
	case step == 4: // Red
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 2,
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      fmt.Sprintf("Demo %d", step),
							Textline1:  "Red(1)",
						},
					},
				},
			},
		}
	case step == 5: // Green
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 3,
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      fmt.Sprintf("Demo %d", step),
							Textline1:  "Green(3)",
						},
					},
				},
			},
		}
	case step >= 6 && step < 6+17: // On
		colorNames := []string{"Default", "Off", "White", "Warm", "Red", "Rose", "Pink", "Purple", "Amber", "Yellow", "Dark Blue", "Blue", "Ice", "Cyan", "Spring", "Green", "Mint"}
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 4,
						},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: rwp.ColorIndex_Colors(step - 6),
							},
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      fmt.Sprintf("Demo %d", step),
							Textline1:  "On Color:",
							Textline2:  colorNames[step-6],
							PairMode:   1,
						},
					},
				},
			},
		}
	case step >= 23 && step < 23+15: // Strength
		StrengthValue := uint32(1000 * (step - 23) / 14)

		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 0,
						},
						HWCExtended: &rwp.HWCExtended{
							Interpretation: 1,
							Value:          StrengthValue,
						},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: rwp.ColorIndex_Colors(step - 23 + 2),
							},
						},
						HWCText: &rwp.HWCText{
							Formatting:     7,
							SolidHeaderBar: true,
							Title:          fmt.Sprintf("Ext. %d", step),
							Textline1:      "Strength",
							Textline2:      fmt.Sprintf("%d", StrengthValue),
							PairMode:       1,
						},
					},
				},
			},
		}
	case step >= 38 && step < 38+16: // Step
		StepValue := uint32(step - 38)

		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 0,
						},
						HWCExtended: &rwp.HWCExtended{
							Interpretation: 3,
							Value:          StepValue,
						},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: rwp.ColorIndex_Colors(((step - 38) % 15) + 2),
							},
						},
						HWCText: &rwp.HWCText{
							Formatting:     7,
							SolidHeaderBar: true,
							Title:          fmt.Sprintf("Ext. %d", step),
							Textline1:      "Step",
							Textline2:      fmt.Sprintf("%d", StepValue),
							PairMode:       1,
						},
					},
				},
			},
		}
	case step >= 54 && step < 54+21: // VU
		VUValue := uint32(1000 * (step - 54) / 20)

		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 0,
						},
						HWCExtended: &rwp.HWCExtended{
							Interpretation: 4,
							Value:          VUValue,
						},
						HWCText: &rwp.HWCText{
							Formatting:     7,
							SolidHeaderBar: true,
							Title:          fmt.Sprintf("Ext. %d", step),
							Textline1:      "VU",
							Textline2:      fmt.Sprintf("%d", VUValue),
							PairMode:       1,
						},
					},
				},
			},
		}
	case step >= 75 && step < 75+11: // Position
		PosValue := uint32(1000 * (step - 75) / 10)

		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State: 0,
						},
						HWCExtended: &rwp.HWCExtended{
							Interpretation: 5,
							Value:          PosValue,
						},
						HWCText: &rwp.HWCText{
							Formatting:     7,
							SolidHeaderBar: true,
							Title:          fmt.Sprintf("Ext. %d", step),
							Textline1:      "Position",
							Textline2:      fmt.Sprintf("%d", PosValue),
							PairMode:       1,
						},
					},
				},
			},
		}
	case step == 86: // Off
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State:  0,
							Output: true,
						},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: 0,
							},
						},
						HWCText: &rwp.HWCText{
							Formatting: 7,
							Title:      fmt.Sprintf("Demo %d", step),
							Textline1:  "(Off)",
							Textline2:  "Relay ON",
							PairMode:   1,
						},
					},
				},
			},
		}

	case step == 87: // Blinking
		incomingMessages = []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: demoHWCids.Load().([]uint32),
						HWCMode: &rwp.HWCMode{
							State:        5,
							BlinkPattern: 2,
						},
						HWCExtended: &rwp.HWCExtended{},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: 0,
							},
						},
						HWCText: &rwp.HWCText{
							Formatting:     7,
							Title:          "Demo Is Finished!",
							SolidHeaderBar: true,
							Textline1:      "Blinking...",
						},
					},
				},
			},
		}

	}

	stateAsJsonString, _ := json.Marshal(incomingMessages[0].States[0])

	pbdata, err := proto.Marshal(incomingMessages[0])
	log.Should(err)
	header := make([]byte, 4)                                  // Create a 4-bytes header
	binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
	pbdata = append(header, pbdata...)

	wsslice.Iter(func(w *wsclient) {
		w.msgToClient <- &wsToClient{
			RWPASCIIToPanel:    strings.Join(helpers.InboundMessagesToRawPanelASCIIstrings(incomingMessages), "\n"),
			RWPJSONToPanel:     string(stateAsJsonString),
			RWPProtobufToPanel: prettyHexPrint(pbdata),
			StepDescription:    StepDescription,
		}
	})

	incoming <- incomingMessages
}
