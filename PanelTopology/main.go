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
	"flag"
	"fmt"
	"net/http"
	"time"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
)

var PanelToSystemMessages *bool
var writeTopologiesToFiles *bool
var AggressiveQuery *bool

func main() {

	// Welcome message!
	fmt.Println("Welcome to Raw Panel Topology Explorer made by Kasper Skaarhoj (c) 2021-2022")
	fmt.Println("Opens a Web Browser on localhost:8080 to explore the topology interactively.")
	fmt.Println("usage: [options] [panelIP:port] [Shadow panelIP:port]")
	fmt.Println("-h for help")
	fmt.Println()

	// Setting up and parsing command line parameters
	//binPanel := flag.Bool("binPanel", false, "Works with the panel in binary mode")
	PanelToSystemMessages = flag.Bool("panelToSystemMessages", false, "If set, you will see panel to system messages written to the console")
	writeTopologiesToFiles = flag.Bool("writeTopologiesToFiles", false, "If set, the JSON, SVG and rendered full SVG icon is written to files in the working directory.")
	dontOpenBrowser := flag.Bool("dontOpenBrowser", false, "If set, a web browser won't open automatically")
	AggressiveQuery = flag.Bool("aggressive", false, "If set, will connect to panels, query various info and disconnect.")
	flag.Parse()

	arguments := flag.Args()

	// Start webserver:
	port := 8080
	log.Infof("Starting server on localhost:%d\n", port)
	setupRoutes()
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	if !(*dontOpenBrowser) {
		go func() {
			time.Sleep(time.Millisecond * 500)
			openBrowser(fmt.Sprintf("http://localhost:%d", port))
		}()
	}

	wsslice = threadSafeSlice{}

	// Set up server:
	incoming = make(chan []*rwp.InboundMessage, 10)
	outgoing = make(chan []*rwp.OutboundMessage, 50)
	shadowPanelIncoming = make(chan []*rwp.InboundMessage, 10)

	demoHWCids.Store([]uint32{})

	go runZeroConfSearch()

	// Load default panel up, if set:
	if len(arguments) > 0 {
		switchToPanel(string(arguments[0]))
	}

	if len(arguments) > 1 {
		fmt.Println("Connection to shadow panel: ", string(arguments[1]))
		connectToShadowPanel(string(arguments[1]), shadowPanelIncoming)
	}

	// Wait forever:
	for {
		select {}
	}
}
