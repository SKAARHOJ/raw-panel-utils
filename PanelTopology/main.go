/*
	Raw Panel Topology Extraction and SVG rendering (Example)

	Will connect to a panel, ask for its topology (SVG + JSON) and render a combined SVG
	saved into the filename "_topologySVGFullRender.svg"

	Distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
	without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
	PARTICULAR PURPOSE. MIT License
*/
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"

	log "github.com/s00500/env_logger"

	topology "github.com/SKAARHOJ/rawpanel-lib/topology"

	"go.uber.org/atomic"
)

var lastState *wsToClient
var lastStateMu sync.Mutex

// Panel centric view:
// Inbound TCP commands - from external system to SKAARHOJ panel
// Outbound TCP commands - from panel to external system
func connectToPanel(panelIPAndPort string, incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, binaryPanel bool) {

	for {
		fmt.Println("Trying to connect to panel on " + panelIPAndPort + " using " + su.Qstr(binaryPanel, "Binary mode", "ASCII mode") + "...")
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
						SendPanelTopology:     true,
						ReportHWCavailability: true,
						GetConnections:        true,
						GetRunTimeStats:       true,
						PublishSystemStat: &rwp.PublishSystemStat{
							PeriodSec: 15,
						},
						SetHeartBeatTimer: &rwp.HeartBeatTimer{
							Value: 3000,
						},
					},
				},
			}

			quit := make(chan bool)
			poll := time.NewTicker(time.Millisecond * 60 * 1000)
			go func() {
				//a := 0
				for {
					select {
					case <-quit:
						close(quit)
						return
					case incomingMessages := <-incoming:
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
								//fmt.Println(string("System -> Panel: " + strings.TrimSpace(string(line))))
								c.Write([]byte(line + "\n"))
							}
						}
					case <-poll.C:
						incoming <- []*rwp.InboundMessage{
							&rwp.InboundMessage{
								Command: &rwp.Command{
									GetConnections:  true,
									GetRunTimeStats: true,
								},
							},
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
						fmt.Println("Binary: ", err)
						break
					} else {
						currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
						if currentPayloadLength < 500000 {
							payload := make([]byte, currentPayloadLength)
							c.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
							_, err := io.ReadFull(c, payload)
							if err != nil {
								fmt.Println(err)
								break
							} else {
								outcomingMessage := &rwp.OutboundMessage{}
								proto.Unmarshal(payload, outcomingMessage)
								outgoing <- []*rwp.OutboundMessage{outcomingMessage}
							}
						} else {
							fmt.Println("Error: Payload", currentPayloadLength, "exceed limit")
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
			time.Sleep(time.Second * 3)
		}
	}
}

var TopologyData = &topology.Topology{}
var TotalUptimeGlobal uint32

func getTopology(incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage) {

	var sendStateToClient atomic.Bool
	var HWCavailabilityMap = make(map[uint32]uint32)
	var ReceivedTopology bool
	var IsSleeping bool
	var SendTopMutex sync.Mutex

	topologyJSON := ""
	topologySVG := ""

	t := time.NewTicker(time.Millisecond * 500)

	for {
		select {
		case outboundMessages := <-outgoing:
			// First, print the lines coming in as ASCII:
			lines := helpers.OutboundMessagesToRawPanelASCIIstrings(outboundMessages)
			for _, line := range lines {
				fmt.Println(string("Panel -> System: " + strings.TrimSpace(string(line))))
			}

			// Next, do some processing on it:
			SendTopMutex.Lock()
			for _, msg := range outboundMessages {

				if msg.PanelTopology != nil {
					if msg.PanelTopology.Json != "" {
						ReceivedTopology = true
						err := json.Unmarshal([]byte(msg.PanelTopology.Json), TopologyData)
						if err != nil {
							fmt.Println("Topology JSON parsing Error: ", err)
						} else {
							//fmt.Println("Received Topology JSON")
							topologyJSON = msg.PanelTopology.Json
							//log.Println(log.Indent(topologyData))
						}
					}
					if msg.PanelTopology.Svgbase != "" {
						ReceivedTopology = true
						topologySVG = msg.PanelTopology.Svgbase
						//	fmt.Println("Received Topology SVG")
					}
				}

				if msg.PanelInfo != nil {
					lastStateMu.Lock()
					if msg.PanelInfo.Name != "" {
						lastState.Title = msg.PanelInfo.Name
					}
					if msg.PanelInfo.Model != "" {
						lastState.Model = msg.PanelInfo.Model
					}
					if msg.PanelInfo.Serial != "" {
						lastState.Serial = msg.PanelInfo.Serial
					}
					if msg.PanelInfo.SoftwareVersion != "" {
						lastState.SoftwareVersion = msg.PanelInfo.SoftwareVersion
					}
					if msg.PanelInfo.Platform != "" {
						lastState.Platform = msg.PanelInfo.Platform
					}
					if msg.PanelInfo.BluePillReady {
						lastState.BluePillReady = "Yes"
					}
					if msg.PanelInfo.MaxClients != 0 {
						lastState.MaxClients = msg.PanelInfo.MaxClients
					}
					if len(msg.PanelInfo.LockedToIPs) != 0 {
						lastState.LockedToIPs = strings.Join(msg.PanelInfo.LockedToIPs, ";")
					}

					lastState.Time = getTimeString()
					lastStateMu.Unlock()
				}

				if msg.FlowMessage == 1 { // Ping:
					incoming <- []*rwp.InboundMessage{
						&rwp.InboundMessage{
							FlowMessage: 2,
						},
					}
				}

				if msg.FlowMessage == 5 { // RDY
					wsslice.Iter(func(w *wsclient) {
						w.msgToClient <- &wsToClient{
							RDYBSY: "<span style='color: red;'>BSY</span>",
						}
					})
				}

				if msg.FlowMessage == 5 { // BSY
					wsslice.Iter(func(w *wsclient) {
						w.msgToClient <- &wsToClient{
							RDYBSY: "<span style='color: green;'>RDY</span>",
						}
					})
				}

				if msg.SleepState != nil { // Sleeping flag
					IsSleeping = msg.SleepState.IsSleeping
					wsslice.Iter(func(w *wsclient) {
						w.msgToClient <- &wsToClient{
							Sleeping: su.Qstr(msg.SleepState.IsSleeping, "<span style='color: orange;'>Sleeping</span>", "<span style='color: green;'>Awake</span>"),
						}
					})
				}

				if msg.Connections != nil {
					lastStateMu.Lock()
					lastState.Connections = strings.Join(msg.Connections.Connection, " ") + " "
					lastStateMu.Unlock()
					sendStateToClient.Store(true)
				}
				if msg.RunTimeStats != nil {
					lastStateMu.Lock()
					if msg.RunTimeStats.BootsCount > 0 {
						lastState.BootsCount = msg.RunTimeStats.BootsCount
					}
					if msg.RunTimeStats.TotalUptime > 0 {
						TotalUptimeGlobal = msg.RunTimeStats.TotalUptime // Because we need the value below and these may not come in the same message (they DONT on ASCII version of RWP protocol...)
						lastState.TotalUptime = fmt.Sprintf("%dd %dh", msg.RunTimeStats.TotalUptime/60/24, (msg.RunTimeStats.TotalUptime/60)%24)
					}
					if msg.RunTimeStats.SessionUptime > 0 {
						lastState.SessionUptime = fmt.Sprintf("%dh %dm", msg.RunTimeStats.SessionUptime/60, msg.RunTimeStats.SessionUptime%60)
					}
					if msg.RunTimeStats.ScreenSaveOnTime > 0 {
						pct := -1
						if TotalUptimeGlobal > 0 {
							pct = 100 * int(msg.RunTimeStats.ScreenSaveOnTime) / int(TotalUptimeGlobal)
						}
						lastState.ScreenSaveOnTime = fmt.Sprintf("%dd %dh (%d%%)", msg.RunTimeStats.ScreenSaveOnTime/60/24, (msg.RunTimeStats.ScreenSaveOnTime/60)%24, pct)
					}
					lastStateMu.Unlock()
					sendStateToClient.Store(true)
				}

				// Picking up availability information (map command)
				if msg.HWCavailability != nil && !IsSleeping { // Only update the map internally if the panel is not asleep. Luckily the sleep indication will arrive before the updated map, so we can use this to prevent the map from being updated.
					//log.Println(log.Indent(msg.HWCavailability))
					for HWCid, MappedTo := range msg.HWCavailability {
						sendStateToClient.Store(true)
						HWCavailabilityMap[HWCid] = MappedTo
					}
				}

				if msg.Events != nil {
					for _, Event := range msg.Events {
						if Event.SysStat != nil {
							wsslice.Iter(func(w *wsclient) {
								w.msgToClient <- &wsToClient{
									CPUState: fmt.Sprintf("%.1fC, %d%%, %dMHz", Event.SysStat.CPUTemp, Event.SysStat.CPUUsage, Event.SysStat.CPUFreqCurrent/1000),
								}
							})
						} else {
							eventMessage := &wsToClient{
								PanelEvent: Event,
								Time:       getTimeString(),
							}
							wsslice.Iter(func(w *wsclient) { w.msgToClient <- eventMessage })

							eventPlot(Event)
						}
					}
				}
			}
			SendTopMutex.Unlock()
		case <-t.C: // Send topology based on a timer so that we don't trigger it on every received map command for example. Rather, state for map and topology will be pooled together and forwarded every half second.
			SendTopMutex.Lock()
			if (ReceivedTopology || sendStateToClient.Load()) && (topologyJSON != "" && topologySVG != "") {
				//fmt.Println("ReceivedTopology, sendStateToClient", ReceivedTopology, sendStateToClient)
				ReceivedTopology = false
				sendStateToClient.Store(false)
				//log.Println(log.Indent(HWCavailabilityMap))

				svgIcon := topology.GenerateCompositeSVG(topologyJSON, topologySVG, HWCavailabilityMap)

				regex := regexp.MustCompile(`id="HWc([0-9]+)"`)
				svgIcon = regex.ReplaceAllString(svgIcon, fmt.Sprintf("id=\"SVG_HWc$1\" onclick=\"clickHWC(evt,$1)\""))

				topOverviewTable := GenerateTopologyOverviewTable(topologyJSON, HWCavailabilityMap)
				topOverviewTable = regex.ReplaceAllString(topOverviewTable, fmt.Sprintf("id=\"Row_HWc$1\" onclick=\"clickHWC(event,$1)\""))
				//fmt.Println(topOverviewTable)

				// Create a JSON object to marshal in a pretty format
				var obj map[string]interface{}
				json.Unmarshal([]byte(topologyJSON), &obj)
				s, _ := json.MarshalIndent(obj, "", "  ")
				topJson := string(s)

				// Horrible, but functional processing of the JSON to insert some HTML to be able to highlight the HWCs
				regex = regexp.MustCompile(`"id": ([0-9]+),`)
				topJsonPartsBegin := strings.Split(topJson, "\n    {\n")
				for i := range topJsonPartsBegin {
					topJsonParts := strings.Split(topJsonPartsBegin[i], "\n    }")

					matches := regex.FindStringSubmatch(topJsonParts[0])
					if matches != nil {
						topJsonParts[0] = fmt.Sprintf(`<span id="Top_HWc%s" onclick="clickHWC(event,%s)">`, matches[1], matches[1]) + topJsonParts[0] + `</span>`
					}
					topJsonPartsBegin[i] = strings.Join(topJsonParts, "\n    }")
				}
				topJson = strings.Join(topJsonPartsBegin, "\n    {\n")
				//fmt.Println(topJson)

				// Process it...
				f, _ := os.Create("_topologySVGFullRender.svg")
				defer f.Close()
				f.WriteString(svgIcon)
				f.Sync()

				lastStateMu.Lock()
				lastState.SvgIcon = svgIcon
				lastState.TopologyTable = topOverviewTable
				lastState.TopologyJSON = topJson
				lastState.Time = getTimeString()
				wsslice.Iter(func(w *wsclient) { w.msgToClient <- lastState })
				lastStateMu.Unlock()
			}
			SendTopMutex.Unlock()
		}
	}
}

var incoming chan []*rwp.InboundMessage

func main() {

	// Setting up and parsing command line parameters
	binPanel := flag.Bool("binPanel", false, "Works with the panel in binary mode")
	flag.Parse()

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: PanelTopology [-binPanel] [panelIP:port]")
		fmt.Println("help:  PanelTopology -h")
		fmt.Println("")
		return
	}

	panelIPAndPort := string(arguments[0])

	// Welcome message!
	fmt.Println("Welcome to Raw Panel Topology Explorer made by Kasper Skaarhoj (c) 2021-2022")
	fmt.Println("Open a Web Browser on localhost:8080 to explore the topology interactively.")
	fmt.Println()

	lastStateMu.Lock()
	lastState = &wsToClient{
		Title:         "-",
		Model:         "-",
		Serial:        "-",
		SvgIcon:       "-",
		TopologyTable: "-",
		Time:          time.Now().String(),
	}
	lastStateMu.Unlock()

	// Set up server:
	incoming = make(chan []*rwp.InboundMessage, 10)
	outgoing := make(chan []*rwp.OutboundMessage, 10)

	go connectToPanel(panelIPAndPort, incoming, outgoing, *binPanel)

	// Start webserver:
	fmt.Println("Starting server on :8080")
	setupRoutes()
	go http.ListenAndServe(":8080", nil)

	wsslice = threadSafeSlice{}

	getTopology(incoming, outgoing)
}
