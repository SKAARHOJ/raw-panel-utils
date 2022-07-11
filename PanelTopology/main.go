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

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
)

var PanelToSystemMessages *bool

func main() {

	// Welcome message!
	fmt.Println("Welcome to Raw Panel Topology Explorer made by Kasper Skaarhoj (c) 2021-2022")
	fmt.Println("Open a Web Browser on localhost:8080 to explore the topology interactively.")
	fmt.Println("-h for help")
	fmt.Println()

	// Setting up and parsing command line parameters
	//binPanel := flag.Bool("binPanel", false, "Works with the panel in binary mode")
	PanelToSystemMessages = flag.Bool("panelToSystemMessages", false, "If set, you will see panel to system messages written to the console")
	flag.Parse()

	arguments := flag.Args()

	// Start webserver:
	log.Infoln("Starting server on :8080")
	setupRoutes()
	go http.ListenAndServe(":8080", nil)

	wsslice = threadSafeSlice{}

	// Set up server:
	incoming = make(chan []*rwp.InboundMessage, 10)
	outgoing = make(chan []*rwp.OutboundMessage, 10)

	go runZeroConfSearch()

	// Load default panel up, if set:
	if len(arguments) > 0 {
		switchToPanel(string(arguments[0]))
	}

	// Wait forever:
	for {
		select {}
	}
}
