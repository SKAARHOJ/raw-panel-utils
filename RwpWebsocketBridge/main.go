package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/s00500/env_logger"
)

var (
	clientID          = flag.String("client_id", "", "Client ID for authentication")
	clientSecret      = flag.String("client_secret", "", "Client secret for authentication")
	allowInsecureAuth = flag.Bool("allow_insecure_auth", false, "Allow authentication over insecure ws:// connections (NOT RECOMMENDED)")
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage: RwpWebsocketBridge [options] "IP[:PORT]=>EndpointAddress" ...
	
	Options:
	  -client_id     Client ID for authentication
	  -client_secret Client secret for authentication
  	  -allow_insecure_auth   Allow client_id/client_secret over insecure ws:// (NOT RECOMMENDED)
	`)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	type AddressMapping struct {
		PanelAddr      string // The IP:Port address of the panel device. Port is optional, defaults to 9923.
		ServerEndpoint string // The WebSocket endpoint address for this panel. Format: "ws://example.com:port/path" or "wss://example.com:port/path".
	}

	// Parse the command line arguments into address mappings
	var mappings []AddressMapping
	for _, arg := range flag.Args() {
		parts := strings.SplitN(arg, "=>", 2)
		if len(parts) != 2 {
			log.Errorf("Invalid format: %s. Expected IP[:PORT]=>Endpoint", arg)
			os.Exit(1)
		}

		local := parts[0]
		if !strings.Contains(local, ":") {
			local += ":9923"
		}

		endpoint := parts[1]
		if !strings.HasPrefix(endpoint, "ws://") && !strings.HasPrefix(endpoint, "wss://") {
			log.Errorf("Invalid endpoint: %s. Must start with ws:// or wss://", endpoint)
			os.Exit(1)
		}

		if (*clientID != "" || *clientSecret != "") && !strings.HasPrefix(endpoint, "wss://") {
			if !*allowInsecureAuth {
				log.Errorf("Endpoint %s must use wss:// when client_id or client_secret is set. Use -allow_insecure_auth to override (NOT RECOMMENDED).", endpoint)
				os.Exit(1)
			} else {
				log.Warnf("WARNING: Using authentication over insecure ws:// connection to %s", endpoint)
			}
		}

		mappings = append(mappings, AddressMapping{
			PanelAddr:      local,
			ServerEndpoint: endpoint,
		})
	}

	for {
		var bridges []*BridgeConnection

		// Create a bridge connection for each mapping
		for _, m := range mappings {
			log.Infof("Starting bridge for %s => %s", m.PanelAddr, m.ServerEndpoint)
			bridgeConn := NewBridgeManager(m.PanelAddr, m.ServerEndpoint, *clientID, *clientSecret)
			bridgeConn.Start()
			bridges = append(bridges, bridgeConn)
		}

		select {} // Block forever until we manually stop the bridges

		// Wait for 30 seconds
		log.Infof("Running bridges for 30 seconds...")
		time.Sleep(30 * time.Second)

		log.Infof("Shutting down all bridges...")

		for _, conn := range bridges {
			conn.Shutdown()
			time.Sleep(time.Second)
		}

		log.Infof("Restarting bridge connections in 5 seconds...\n")
		time.Sleep(5 * time.Second)
	}
}
