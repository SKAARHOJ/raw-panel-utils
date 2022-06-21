/*
	Raw Panel Test Application

	Cycles button colors and display content on all hardware components
	The cycle happens as buttons are pressed or automatically after some seconds of idle time.

	Distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
	without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
	PARTICULAR PURPOSE. MIT License
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"

	"encoding/binary"
	"encoding/json"

	"google.golang.org/protobuf/proto"
)

// TODO: Import these definitions from somewhere else... (so it is shared)
type Topology struct {
	Title     string `json:"title,omitempty"` // Controller Title
	HWc       []TopologyHWcomponent
	TypeIndex map[uint32]TopologyHWcTypeDef `json:"typeIndex"`
}
type TopologyHWcomponent struct {
	Id           uint32             `json:"id"`   // The HWCid - follows the index (+1) of the $HWc
	X            int                `json:"x"`    // x coordinate (1/10th mm) - index 1 of the entries in $HWc
	Y            int                `json:"y"`    // y coordinate (1/10th mm) - index 2 of the entries in $HWc
	Txt          string             `json:"txt"`  // text label, possibly split in two lines by "|"
	Type         uint32             `json:"type"` // Type number, must be a key in $subElements (generateTopologies.phpsh) and thereby a key in the TypeIndex map. Type 0 (zero) means disabled.
	TypeOverride TopologyHWcTypeDef `json:"typeOverride"`
}

// See DC_SKAARHOJ_RawPanel.odt for descriptions:
type TopologyHWcTypeDef struct {
	W      int                        `json:"w,omitempty"`      // Width of component
	H      int                        `json:"h,omitempty"`      // Height of component. If defined, the component will be a rectangle, otherwise a circle with diameter W.
	Out    string                     `json:"out,omitempty"`    // Output type
	In     string                     `json:"in,omitempty"`     // Input type
	Desc   string                     `json:"desc,omitempty"`   // Description
	Ext    string                     `json:"ext,omitempty"`    // Extended return value mode
	Subidx int                        `json:"subidx,omitempty"` // A reference to the index of an element in the "sub" element which has a "special" meaning. For analog (av, ah, ar) and intensity (iv, ih, ir) elements, this would be an element suggested for being used as a handle for a fader or joystick.
	Rotate float32                    `json:"rotate,omitempty"`
	Disp   TopologyHWcTypeDef_Display `json:"disp,omitempty"` // Display description
	Sub    []TopologyHWcTypeDefSubEl  `json:"sub,omitempty"`
}

func (topology *Topology) getTypeDefWithOverride(HWcDef *TopologyHWcomponent) TopologyHWcTypeDef {

	typeDef := topology.TypeIndex[HWcDef.Type]

	// Look for local type override and overlay it if it's there..:
	// Across controllers, this is largely alternative disp{} pixel dimensions and some sub[] changes.
	if fmt.Sprint(HWcDef.TypeOverride) != fmt.Sprint(TopologyHWcTypeDef{}) {
		if HWcDef.TypeOverride.W > 0 {
			typeDef.W = HWcDef.TypeOverride.W
		}
		if HWcDef.TypeOverride.H > 0 {
			typeDef.H = HWcDef.TypeOverride.H
		}
		if HWcDef.TypeOverride.Rotate != 0 {
			typeDef.Rotate = HWcDef.TypeOverride.Rotate
		}
		if HWcDef.TypeOverride.Out != "" {
			typeDef.Out = HWcDef.TypeOverride.Out
		}
		if HWcDef.TypeOverride.In != "" {
			typeDef.In = HWcDef.TypeOverride.In
		}
		if HWcDef.TypeOverride.Ext != "" {
			typeDef.Ext = HWcDef.TypeOverride.Ext
		}
		if HWcDef.TypeOverride.Subidx > 0 {
			typeDef.Subidx = HWcDef.TypeOverride.Subidx
		}
		if HWcDef.TypeOverride.Disp != (TopologyHWcTypeDef_Display{}) {
			typeDef.Disp = HWcDef.TypeOverride.Disp
		}
		if len(HWcDef.TypeOverride.Sub) > 0 {
			typeDef.Sub = HWcDef.TypeOverride.Sub
		}
	}

	return typeDef
}

type TopologyHWcTypeDefSubEl struct {
	ObjType string `json:"_,omitempty"`
	X       int    `json:"_x,omitempty"`
	Y       int    `json:"_y,omitempty"`
	W       int    `json:"_w,omitempty"`
	H       int    `json:"_h,omitempty"`
	R       int    `json:"r,omitempty"`
	Rx      int    `json:"rx,omitempty"`
	Ry      int    `json:"ry,omitempty"`
	Style   string `json:"style,omitempty"`
	Idx     int    `json:"_idx,omitempty"`
}
type TopologyHWcTypeDef_Display struct {
	W      int    `json:"w,omitempty"`      // Pixel width of display
	H      int    `json:"h,omitempty"`      // Pixel height of display
	Subidx int    `json:"subidx,omitempty"` // Index of the sub element which placeholds for the display area. -1 if no sub element is used for that
	Type   string `json:"type,omitempty"`   // Additional features of display. "color" for example.
}

// Panel centric view:
// Inbound TCP commands - from external system to SKAARHOJ panel
// Outbound TCP commands - from panel to external system
func connectToPanel(panelIPAndPort string, incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, binaryPanel bool, panelNum int, verboseIncoming *int, analogProfiling *bool, cpuProfiling *int, brightness *int, fullPowerStartUp *bool) {

	connected := false
	var c net.Conn
	var err error

	// This go routine will listen indefinitely on the incoming channel for Raw Panel messages to send to the connected panel.
	go func() {
		for {
			select {
			case incomingMessages := <-incoming:
				if connected {
					if *verboseIncoming > 1 {
						log.Println(log.Indent(incomingMessages))
					}
					if binaryPanel { // Writing Raw Panel Protobuf messages out to panel in Binary encoding:
						for _, msg := range incomingMessages {
							pbdata, err := proto.Marshal(msg)
							log.Should(err)
							header := make([]byte, 4)                                  // Create a 4-bytes header
							binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
							pbdata = append(header, pbdata...)                         // and concatenate it with the binary message

							if *verboseIncoming > 0 {
								log.Println("System -> Panel: ", pbdata)
							}
							_, err = c.Write(pbdata)
							log.Should(err)
						}
					} else { // Writing Raw Panel Protobuf messages out to panel in ASCII encoding:
						lines := helpers.InboundMessagesToRawPanelASCIIstrings(incomingMessages)
						for _, line := range lines {
							if *verboseIncoming > 0 {
								log.Println(string("System -> Panel: " + strings.TrimSpace(string(line))))
							}
							c.Write([]byte(line + "\n"))
						}
					}
				}
			}
		}
	}()

	// Here, we will connect to the panel and manage message from the panel to the connecting system:
	for {
		log.Printf("Trying to connect to panel %d on %s ...\n", panelNum, panelIPAndPort)
		c, err = net.Dial("tcp", panelIPAndPort)

		if err != nil {
			log.Should(err)
			time.Sleep(time.Second * 3) // Wait three seconds before a connection retry
		} else {
			log.Printf("Success - Connected to panel %d on %s ...\n", panelNum, panelIPAndPort)
			connected = true

			// Send a connection message to the panel:
			incoming <- []*rwp.InboundMessage{
				{
					Command: &rwp.Command{
						ActivatePanel:         true,
						SendPanelInfo:         true, // Asks panel to return info about itself
						ReportHWCavailability: true, // Asks panel to return mapping of hardware components
						SendPanelTopology:     true, // Asks panel to send its topology
						SetSleepTimeout: &rwp.SleepTimeout{ // Disable sleep
							Value: 0,
						},
						PanelBrightness: &rwp.Brightness{
							OLEDs: uint32(*brightness),
							LEDs:  uint32(*brightness),
						},
					},
				},
			}

			if *cpuProfiling >= 0 {
				log.Println("CPU Profiling enabled - check folder ColorDisplayButtonTest/ for log files")

				incoming <- []*rwp.InboundMessage{
					{
						Command: &rwp.Command{
							LoadCPU: &rwp.LoadCPU{
								Level: rwp.LoadCPU_LevelE(*cpuProfiling), // 0-4, 2 cores
							},
							PublishSystemStat: &rwp.PublishSystemStat{
								PeriodSec: 5,
							},
						},
					},
				}
			} else {
				incoming <- []*rwp.InboundMessage{
					{
						Command: &rwp.Command{
							LoadCPU: &rwp.LoadCPU{
								Level: rwp.LoadCPU_LevelE(0), // Disable
							},
							PublishSystemStat: &rwp.PublishSystemStat{
								PeriodSec: 0, // Disable
							},
						},
					},
				}
			}

			if *analogProfiling {
				log.Println("Analog Profiling enabled - check folder ColorDisplayButtonTest/ for log files")
			}
			allHWCs := []uint32{}
			for a := 0; a < 200; a++ {
				allHWCs = append(allHWCs, uint32(a+1))
			}
			incoming <- []*rwp.InboundMessage{
				{
					States: []*rwp.HWCState{
						&rwp.HWCState{
							HWCIDs: allHWCs,
							PublishRawADCValues: &rwp.PublishRawADCValues{
								Enabled: *analogProfiling,
							},
						},
					},
				},
			}

			// TEMPORARY because there is an issue with it! Remove when brightness works when set in first package (KS)
			go func() {
				time.Sleep(time.Second * 1)
				incoming <- []*rwp.InboundMessage{
					{
						Command: &rwp.Command{
							PanelBrightness: &rwp.Brightness{
								OLEDs: uint32(*brightness),
								LEDs:  uint32(*brightness),
							},
						},
					},
				}
			}()

			if *fullPowerStartUp {
				log.Println("Setting Full Power Startup")
				allHWCs := []uint32{}
				for a := 0; a < 200; a++ {
					allHWCs = append(allHWCs, uint32(a+1))
				}
				incoming <- []*rwp.InboundMessage{
					&rwp.InboundMessage{
						States: []*rwp.HWCState{
							&rwp.HWCState{
								HWCIDs: allHWCs,
								HWCMode: &rwp.HWCMode{
									State: rwp.HWCMode_ON,
								},
								HWCColor: &rwp.HWCColor{
									ColorIndex: &rwp.ColorIndex{
										Index: rwp.ColorIndex_Colors(2),
									},
								},
								HWCText: &rwp.HWCText{
									Inverted:   true,
									Formatting: 7,
									//Textline1:  "Inverted",
								},
							},
						},
					},
				}
			}

			if binaryPanel { // Reading from the panel, expecting binary protobud encoder content:
				for {
					c.SetReadDeadline(time.Time{}) // Reset deadline, waiting for header
					headerArray := make([]byte, 4)
					_, err := io.ReadFull(c, headerArray) // Read 4 header bytes
					if err != nil {
						log.Error("on binary panel read: ", err)
						break
					}
					currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
					if currentPayloadLength < 500000 {
						payload := make([]byte, currentPayloadLength)
						c.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
						_, err := io.ReadFull(c, payload)
						if err != nil {
							log.Errorln(err)
							break
						}
						outcomingMessage := &rwp.OutboundMessage{}
						//fmt.Println(payload)
						proto.Unmarshal(payload, outcomingMessage)
						outgoing <- []*rwp.OutboundMessage{outcomingMessage}
					} else {
						log.Errorln("Error: Payload", currentPayloadLength, "exceed limit")
						break
					}
				}
			} else { // Reading from the panel, expecting ASCII encoded content:
				err := c.SetReadDeadline(time.Time{})
				log.Should(err)

				connectionReader := bufio.NewReader(c)
				for {
					netData, err := connectionReader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							log.Printf("Panel %d: %s disconnected\n", panelNum, c.RemoteAddr().String())
							time.Sleep(time.Second)
						} else {
							log.Should(err)
						}
						break
					} else {
						outgoing <- helpers.RawPanelASCIIstringsToOutboundMessages([]string{strings.TrimSpace(netData)})
					}
				}
			}

			c.Close()
			connected = false
			time.Sleep(time.Second * 1)
		}
	}
}

var PanelName = make(map[int]string)
var PanelFaders = make(map[int][]uint32)

func testManager(incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, invertCallAll bool, panelNum int, autoInterval *int, exclusiveHWClist *string, demoModeDelay *int, verboseOutgoing *int, demoModeFaders *bool, demoModeImgsOnly *bool) {

	numberOfTextStrings := su.Qint(*demoModeImgsOnly, 0, len(HWCtextStrings))
	HWCavailabilityMap := make(map[int]bool)

	HWCcolor := make(map[int]int)
	HWCdispIndex := make(map[int]int)
	dispIndexCount := numberOfTextStrings + len(HWCgfxStrings)

	var lastTriggerTimeMU = &sync.Mutex{}
	lastTriggerTime := time.Now()
	exclusiveList := strings.Split(*exclusiveHWClist, ",")

	// Auto:
	go func() {
		ticker := time.NewTicker(time.Duration(*autoInterval) * time.Millisecond)
		autoDispIndex := 0
		autoColorIndex := 0
		autoHWCIndex := 0

		faderValue := 0
		faderIndex := 0

		for {
			select {
			case <-ticker.C:
				lastTriggerTimeMU.Lock()
				goAuto := time.Now().After(lastTriggerTime.Add(time.Duration(*demoModeDelay) * time.Second))
				lastTriggerTimeMU.Unlock()

				if goAuto && *demoModeDelay != 0 { // Running the demo cycle:
					activeHWCs := make([]uint32, 0)
					if invertCallAll {
						for k, isActive := range HWCavailabilityMap {
							if isActive && inList(k, exclusiveList) {
								activeHWCs = append(activeHWCs, uint32(k))
							}
						}
						// Rotate color:
						autoDispIndex = (autoDispIndex + 1) % dispIndexCount
						autoColorIndex = (autoColorIndex + 15 + 1) % 15
					} else {
						HWCslice := make([]int, 0)
						for k := range HWCavailabilityMap {
							HWCslice = append(HWCslice, k)
						}
						sort.Ints(HWCslice)
						for k := 0; k < len(HWCslice); k++ {
							index := (k + autoHWCIndex + 1) % len(HWCslice)
							HWCkey := HWCslice[index]
							if index == 0 {
								// Rotate color:
								autoColorIndex = (autoColorIndex + 15 + +1) % 15
							}

							if HWCavailabilityMap[HWCkey] && inList(HWCkey, exclusiveList) {
								activeHWCs = append(activeHWCs, uint32(HWCkey))
								autoHWCIndex = index
								break
							}
						}
						autoDispIndex = (autoDispIndex + 1) % dispIndexCount
					}

					var dispMsg []*rwp.InboundMessage
					if autoDispIndex < numberOfTextStrings { // Texts:
						dispMsg = helpers.RawPanelASCIIstringsToInboundMessages([]string{HWCtextStrings[autoDispIndex]})
					} else { // Images:
						dispMsg = helpers.RawPanelASCIIstringsToInboundMessages(HWCgfxStrings[autoDispIndex-numberOfTextStrings])
					}

					txt := &rwp.HWCText{}
					if len(dispMsg) > 0 && dispMsg[0].States[0].HWCText != nil {
						txt = dispMsg[0].States[0].HWCText
					} else {
						txt = nil
					}

					img := &rwp.HWCGfx{}
					if len(dispMsg) > 0 && dispMsg[0].States[0].HWCGfx != nil {
						img = dispMsg[0].States[0].HWCGfx
					} else {
						img = nil
					}

					incoming <- []*rwp.InboundMessage{
						&rwp.InboundMessage{
							States: []*rwp.HWCState{
								&rwp.HWCState{
									HWCIDs: activeHWCs,
									HWCMode: &rwp.HWCMode{
										State: rwp.HWCMode_ON,
									},
									HWCColor: &rwp.HWCColor{
										ColorIndex: &rwp.ColorIndex{
											Index: rwp.ColorIndex_Colors(autoColorIndex + 2),
										},
									},
									HWCText: txt,
									HWCGfx:  img,
								},
							},
						},
					}

					if *demoModeFaders {
						if _, exists := PanelFaders[panelNum]; exists {
							incoming <- []*rwp.InboundMessage{
								&rwp.InboundMessage{
									States: []*rwp.HWCState{
										&rwp.HWCState{
											HWCIDs: []uint32{uint32(PanelFaders[panelNum][faderIndex])},
											HWCExtended: &rwp.HWCExtended{
												Interpretation: rwp.HWCExtended_FADER,
												Value:          uint32(faderValue),
											},
										},
									},
								},
							}

							faderIndex = (faderIndex + 1) % len(PanelFaders[panelNum])
							faderValue = faderValue + 50
							if faderValue > 1000 {
								faderValue = 0
							}
						}
					}
				}
			}
		}
	}()

	for {
		select {
		case outboundMessages := <-outgoing:

			// First, print the lines coming in as ASCII:
			if *verboseOutgoing > 1 {
				log.Println(log.Indent(outboundMessages))
			}
			lines := helpers.OutboundMessagesToRawPanelASCIIstrings(outboundMessages)
			for _, line := range lines {
				if *verboseOutgoing > 0 || (strings.HasPrefix(strings.TrimSpace(line), "HWC") && !strings.Contains(strings.TrimSpace(line), "Raw:")) {
					log.Printf("Panel %d -> System: %s\n", panelNum, strings.TrimSpace(line))
				}
			}

			// Next, do some processing on it:
			for _, msg := range outboundMessages {

				if msg.PanelInfo != nil {
					PanelName[panelNum] = fmt.Sprintf("%d: %s - %s", panelNum, msg.PanelInfo.Model, msg.PanelInfo.Serial)
				}

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

				if msg.PanelTopology != nil {
					// Reading JSON topology:
					var topology Topology
					json.Unmarshal([]byte(msg.PanelTopology.Json), &topology)

					for _, HWcDef := range topology.HWc {
						typeDef := topology.getTypeDefWithOverride(&HWcDef)
						if typeDef.Ext == "pos" {
							if _, exists := PanelFaders[panelNum]; !exists {
								PanelFaders[panelNum] = []uint32{}
							}
							PanelFaders[panelNum] = append(PanelFaders[panelNum], HWcDef.Id)
						}
					}
				}

				// Check events:
				if msg.Events != nil {

					for _, Event := range msg.Events {

						activeHWCs := make([]uint32, 0)
						if (Event.Binary != nil && (Event.Binary.Edge == rwp.BinaryEvent_LEFT || Event.Binary.Edge == rwp.BinaryEvent_RIGHT)) != invertCallAll || (Event.Absolute != nil || Event.Speed != nil) {
							for k, isActive := range HWCavailabilityMap {
								if isActive && inList(k, exclusiveList) {
									activeHWCs = append(activeHWCs, uint32(k))
								}
							}
						} else {
							if inList(int(Event.HWCID), exclusiveList) {
								activeHWCs = append(activeHWCs, uint32(Event.HWCID))
							}
						}

						// Will only happen if analog profiling is enabled:
						if Event.SysStat != nil {
							procesSysStatValues(panelNum, Event)
							continue
						}

						// Will only happen if analog profiling is enabled:
						if Event.RawAnalog != nil {
							if Event.RawAnalog.Value <= 0xFFFF {
								procesRawAnalogValue(panelNum, Event)
								//fmt.Printf("Raw Analog From #%d: %d\n", Event.HWCID, Event.RawAnalog.Value)
							} else {
								log.Errorf("Raw Analog From #_p%d.%d: %d is WAY OFF! WTF?\n", panelNum, Event.HWCID, Event.RawAnalog.Value)
							}
							continue
						}

						if Event.Binary != nil {
							// Only register a trigger time if its not a raw analog value returned...
							lastTriggerTimeMU.Lock()
							lastTriggerTime = time.Now()
							lastTriggerTimeMU.Unlock()

							direction := su.Qint(Event.Binary.Edge == rwp.BinaryEvent_LEFT || Event.Binary.Edge == rwp.BinaryEvent_TOP, -1, 1)

							// Button presses:
							switch Event.Binary.Pressed {
							case true:
								// Rotate color:
								HWCdispIndex[int(Event.HWCID)] = (HWCdispIndex[int(Event.HWCID)] + dispIndexCount + direction) % dispIndexCount
								HWCcolor[int(Event.HWCID)] = (HWCcolor[int(Event.HWCID)] + 15 + direction) % 15

								var dispMsg []*rwp.InboundMessage
								if HWCdispIndex[int(Event.HWCID)] < numberOfTextStrings { // Texts:
									dispMsg = helpers.RawPanelASCIIstringsToInboundMessages([]string{HWCtextStrings[HWCdispIndex[int(Event.HWCID)]]})
								} else { // Images:
									dispMsg = helpers.RawPanelASCIIstringsToInboundMessages(HWCgfxStrings[HWCdispIndex[int(Event.HWCID)]-numberOfTextStrings])
								}

								txt := rwp.HWCText{}
								if len(dispMsg) > 0 && dispMsg[0].States[0].HWCText != nil {
									txt = *dispMsg[0].States[0].HWCText
									txt.Inverted = !txt.Inverted
								}
								img := rwp.HWCGfx{}
								if len(dispMsg) > 0 && dispMsg[0].States[0].HWCGfx != nil {
									img = *dispMsg[0].States[0].HWCGfx
								}

								incoming <- []*rwp.InboundMessage{
									&rwp.InboundMessage{
										States: []*rwp.HWCState{
											&rwp.HWCState{
												HWCIDs: activeHWCs,
												HWCMode: &rwp.HWCMode{
													State: rwp.HWCMode_ON,
												},
												HWCColor: &rwp.HWCColor{
													ColorIndex: &rwp.ColorIndex{
														Index: rwp.ColorIndex_Colors(HWCcolor[int(Event.HWCID)] + 2),
													},
												},
												HWCText: &txt,
												HWCGfx:  &img,
											},
										},
									},
								}
							default: // Release:
								incoming <- []*rwp.InboundMessage{
									&rwp.InboundMessage{
										States: []*rwp.HWCState{
											&rwp.HWCState{
												HWCIDs: activeHWCs,
												HWCMode: &rwp.HWCMode{
													State: rwp.HWCMode_DIMMED,
												},
											},
										},
									},
								}
							}
							//							su.Debug(Event)

						}
						if Event.Pulsed != nil {
							// Only register a trigger time if its not a raw analog value returned...
							lastTriggerTimeMU.Lock()
							lastTriggerTime = time.Now()
							lastTriggerTimeMU.Unlock()

							if Event.Pulsed.Value != 0 {
								// Rotate color:
								HWCdispIndex[int(Event.HWCID)] = int(math.Abs(float64(HWCdispIndex[int(Event.HWCID)]+dispIndexCount+int(Event.Pulsed.Value)))) % dispIndexCount
								HWCcolor[int(Event.HWCID)] = int(math.Abs(float64(HWCcolor[int(Event.HWCID)]+15+int(Event.Pulsed.Value)))) % 15

								var dispMsg []*rwp.InboundMessage
								if HWCdispIndex[int(Event.HWCID)] < numberOfTextStrings { // Texts:
									dispMsg = helpers.RawPanelASCIIstringsToInboundMessages([]string{HWCtextStrings[HWCdispIndex[int(Event.HWCID)]]})
								} else { // Images:
									dispMsg = helpers.RawPanelASCIIstringsToInboundMessages(HWCgfxStrings[HWCdispIndex[int(Event.HWCID)]-numberOfTextStrings])
								}

								txt := rwp.HWCText{}
								if len(dispMsg) > 0 && dispMsg[0].States[0].HWCText != nil {
									txt = *dispMsg[0].States[0].HWCText
								}
								img := rwp.HWCGfx{}
								if len(dispMsg) > 0 && dispMsg[0].States[0].HWCGfx != nil {
									img = *dispMsg[0].States[0].HWCGfx
								}

								incoming <- []*rwp.InboundMessage{
									&rwp.InboundMessage{
										States: []*rwp.HWCState{
											&rwp.HWCState{
												HWCIDs: activeHWCs,
												HWCMode: &rwp.HWCMode{
													State: rwp.HWCMode_ON,
												},
												HWCColor: &rwp.HWCColor{
													ColorIndex: &rwp.ColorIndex{
														Index: rwp.ColorIndex_Colors(HWCcolor[int(Event.HWCID)] + 2),
													},
												},
												HWCText: &txt,
												HWCGfx:  &img,
											},
										},
									},
								}
							}
						}
						if Event.Absolute != nil {
							// Only register a trigger time if its not a raw analog value returned...
							lastTriggerTimeMU.Lock()
							lastTriggerTime = time.Now()
							lastTriggerTimeMU.Unlock()

							// Rotate color:
							HWCdispIndex[int(Event.HWCID)] = (HWCdispIndex[int(Event.HWCID)] + dispIndexCount + int(Event.Absolute.Value)) % dispIndexCount
							HWCcolor[int(Event.HWCID)] = (HWCcolor[int(Event.HWCID)] + 15 + int(Event.Absolute.Value)) % 15

							txt := rwp.HWCText{}
							txt.Formatting = 7
							txt.Title = fmt.Sprintf("HWc #%d", Event.HWCID)
							txt.SolidHeaderBar = true
							txt.Textline1 = fmt.Sprintf("%v", Event.Absolute.Value)
							txt.Inverted = true

							incoming <- []*rwp.InboundMessage{
								&rwp.InboundMessage{
									States: []*rwp.HWCState{
										&rwp.HWCState{
											HWCIDs: activeHWCs,
											/*
												HWCMode: &rwp.HWCMode{
													State: rwp.HWCMode_ON,
												},
												HWCColor: &rwp.HWCColor{
													ColorIndex: &rwp.ColorIndex{
														Index: rwp.ColorIndex_Colors(HWCcolor[int(Event.HWCID)] + 2),
													},
												},*/
											HWCText: &txt,
										},
									},
								},
							}

						}
						if Event.Speed != nil {
							// Only register a trigger time if its not a raw analog value returned...
							lastTriggerTimeMU.Lock()
							lastTriggerTime = time.Now()
							lastTriggerTimeMU.Unlock()

							// Rotate color:
							HWCdispIndex[int(Event.HWCID)] = (HWCdispIndex[int(Event.HWCID)] + dispIndexCount + int(Event.Speed.Value)) % dispIndexCount
							HWCcolor[int(Event.HWCID)] = (HWCcolor[int(Event.HWCID)] + 15 + int(Event.Speed.Value)) % 15

							txt := rwp.HWCText{}
							txt.Formatting = 7
							txt.Title = fmt.Sprintf("HWc #%d", Event.HWCID)
							txt.SolidHeaderBar = true
							txt.Textline1 = fmt.Sprintf("%v", Event.Speed.Value)
							txt.Inverted = true

							incoming <- []*rwp.InboundMessage{
								&rwp.InboundMessage{
									States: []*rwp.HWCState{
										&rwp.HWCState{
											HWCIDs: activeHWCs,
											/*
												HWCMode: &rwp.HWCMode{
													State: rwp.HWCMode_ON,
												},
												HWCColor: &rwp.HWCColor{
													ColorIndex: &rwp.ColorIndex{
														Index: rwp.ColorIndex_Colors(HWCcolor[int(Event.HWCID)] + 2),
													},
												},*/
											HWCText: &txt,
										},
									},
								},
							}
						}
					}
				}
			}
		}
	}
}

// Main:
func main() {

	// Setting up and parsing command line parameters
	binPanel := flag.Bool("binPanel", false, "Connects to the panels in binary mode")
	invertCallAll := flag.Bool("invertCallAll", false, "Inverts which button edges that triggers 'call all' change of button colors and display contents. False=Left+Right edge, True=Up+Down+Encoder+None")
	verboseO := flag.Int("verboseOutgoing", 0, "Verbose output from panel, otherwise only events are shown. 1=Low intensity, 2=Higher intensity (protobuf messages as JSON)")
	verboseI := flag.Int("verboseIncoming", 0, "Verbose input messages to panel (default is none shown). 1=Low intensity, 2=Higher intensity (protobuf messages as JSON)")
	autoInterval := flag.Int("autoInterval", 100, "Interval in ms for demo engine sending out content")
	exclusiveHWClist := flag.String("exclusiveHWClist", "", "Comma separated list of HWC numbers to test exclusively")
	demoModeDelay := flag.Int("demoModeDelay", 10, "The number of seconds before demo mode starts after having manually operated a panel. Zero will disable demo mode.")
	demoModeImgsOnly := flag.Bool("demoModeImgsOnly", false, "If set, only images will be cycled to displays in demo mode")
	demoModeFaders := flag.Bool("demoModeFaders", false, "Exercise motorized faders continuously in demo mode.")
	analogProfiling := flag.Bool("analogProfiling", false, "If set, will track raw analog performance into CSV file and HTML pages in folder ColorDisplayButtonTest/")
	cpuProfiling := flag.Int("cpuProfiling", -1, "If >= zero, will turn on that number of CPU cores (0-4) and track temperature into CSV file and HTML pages in folder ColorDisplayButtonTest/")
	brightness := flag.Int("brightness", 4, "OLED and Display brightness. 0-8, default is 4.")
	fullPowerStartUp := flag.Bool("fullPowerStartUp", false, "If set, will panel will boot up with white screens and white LEDs all over.")

	flag.Parse()

	fmt.Println("Welcome to Raw Panel - Server Panel Color/Display/Button test! Made by Kasper Skaarhoj, (c) 2020-22")

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: [options] [panelIP:port]...")
		fmt.Println("help:  -h")
		fmt.Println("")
		return
	}

	if *analogProfiling {
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			for {
				select {
				case <-ticker.C:
					writeAnalogStatus()
				}
			}
		}()
	}

	startTicker()

	for panelNum, argument := range arguments {
		startTest(argument, binPanel, invertCallAll, panelNum+1, autoInterval, exclusiveHWClist, demoModeDelay, verboseO, verboseI, analogProfiling, cpuProfiling, brightness, fullPowerStartUp, demoModeFaders, demoModeImgsOnly)
	}
	select {}
}

func startTest(panelIPAndPort string, binPanel *bool, invertCallAll *bool, panelNum int, autoInterval *int, exclusiveHWClist *string, demoModeDelay *int, verboseO *int, verboseI *int, analogProfiling *bool, cpuProfiling *int, brightness *int, fullPowerStartUp *bool, demoModeFaders *bool, demoModeImgsOnly *bool) {

	// Set up server:
	incoming := make(chan []*rwp.InboundMessage, 100)
	outgoing := make(chan []*rwp.OutboundMessage, 100)

	go connectToPanel(panelIPAndPort, incoming, outgoing, *binPanel, panelNum, verboseI, analogProfiling, cpuProfiling, brightness, fullPowerStartUp)
	go testManager(incoming, outgoing, *invertCallAll, panelNum, autoInterval, exclusiveHWClist, demoModeDelay, verboseO, demoModeFaders, demoModeImgsOnly)
}

// Outputs a dot every second and line break after a minute. Great for logging activity under test
func startTicker() {
	ticker := time.NewTicker(1000 * time.Millisecond)
	count := 0
	go func() {
		for {
			select {
			case <-ticker.C:
				if count%60 == 0 {
					fmt.Printf("\n%4d:", count)
				}
				if count%10 == 0 {
					fmt.Print(" ")
				}
				fmt.Print(".")
				count++
			}
		}
	}()
}
