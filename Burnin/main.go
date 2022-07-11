/*
	Raw Panel / Blue Pill Burn-In tester
	Uses protobuf format internally
*/
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"

	"google.golang.org/protobuf/proto"
)

type BurninData struct {
	InitialCycle []BurninInitialCycleEvent `json:"initialCycle"`
	DisplayMap   map[string]int            `json:"displaymap"`
	OutputMap    map[string]int            `json:"outputmap"`
	Ignore       []string                  `json:"ignore"`
	Events       []BurninEvent             `json:"events"`
}
type BurninInitialCycleEvent struct {
	HWC    int
	Text   string `json:",omitempty"` // Text for display
	Delay  int    `json:",omitempty"` // Delay before next
	Output string `json:",omitempty"` // Output: On, Off, Dimmed
	Color  int    `json:",omitempty"` // LED Color number
}
type BurninEvent struct {
	HWC    int
	Type   string `json:"type,omitempty"`
	Action string `json:"action"`
	Edge   int    `json:"_edge,omitempty"`
}

func (burninData *BurninData) save(file string) {
	if file != "" {
		jsonRes, _ := json.MarshalIndent(burninData, "", "\t")

		err := ioutil.WriteFile(file, jsonRes, 0644)
		if err != nil {
			panic(fmt.Sprintf("ERROR: File %s could not be written\n", file))
		}
	}
}

// Panel centric view:
// Inbound TCP commands - from external system to SKAARHOJ panel
// Outbound TCP commands - from panel to external system
func connectToPanel(panelIPAndPort string, incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, binaryPanel bool, brightness int) {

	for {
		fmt.Println("Trying to connect to panel on " + panelIPAndPort + "...")
		c, err := net.Dial("tcp", panelIPAndPort)

		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
		} else {
			fmt.Println("Success - Connected to panel")

			incoming <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					Command: &rwp.Command{
						ActivatePanel:         true,
						SendPanelInfo:         true,
						SendBurninProfile:     true,
						ReportHWCavailability: true,
						PanelBrightness: &rwp.Brightness{
							LEDs:  uint32(brightness),
							OLEDs: uint32(brightness),
						},
					},
				},
			}

			quit := make(chan bool)
			go func() {
				for {
					select {
					case <-quit:
						close(quit)
						return
					case incomingMessages := <-incoming:

						//su.Debug(outboundMessages)
						if binaryPanel {
							for _, msg := range incomingMessages {
								pbdata, err := proto.Marshal(msg)
								log.Should(err)
								header := make([]byte, 4)                                  // Create a 4-bytes header
								binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
								pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
								//log.Println("System -> Panel: ", pbdata)
								_, err = c.Write(pbdata)
								log.Should(err)
							}
						} else {
							lines := helpers.InboundMessagesToRawPanelASCIIstrings(incomingMessages)

							for _, line := range lines {
								fmt.Println(string("System -> Panel: " + strings.TrimSpace(string(line))))
								c.Write([]byte(line + "\n"))
							}
						}
					}
				}
			}()

			if binaryPanel {
				for {
					c.SetReadDeadline(time.Time{}) // Reset deadline, waiting for header
					headerArray := make([]byte, 4)
					_, err := io.ReadFull(c, headerArray) // Read 4 header bytes
					if err != nil {
						log.Println("Binary: ", err)
						break
					} else {
						currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
						if currentPayloadLength < 500000 {
							payload := make([]byte, currentPayloadLength)
							c.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
							_, err := io.ReadFull(c, payload)
							if err != nil {
								log.Println(err)
								break
							} else {
								outcomingMessage := &rwp.OutboundMessage{}
								proto.Unmarshal(payload, outcomingMessage)
								outgoing <- []*rwp.OutboundMessage{outcomingMessage}
							}
						} else {
							log.Error(" Payload", currentPayloadLength, "exceed limit")
							break
						}
					}
				}
			} else {
				connectionReader := bufio.NewReader(c) // Define OUTSIDE the for loop
				for {
					netData, err := connectionReader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							fmt.Println("Panel: " + c.RemoteAddr().String() + " disconnected")
							time.Sleep(time.Second)
						} else {
							fmt.Println(err)
						}
						break
					} else {
						outgoing <- helpers.RawPanelASCIIstringsToOutboundMessages([]string{strings.TrimSpace(netData)})
					}
				}
			}

			quit <- true
			c.Close()
		}
	}
}

func feedbackBinary(incoming chan []*rwp.InboundMessage, Event *rwp.HWCEvent, displayHWC int, outputHWC int, failed bool) {

	// Text:
	edgeString := ""
	color := 0
	switch Event.Binary.Edge {
	case 1:
		edgeString = "^"
		color = 15
	case 2:
		edgeString = "<"
		color = 8
	case 4:
		edgeString = "_"
		color = 2
	case 8:
		edgeString = ">"
		color = 11
	}
	txt := rwp.HWCText{}
	txt.Formatting = 7
	txt.Title = fmt.Sprintf("HWc #%d", Event.HWCID)
	txt.SolidHeaderBar = int(Event.HWCID) == displayHWC
	txt.Textline1 = fmt.Sprintf("%s %s", su.Qstr(Event.Binary.Pressed, "Dwn", "Up"), edgeString)
	txt.Inverted = failed

	incoming <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs:  []uint32{uint32(displayHWC)},
					HWCText: &txt,
				},
			},
		},
	}

	if Event.Binary.Pressed {
		incoming <- []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: []uint32{uint32(outputHWC)},
						HWCMode: &rwp.HWCMode{
							State: rwp.HWCMode_ON,
						},
						HWCColor: &rwp.HWCColor{
							ColorIndex: &rwp.ColorIndex{
								Index: rwp.ColorIndex_Colors(su.Qint(failed, 4, color)),
							},
						},
					},
				},
			},
		}
	} else {
		incoming <- []*rwp.InboundMessage{
			&rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: []uint32{uint32(outputHWC)},
						HWCMode: &rwp.HWCMode{
							State: rwp.HWCMode_DIMMED,
						},
					},
				},
			},
		}
	}
}

func feedbackEncoder(incoming chan []*rwp.InboundMessage, Event *rwp.HWCEvent, displayHWC int, outputHWC int, failed bool, encPrevPulses int) {

	// Text:
	txt := rwp.HWCText{}
	txt.Formatting = 7
	txt.Title = fmt.Sprintf("HWc #%d", Event.HWCID)
	txt.SolidHeaderBar = int(Event.HWCID) == displayHWC
	if encPrevPulses > 0 {
		txt.Textline1 = fmt.Sprintf("%d >", encPrevPulses)
	} else {
		txt.Textline1 = fmt.Sprintf("< %d", encPrevPulses)
	}
	txt.Inverted = failed

	incoming <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs:  []uint32{uint32(displayHWC)},
					HWCText: &txt,
				},
			},
		},
	}
	incoming <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs: []uint32{uint32(outputHWC)},
					HWCMode: &rwp.HWCMode{
						State: rwp.HWCMode_ON,
					},
					HWCColor: &rwp.HWCColor{
						ColorIndex: &rwp.ColorIndex{
							Index: rwp.ColorIndex_Colors(su.Qint(failed, 4, su.Qint(encPrevPulses < 0, 8, 15))),
						},
					},
				},
			},
		},
	}
}

func feedbackValue(incoming chan []*rwp.InboundMessage, Event *rwp.HWCEvent, displayHWC int, outputHWC int, failed bool, prevDirection int, position int) {

	// Text:
	txt := rwp.HWCText{}
	txt.Formatting = 7
	txt.Title = fmt.Sprintf("HWc #%d", Event.HWCID)
	txt.SolidHeaderBar = int(Event.HWCID) == displayHWC
	if prevDirection > 0 {
		txt.Textline1 = fmt.Sprintf("%d^", position)
	} else {
		txt.Textline1 = fmt.Sprintf("%d_", position)
	}
	txt.Inverted = failed

	incoming <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs:  []uint32{uint32(displayHWC)},
					HWCText: &txt,
				},
			},
		},
	}
	incoming <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs: []uint32{uint32(outputHWC)},
					HWCMode: &rwp.HWCMode{
						State: rwp.HWCMode_StateE(su.Qint(prevDirection > 0, int(rwp.HWCMode_ON), int(rwp.HWCMode_DIMMED))),
					},
					HWCColor: &rwp.HWCColor{
						ColorIndex: &rwp.ColorIndex{
							Index: rwp.ColorIndex_Colors(su.Qint(failed, 4, su.Qint(prevDirection < 0, 8, 15))),
						},
					},
				},
			},
		},
	}
}

func outputCycle(incoming chan []*rwp.InboundMessage, HWCavailabilityMap map[int]bool, initialOutputCycle int) {
	flashTimer := time.NewTimer(time.Second * 2)
	go func() {
		<-flashTimer.C

		colorIndex := []int{4, 15, 11, 2} // R-G-B-W

		keys := make([]int, 0, len(HWCavailabilityMap))
		for k := range HWCavailabilityMap {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		fmt.Println(keys)
		for _, availableKey := range keys {
			if !HWCavailabilityMap[availableKey] {
				continue
			}
			fmt.Printf("Output for HWC#%d...\n", availableKey)
			for a := 0; a < 4; a++ {
				txt := rwp.HWCText{}
				txt.Formatting = 7
				txt.Inverted = (a % 2) > 0
				txt.SolidHeaderBar = true
				txt.Title = fmt.Sprintf("HWC#%d", availableKey)
				txt.Textline1 = fmt.Sprintf("#%d", availableKey)

				incoming <- []*rwp.InboundMessage{
					&rwp.InboundMessage{
						States: []*rwp.HWCState{
							&rwp.HWCState{
								HWCIDs:  []uint32{uint32(availableKey)},
								HWCText: &txt,
								HWCMode: &rwp.HWCMode{
									State: rwp.HWCMode_ON,
								},
								HWCColor: &rwp.HWCColor{
									ColorIndex: &rwp.ColorIndex{
										Index: rwp.ColorIndex_Colors(colorIndex[a]),
									},
								},
							},
						},
					},
				}
				time.Sleep(time.Millisecond * time.Duration(initialOutputCycle))
			}
		}
	}()
}

func testInit(incoming chan []*rwp.InboundMessage, HWCavailabilityMap map[int]bool, initialCycles []BurninInitialCycleEvent, initialFlashes int) {
	flashTimer := time.NewTimer(time.Second * 2)
	go func() {
		<-flashTimer.C

		HWCkeys := []uint32{}
		for availableKey, _ := range HWCavailabilityMap {
			txt := rwp.HWCText{}
			txt.Formatting = 7
			txt.Inverted = true

			HWCkeys = append(HWCkeys, uint32(availableKey))

			incoming <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					States: []*rwp.HWCState{
						&rwp.HWCState{
							HWCIDs:  []uint32{uint32(availableKey)},
							HWCText: &txt,
							HWCMode: &rwp.HWCMode{
								State: rwp.HWCMode_ON,
							},
							HWCColor: &rwp.HWCColor{
								ColorIndex: &rwp.ColorIndex{
									Index: rwp.ColorIndex_Colors(2),
								},
							},
						},
					},
				},
			}

			time.Sleep(time.Millisecond * 25)
		}
		time.Sleep(time.Second * 3)

		for a := 0; a < initialFlashes*2+1; a++ {
			time.Sleep(time.Millisecond * 100)

			txt := rwp.HWCText{}
			txt.Formatting = 7
			//txt.Textline1 = strconv.Itoa(a)
			txt.Inverted = a%2 != 0
			_ = txt

			//HWCkeys = []uint32{1, 2, 3, 4, 5, 6, 7, 8, 17, 18, 19, 20, 21, 22, 23, 24}
			//HWCkeys = []uint32{9, 10, 11, 12, 25, 26, 27, 28}
			//HWCkeys = []uint32{13, 14, 15, 16, 29, 30, 31, 32}
			/*
				HWCkeys = []uint32{}
				for b := 0; b < 16; b += 4 {
					HWCkeys = append(HWCkeys, uint32(b+1))
				}
			*/
			incoming <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					States: []*rwp.HWCState{
						&rwp.HWCState{
							HWCIDs:  HWCkeys,
							HWCText: &txt,
							HWCMode: &rwp.HWCMode{
								State: rwp.HWCMode_StateE(su.Qint(a%2 != 0, int(rwp.HWCMode_ON), su.Qint(a == initialFlashes*2, int(rwp.HWCMode_DIMMED), int(rwp.HWCMode_OFF)))),
							},
						},
					},
				},
			}
		}

		//
		for _, cycleEvent := range initialCycles {

			state := rwp.HWCState{
				HWCIDs: []uint32{uint32(cycleEvent.HWC)},
			}

			if cycleEvent.Text != "" {
				txt := rwp.HWCText{}
				txt.Formatting = 7
				txt.Textline1 = cycleEvent.Text
				state.HWCText = &txt
			}

			if cycleEvent.Color > 0 {
				color := rwp.HWCColor{
					ColorIndex: &rwp.ColorIndex{
						Index: rwp.ColorIndex_Colors(cycleEvent.Color),
					},
				}
				state.HWCColor = &color
			}

			if cycleEvent.Output != "" {
				mode := rwp.HWCMode{
					State: rwp.HWCMode_StateE(su.Qint(cycleEvent.Output == "On", int(rwp.HWCMode_ON), su.Qint(cycleEvent.Output == "Dimmed", int(rwp.HWCMode_DIMMED), int(rwp.HWCMode_OFF)))),
				}
				state.HWCMode = &mode
			}

			incoming <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					States: []*rwp.HWCState{
						&state,
					},
				},
			}

			time.Sleep(time.Millisecond * time.Duration(cycleEvent.Delay))
		}

	}()
}

func testDone(incoming chan []*rwp.InboundMessage, HWCavailabilityMap map[int]bool, failuresInTheProcess bool) {
	flashTimer := time.NewTimer(time.Second * 1)
	go func() {
		<-flashTimer.C

		HWCkeys := []uint32{}
		for availableKey, _ := range HWCavailabilityMap {
			HWCkeys = append(HWCkeys, uint32(availableKey))
		}

		for a := 0; a < 4; a++ {
			time.Sleep(time.Millisecond * 500)
			incoming <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					States: []*rwp.HWCState{
						&rwp.HWCState{
							HWCIDs: HWCkeys,
							HWCMode: &rwp.HWCMode{
								State: rwp.HWCMode_StateE(su.Qint(a%2 == 0, int(rwp.HWCMode_ON), int(rwp.HWCMode_DIMMED))),
							},
							HWCColor: &rwp.HWCColor{
								ColorIndex: &rwp.ColorIndex{
									Index: rwp.ColorIndex_Colors(su.Qint(a%2 == 0, su.Qint(failuresInTheProcess, 8, 15), 2)),
								},
							},
						},
					},
				},
			}
		}

	}()
}

func testManager(incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, initialFlashes int, file string, recordMode bool, initialOutputCycle int) {

	HWCavailabilityMap := make(map[int]bool)

	// Parse the Burn-in data file:
	var burningData BurninData
	if !recordMode && file != "" {
		fmt.Println("Reading Burnin-json profile from file: " + file)
		jsonFile, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		defer jsonFile.Close()
		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &burningData)
		su.Debug(burningData)
	}

	currentTestIndex := 0

	encPrevPulses := 0
	prevDirection := 0
	prevPosition := 0
	prevHWC := 0
	lockedHWC := false
	newEntry := false

	first := true
	testIsDone := false
	failuresInTheProcess := false
	for {
		select {
		case outboundMessages := <-outgoing:
			// First, print the lines coming in as ASCII:
			lines := helpers.OutboundMessagesToRawPanelASCIIstrings(outboundMessages)
			for _, line := range lines {
				fmt.Println(string("Panel -> System: " + strings.TrimSpace(string(line))))
			}

			// Next, do some processing on it:
			for _, msg := range outboundMessages {

				// Picking up availability information (map command)
				if msg.HWCavailability != nil {
					for HWCid, available := range msg.HWCavailability {
						if available > 0 {
							HWCavailabilityMap[int(HWCid)] = true
						} else {
							delete(HWCavailabilityMap, int(HWCid))
						}
					}
				}

				if msg.BurninProfile != nil && file == "" {
					fmt.Println("Got Burnin-json profile from panel!")
					json.Unmarshal([]byte(msg.BurninProfile.Json), &burningData)
					//su.Debug(burningData)

					if first {
						first = false
						if initialOutputCycle > 0 {
							outputCycle(incoming, HWCavailabilityMap, initialOutputCycle)
						} else if initialFlashes > 0 {
							testInit(incoming, HWCavailabilityMap, burningData.InitialCycle, initialFlashes)
						}
					}
				}

				// Check button presses:
				if msg.Events != nil {
					for _, Event := range msg.Events {
						displayHWC, ok := burningData.DisplayMap[fmt.Sprintf("HWc#%d", Event.HWCID)]
						if !ok {
							displayHWC = int(Event.HWCID)
						}
						outputHWC, ok := burningData.OutputMap[fmt.Sprintf("HWc#%d", Event.HWCID)]
						if !ok {
							outputHWC = int(Event.HWCID)
						}
						failed := false

						//su.Debug(Event)
						if Event.Binary != nil {
							prevHWC = 0

							incomingEvent := BurninEvent{
								HWC:    int(Event.HWCID),
								Type:   "binary",
								Action: su.Qstr(Event.Binary.Pressed, "Down", "Up"),
								Edge:   su.Qint(Event.Binary.Edge == 0, 4, su.Qint(Event.Binary.Edge == 16, 4, int(Event.Binary.Edge))),
							}
							if recordMode {
								burningData.Events = append(burningData.Events, incomingEvent)
								burningData.save(file)
								fmt.Println("Recorded event (stop with ctrl+c):")
								su.Debug(incomingEvent)
							} else {
								if currentTestIndex >= len(burningData.Events) {
									fmt.Println("Received:")
									su.Debug(Event)
								} else if reflect.DeepEqual(burningData.Events[currentTestIndex], incomingEvent) {
									fmt.Printf("OK index %d (%d left)\n", currentTestIndex, len(burningData.Events)-currentTestIndex-1)
									currentTestIndex++
								} else {
									failed = true
									failuresInTheProcess = true
									fmt.Println("ERROR: No match!")
									fmt.Println("Received:")
									su.Debug(incomingEvent)
									fmt.Println("Expected:")
									su.Debug(burningData.Events[currentTestIndex])
								}
							}
							feedbackBinary(incoming, Event, displayHWC, outputHWC, failed)
						}
						if Event.Pulsed != nil {

							newEntry = false
							if !(int(Event.HWCID) == prevHWC && int(Event.Pulsed.Value)*encPrevPulses > 0) {
								encPrevPulses = int(Event.Pulsed.Value)
								prevHWC = int(Event.HWCID)
								newEntry = true
							} else {
								encPrevPulses += int(Event.Pulsed.Value)
							}

							if newEntry {
								incomingEvent := BurninEvent{
									HWC:    int(Event.HWCID),
									Type:   "pulsed",
									Action: "Enc",
									Edge:   su.Qint(Event.Pulsed.Value > 0, 1, -1),
								}
								if recordMode {
									burningData.Events = append(burningData.Events, incomingEvent)
									burningData.save(file)
									fmt.Println("Recorded event (stop with ctrl+c):")
									su.Debug(incomingEvent)
								} else {
									if currentTestIndex >= len(burningData.Events) {
										fmt.Println("Received:")
										su.Debug(Event)
									} else if reflect.DeepEqual(burningData.Events[currentTestIndex], incomingEvent) {
										fmt.Printf("OK index %d (%d left)\n", currentTestIndex, len(burningData.Events)-currentTestIndex-1)
										currentTestIndex++
									} else {
										failed = true
										failuresInTheProcess = true
										fmt.Println("ERROR: No match!")
										fmt.Println("Received:")
										su.Debug(incomingEvent)
										fmt.Println("Expected:")
										su.Debug(burningData.Events[currentTestIndex])
									}
								}
							}
							feedbackEncoder(incoming, Event, displayHWC, outputHWC, failed, encPrevPulses)
						}
						if Event.Absolute != nil {
							if int(Event.HWCID) != prevHWC {
								prevHWC = int(Event.HWCID)
								prevPosition = int(Event.Absolute.Value)
								prevDirection = 0
							} else {
								newEntry = false
								if !((int(Event.Absolute.Value)-prevPosition)*prevDirection > 0) {
									newEntry = true
								}
								prevDirection = int(Event.Absolute.Value) - prevPosition
								prevPosition = int(Event.Absolute.Value)

								if newEntry {
									incomingEvent := BurninEvent{
										HWC:    int(Event.HWCID),
										Type:   "absolute",
										Action: "",
										Edge:   su.Qint(prevDirection > 0, 1, -1),
									}
									if recordMode {
										burningData.Events = append(burningData.Events, incomingEvent)
										burningData.save(file)
										fmt.Println("Recorded event (stop with ctrl+c):")
										su.Debug(incomingEvent)
									} else {
										if currentTestIndex >= len(burningData.Events) {
											fmt.Println("Received:")
											su.Debug(Event)
										} else if reflect.DeepEqual(burningData.Events[currentTestIndex], incomingEvent) {
											fmt.Printf("OK index %d (%d left)\n", currentTestIndex, len(burningData.Events)-currentTestIndex-1)
											currentTestIndex++
										} else {
											failed = true
											failuresInTheProcess = true
											fmt.Println("ERROR: No match!")
											fmt.Println("Received:")
											su.Debug(incomingEvent)
											fmt.Println("Expected:")
											su.Debug(burningData.Events[currentTestIndex])
										}
									}
								}
								feedbackValue(incoming, Event, displayHWC, outputHWC, failed, prevDirection, prevPosition)
							}

						}
						if Event.Speed != nil {
							if lockedHWC {
								if int(Event.HWCID) == prevHWC {
									if Event.Speed.Value == 0 {
										lockedHWC = false
										prevHWC = 0
									}

									if newEntry {
										newEntry = false
										incomingEvent := BurninEvent{
											HWC:    int(Event.HWCID),
											Type:   "speed",
											Action: "",
											Edge:   su.Qint(prevDirection > 0, 1, -1),
										}
										if recordMode {
											burningData.Events = append(burningData.Events, incomingEvent)
											burningData.save(file)
											fmt.Println("Recorded event (stop with ctrl+c):")
											su.Debug(incomingEvent)
										} else {
											if currentTestIndex >= len(burningData.Events) {
												fmt.Println("Received:")
												su.Debug(Event)
											} else if reflect.DeepEqual(burningData.Events[currentTestIndex], incomingEvent) {
												fmt.Printf("OK index %d (%d left)\n", currentTestIndex, len(burningData.Events)-currentTestIndex-1)
												currentTestIndex++
											} else {
												failed = true
												failuresInTheProcess = true
												fmt.Println("ERROR: No match!")
												fmt.Println("Received:")
												su.Debug(incomingEvent)
												fmt.Println("Expected:")
												su.Debug(burningData.Events[currentTestIndex])
											}
										}
									}
									feedbackValue(incoming, Event, displayHWC, outputHWC, failed, prevDirection, int(Event.Speed.Value))
								}
							} else {
								if int(Event.HWCID) != prevHWC && Event.Speed.Value != 0 {
									prevHWC = int(Event.HWCID)
									prevDirection = int(Event.Speed.Value)
									lockedHWC = true
									newEntry = true
								}
							}
						}

						if !recordMode && currentTestIndex == len(burningData.Events) && !testIsDone {
							testIsDone = true
							testDone(incoming, HWCavailabilityMap, failuresInTheProcess)
						}
					}
				}
			}
		}
	}
}

func main() {
	// Setting up and parsing command line parameters
	initialFlashes := flag.Int("initialFlashes", 2, "Number of initial white full power on flashing of the panel LEDs and displays. Default is 2")
	initialOutputCycle := flag.Int("initialOutputCycle", 0, "If >0 this will cycle through all HWcs, identify with display label and set color to R-G-B-W with a delay of this value in milliseconds")
	brightness := flag.Int("brightness", 8, "Sets the brightness, value from 0-8 (8 is default)")
	file := flag.String("file", "", "File to read burnin data from. By default it will be fetched from the panel, if possible")
	record := flag.Bool("record", false, "Will record to the file instead of reading from it")
	binPanel := flag.Bool("binPanel", false, "Connects to the panels in binary mode")
	flag.Parse()

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: Burnin [options] [panelIP:port]...")
		fmt.Println("help:  Burnin -h")
		fmt.Println("")
		return
	}

	// Welcome message!
	fmt.Println("Welcome to Raw Panel - Server Panel BurnIn test! Made by Kasper Skaarhoj (c) 2021-2022")

	for _, argument := range arguments {
		startTest(argument, initialFlashes, initialOutputCycle, brightness, file, record, binPanel)
	}
	select {}
}

func startTest(panelIPAndPort string, initialFlashes *int, initialOutputCycle *int, brightness *int, file *string, record *bool, binPanel *bool) {
	fmt.Println("Ready to test panel " + panelIPAndPort + "...\n")

	// Set up server:
	incoming := make(chan []*rwp.InboundMessage, 10)
	outgoing := make(chan []*rwp.OutboundMessage, 10)

	go connectToPanel(panelIPAndPort, incoming, outgoing, *binPanel, *brightness)
	go testManager(incoming, outgoing, *initialFlashes, *file, *record, *initialOutputCycle)

}
