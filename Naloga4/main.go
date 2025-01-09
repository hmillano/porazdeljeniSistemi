// main.go

package main

import (
	"flag"
	"fmt"
	"os"
)

type NodeConfig struct {
    nodeType    NodeType
    listenPort  int
    nextNodeURL string
}

func main() {
    nodeIDPtr := flag.Int("id", 0, "Node ID in chain")
    portPtr := flag.Int("p", 8500, "Base port number")
    nodeCountPtr := flag.Int("n", 1, "Total number of nodes in chain")
    clientModePtr := flag.Bool("client", false, "Run in client mode")
    flag.Parse()
    fmt.Println("main.go: flags parsing finished")
    if *clientModePtr {
        runClient(*nodeCountPtr, *portPtr)
    } else {
        runServer(*nodeIDPtr, *nodeCountPtr, *portPtr)
    }
}

func getHostname() string {
    hostname, err := os.Hostname()
    if err != nil {
        fmt.Printf("Failed to get hostname: %v\n", err)
        os.Exit(1)
    }
    return hostname
}

func runClient(nodeCount, basePort int) {
    hostname := getHostname()
    headURL := fmt.Sprintf("%s:%d", hostname, basePort)
    tailURL := fmt.Sprintf("%s:%d", hostname, basePort+nodeCount-1)

    fmt.Printf("RPC client connecting to headURL at %s and tailURL at %s\n", headURL, tailURL)
    Client(headURL, tailURL)
}

func runServer(nodeID, nodeCount, basePort int) {
    if nodeID >= nodeCount {
        fmt.Printf("Invalid node ID %d for chain of length %d\n", nodeID, nodeCount)
        os.Exit(1)
    }

    var nodeType NodeType
    switch {
    case nodeID == 0:
        nodeType = HEAD
    case nodeID == nodeCount-1:
        nodeType = TAIL
    default:
        nodeType = MIDDLE
    }

    hostname := getHostname()
    currentPort := basePort + nodeID
    listenURL := fmt.Sprintf(":%d", currentPort)
    
    var prevNodeURL, nextNodeURL string
    
    if nodeID > 0 {
        prevNodeURL = fmt.Sprintf("%s:%d", hostname, basePort+nodeID-1)
    }
    
    if nodeID < nodeCount-1 {
        nextNodeURL = fmt.Sprintf("%s:%d", hostname, basePort+nodeID+1)
    }

    fmt.Printf("Starting node %d/%d as %v\n", nodeID, nodeCount-1, nodeType)
    fmt.Printf("Listening on port: %d\n", currentPort)
    if prevNodeURL != "" {
        fmt.Printf("Previous node URL: %s\n", prevNodeURL)
    }
    if nextNodeURL != "" {
        fmt.Printf("Next node URL: %s\n", nextNodeURL)
    }

    err := StartServer(nodeType, listenURL, prevNodeURL, nextNodeURL)
    if err != nil {
        fmt.Printf("Server failed: %v\n", err)
        os.Exit(1)
    }
}