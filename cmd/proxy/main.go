// Huefy Go Proxy Server
// This server allows other language SDKs to proxy API calls through the Go core
// instead of making direct calls to api.huefy.dev

package main

import (
	"flag"
	"log"
	"os"
	
	huefy "github.com/teracrafts/huefy-sdk-go/v2"
)

func main() {
	var port = flag.Int("port", 8080, "Port to run the proxy server on")
	flag.Parse()
	
	// Create and start proxy server
	proxy := huefy.NewProxyServer(*port)
	
	log.Printf("Starting Huefy Go proxy server on port %d", *port)
	log.Println("This proxy allows other Huefy SDKs to route through Go core")
	log.Println("Press Ctrl+C to stop the server")
	
	if err := proxy.Start(); err != nil {
		log.Printf("Failed to start proxy server: %v", err)
		os.Exit(1)
	}
}