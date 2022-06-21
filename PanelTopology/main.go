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
	"os"
	"strconv"
	"strings"
	"time"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/subchen/go-xmldom"
	"google.golang.org/protobuf/proto"

	log "github.com/s00500/env_logger"
	xml "github.com/subchen/go-xmldom"

	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
)

// Panel centric view:
// Inbound TCP commands - from external system to SKAARHOJ panel
// Outbound TCP commands - from panel to external system
func connectToPanel(panelIPAndPort string, incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, binaryPanel bool) {

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
						SendPanelTopology:     true,
						ReportHWCavailability: true,
					},
				},
			}

			quit := make(chan bool)
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

func getTopology(incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage) {

	//	panelInitialized := false
	//HWCavailabilityMap := make(map[int]int)

	topologyJSON := ""
	topologySVG := ""
	var topologyData topology.Topology

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

				if msg.PanelTopology != nil {
					if msg.PanelTopology.Json != "" {
						err := json.Unmarshal([]byte(msg.PanelTopology.Json), &topologyData)
						if err != nil {
							fmt.Println("Topology JSON parsing Error: ", err)
						} else {
							//fmt.Println("Received Topology JSON")
							topologyJSON = msg.PanelTopology.Json
							log.Println(log.Indent(topologyData))
						}
					}
					if msg.PanelTopology.Svgbase != "" {
						topologySVG = msg.PanelTopology.Svgbase
						//	fmt.Println("Received Topology SVG")
					}
				}

				if topologyJSON != "" && topologySVG != "" {
					generateCompositeSVG(topologyJSON, topologySVG)
					return
				}
			}
		}
	}
}

func addFormatting(newHWc *xml.Node, id int) {
	// There is some common conventional formatting regardless of rectangle / circle: Like fill and stroke color and stroke width.
	newHWc.SetAttributeValue("fill", "#dddddd")
	newHWc.SetAttributeValue("stroke", "#000")
	newHWc.SetAttributeValue("stroke-width", "2")
	newHWc.SetAttributeValue("id", "HWc"+strconv.Itoa(id)) // Also, lets add an id to the element! This is not mandatory, but you are likely to want this to program some interaction with the SVG
}

func addSubElFormatting(newHWc *xml.Node, subEl *topology.TopologyHWcTypeDefSubEl) {

	if subEl.Rx != 0 {
		newHWc.SetAttributeValue("rx", strconv.Itoa(subEl.Rx))
	}
	if subEl.Ry != 0 {
		newHWc.SetAttributeValue("ry", strconv.Itoa(subEl.Ry))
	}
	if subEl.Style != "" {
		newHWc.SetAttributeValue("style", subEl.Style)
	}

	// There is some common conventional formatting regardless of rectangle / circle: Like fill and stroke color and stroke width.
	newHWc.SetAttributeValue("fill", "#cccccc")
	newHWc.SetAttributeValue("stroke", "#666")
	newHWc.SetAttributeValue("stroke-width", "1")
}
func generateCompositeSVG(topologyJSON string, topologySVG string) {

	showLabels := true       // Will render text labels on the SVG icon file
	showHWCID := true        // Will render HWC ID number on the SVG icon file
	showType := false        // Will render the type id above each component (for development)
	showDisplaySize := false // Will render the display sizes in pixels at every display (for development)

	// Parsing SVG file:
	svgDoc, err := xmldom.ParseXML(topologySVG)
	if err != nil {
		log.Fatal(err)
	}

	// Reading JSON topology:
	var topology topology.Topology
	json.Unmarshal([]byte(topologyJSON), &topology)

	topology.Verify()

	for _, HWcDef := range topology.HWc {
		typeDef := topology.TypeIndex[HWcDef.Type]

		// Look for local type override and overlay it if it's there..:
		// Across controllers, this is largely alternative disp{} pixel dimensions and some sub[] changes.
		if HWcDef.TypeOverride != nil {
			if HWcDef.TypeOverride.W > 0 {
				typeDef.W = HWcDef.TypeOverride.W
			}
			if HWcDef.TypeOverride.H > 0 {
				typeDef.H = HWcDef.TypeOverride.H
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
			if HWcDef.TypeOverride.Disp != nil {
				typeDef.Disp = HWcDef.TypeOverride.Disp
			}
			if len(HWcDef.TypeOverride.Sub) > 0 {
				typeDef.Sub = HWcDef.TypeOverride.Sub
			}
			//					su.Debug(HWcDef.TypeOverride)
		}

		// Main elements:
		newHWc := svgDoc.Root.CreateNode(su.Qstr(typeDef.H > 0, "rect", "circle"))
		if typeDef.H > 0 { // Rectangle
			newHWc.SetAttributeValue("x", strconv.Itoa(HWcDef.X-typeDef.W/2)) // SVG elements have their reference point in upper left corner, so we subtract half the width from the center x-coordinate of the element
			newHWc.SetAttributeValue("y", strconv.Itoa(HWcDef.Y-typeDef.H/2)) // SVG elements have their reference point in upper left corner, so we subtract half the height from the center y-coordinate of the element
			newHWc.SetAttributeValue("width", strconv.Itoa(typeDef.W))
			newHWc.SetAttributeValue("height", strconv.Itoa(typeDef.H))
			newHWc.SetAttributeValue("rx", strconv.Itoa(10)) // Rounding corners for visual elegance
			newHWc.SetAttributeValue("rx", strconv.Itoa(10)) // Rounding corners for visual elegance
		} else { // Circle
			newHWc.SetAttributeValue("cx", strconv.Itoa(HWcDef.X))
			newHWc.SetAttributeValue("cy", strconv.Itoa(HWcDef.Y))
			newHWc.SetAttributeValue("r", strconv.Itoa(typeDef.W/2)) // Radius is half the width
		}
		addFormatting(newHWc, int(HWcDef.Id))

		// Sub elements:
		if len(typeDef.Sub) > 0 {
			for _, subEl := range typeDef.Sub {
				if subEl.ObjType == "r" {
					subElForHWc := svgDoc.Root.CreateNode("rect")
					subElForHWc.SetAttributeValue("x", strconv.Itoa(HWcDef.X+subEl.X))
					subElForHWc.SetAttributeValue("y", strconv.Itoa(HWcDef.Y+subEl.Y))
					subElForHWc.SetAttributeValue("width", strconv.Itoa(subEl.W))
					subElForHWc.SetAttributeValue("height", strconv.Itoa(subEl.H))
					addSubElFormatting(subElForHWc, &subEl)
				}
				if subEl.ObjType == "c" {
					subElForHWc := svgDoc.Root.CreateNode("circle")
					subElForHWc.SetAttributeValue("cx", strconv.Itoa(HWcDef.X+subEl.X))
					subElForHWc.SetAttributeValue("cy", strconv.Itoa(HWcDef.Y+subEl.Y))
					subElForHWc.SetAttributeValue("r", strconv.Itoa(subEl.R))
					addSubElFormatting(subElForHWc, &subEl)
				}
			}
		}

		// Text labels:
		if showLabels {
			sp := strings.Split(HWcDef.Txt, "|")
			cnt := len(sp)
			if cnt > 1 && len(sp[1]) > 0 {
				cnt = 2
			} else {
				cnt = 1
			}
			for a := 0; a < cnt; a++ {
				textElForHWC := svgDoc.Root.CreateNode("text")
				textElForHWC.SetAttributeValue("x", strconv.Itoa(HWcDef.X))
				textElForHWC.SetAttributeValue("y", strconv.Itoa(HWcDef.Y+33+a*40-(cnt*40/2)))
				textElForHWC.SetAttributeValue("text-anchor", "middle")
				textElForHWC.SetAttributeValue("fill", "#000")
				textElForHWC.SetAttributeValue("font-weight", "bold")
				textElForHWC.SetAttributeValue("font-size", "35")
				textElForHWC.SetAttributeValue("font-family", "sans-serif")
				textElForHWC.Text = sp[a]
			}
		}

		if showType {
			// If type number was printed as label, we will add a small text with the original label too:
			textForTypeNumber := svgDoc.Root.CreateNode("text")
			textForTypeNumber.SetAttributeValue("x", strconv.Itoa(HWcDef.X))
			textForTypeNumber.SetAttributeValue("y", strconv.Itoa(HWcDef.Y-su.Qint(typeDef.H > 0, typeDef.H, typeDef.W)/2-2))
			textForTypeNumber.SetAttributeValue("text-anchor", "middle")
			textForTypeNumber.SetAttributeValue("fill", "#333")
			textForTypeNumber.SetAttributeValue("font-size", "20")
			textForTypeNumber.SetAttributeValue("font-family", "sans-serif")
			textForTypeNumber.Text = "[TYPE=" + strconv.Itoa(int(HWcDef.Type)) + "]"
		}

		if showDisplaySize && typeDef.Disp != nil {
			textForDisplaySize := svgDoc.Root.CreateNode("text")
			dispLabelX := HWcDef.X
			dispLabelY := HWcDef.Y - su.Qint(typeDef.H > 0, typeDef.H, typeDef.W)/2 - 2
			if typeDef.Disp.Subidx >= 0 && len(typeDef.Sub) > typeDef.Disp.Subidx {
				dispLabelX = HWcDef.X + typeDef.Sub[typeDef.Disp.Subidx].X + typeDef.Sub[typeDef.Disp.Subidx].W/2
				dispLabelY = HWcDef.Y + typeDef.Sub[typeDef.Disp.Subidx].Y + typeDef.Sub[typeDef.Disp.Subidx].H/2
			}

			textForDisplaySize.SetAttributeValue("x", strconv.Itoa(dispLabelX))
			textForDisplaySize.SetAttributeValue("y", strconv.Itoa(dispLabelY))
			textForDisplaySize.SetAttributeValue("text-anchor", "middle")
			textForDisplaySize.SetAttributeValue("fill", "#ccc")
			textForDisplaySize.SetAttributeValue("font-size", "25")
			textForDisplaySize.SetAttributeValue("font-family", "sans-serif")
			textForDisplaySize.SetAttributeValue("stroke", "#333")
			textForDisplaySize.SetAttributeValue("stroke-width", "6px")
			textForDisplaySize.SetAttributeValue("paint-order", "stroke")

			dispLabelSuffix := ""
			if typeDef.Disp.Type != "" {
				dispLabelSuffix = " " + typeDef.Disp.Type
			}

			textForDisplaySize.Text = strconv.Itoa(typeDef.Disp.W) + "x" + strconv.Itoa(typeDef.Disp.H) + dispLabelSuffix
		}

		if showHWCID {
			numberForHWC := svgDoc.Root.CreateNode("text")
			numberForHWC.SetAttributeValue("x", strconv.Itoa(HWcDef.X-su.Qint(typeDef.H > 0, typeDef.W/2-4, 0)))
			numberForHWC.SetAttributeValue("y", strconv.Itoa(HWcDef.Y-su.Qint(typeDef.H > 0, typeDef.H, typeDef.W)/2+20))
			if typeDef.H == 0 { // Circle: Center it...
				numberForHWC.SetAttributeValue("text-anchor", "middle")
			}
			numberForHWC.SetAttributeValue("fill", "#000")
			numberForHWC.SetAttributeValue("font-size", "20")
			numberForHWC.SetAttributeValue("font-family", "sans-serif")
			//numberForHWC.SetAttributeValue("stroke", "#dddddd")
			//numberForHWC.SetAttributeValue("stroke-width", "6px")
			//numberForHWC.SetAttributeValue("paint-order", "stroke")
			numberForHWC.Text = strconv.Itoa(int(HWcDef.Id))
		}
	}

	// Process it...
	f, _ := os.Create("_topologySVGFullRender.svg")
	defer f.Close()
	f.WriteString(svgDoc.XMLPretty())
	f.Sync()
}

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
	fmt.Println("Welcome to Raw Panel Topology Extractor Made by Kasper Skaarhoj (c) 2021-2022")
	fmt.Println("Ready to connect to panel on " + panelIPAndPort + "...\n")

	// Set up server:
	incoming := make(chan []*rwp.InboundMessage, 10)
	outgoing := make(chan []*rwp.OutboundMessage, 10)

	go connectToPanel(panelIPAndPort, incoming, outgoing, *binPanel)

	getTopology(incoming, outgoing)
}
