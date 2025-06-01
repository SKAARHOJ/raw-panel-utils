package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
	log "github.com/s00500/env_logger"
)

func main() {

	// Customize the usage message for clarity.
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: GridsExample [IP[:PORT] ...]\n")
	}
	flag.Parse()

	// If no arguments are passed, print usage and exit.
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Extract all IP[:PORT] arguments.
	addresses := flag.Args()

	// Start an infinite loop to continuously (re)connect to panels for demonstration purposes.
	log.Infof("Starting panel connections for addresses: %v", addresses)
	for {
		var panelConnections []*PanelConnection

		// Attempt to create and start a connection for each address.
		for _, addr := range addresses {

			// If no port is specified, default to 9923.
			if !strings.Contains(addr, ":") {
				addr += ":9923"
			}

			// Create a new PanelConnection struct (holds all state for this panel).
			panel := NewPanelConnection(addr)

			// Start the connection in a background goroutine.
			panel.Start()

			// Keep track of it so we can shut it down later.
			panelConnections = append(panelConnections, panel)
		}

		log.Infof("All connections started. Waiting for shutdown in 30 seconds...")
		time.Sleep(30 * time.Second)

		// For demonstration: simulate shutdown and clean reconnect by canceling all connections.
		for _, p := range panelConnections {
			// Cancel the panel's context — this tells all its goroutines to exit.
			p.Cancel()

			// Wait for those goroutines to actually finish.
			p.Wg.Wait()
		}

		// Log and sleep before reconnecting (simulate reconnection attempt).
		log.Warn("All connections exited. Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

// NewPanelConnection creates a new PanelConnection struct for the given address (IP:Port).
// It initializes its channels, context, and internal data structures.
// This function is the entry point for setting up per-panel connection state.
func NewPanelConnection(addr string) *PanelConnection {

	// Create a cancellable context used to control this connection's lifecycle.
	ctx, cancel := context.WithCancel(context.Background())

	// Return a fully initialized struct instance.
	return &PanelConnection{
		Addr:         addr,                                   // Panel's network address
		Incoming:     make(chan []*rwp.InboundMessage, 100),  // Buffered channel for messages to panel
		Outgoing:     make(chan []*rwp.OutboundMessage, 100), // Buffered channel for messages from panel
		Ctx:          ctx,                                    // Context for cancellation
		Cancel:       cancel,                                 // Cancel function
		TopologyData: &topology.Topology{},                   // Empty topology placeholder
		HWCMap:       make(map[uint32]uint32),                // Initial empty HWC map
	}
}
