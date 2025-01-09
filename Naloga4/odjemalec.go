// odjemalec.go

package main

import (
	"fmt"
	"naloga4/storage"
	"net/rpc"
	"time"
)

type ChainClient struct {
    headNode *rpc.Client
    tailNode *rpc.Client
}

func NewChainClient(headURL, tailURL string) (*ChainClient, error) {
    time.Sleep(5 * time.Second)
	
	// Connect to head node
    fmt.Printf("RPC client connecting to headURL at %v\n", headURL)
    headClient, err := rpc.Dial("tcp", headURL)
    if err != nil {
        panic(err)
    }
    time.Sleep(3 * time.Second)

    // Connect to tail node
    fmt.Printf("RPC client connecting to tailURL at %v\n", tailURL)
    tailClient, err := rpc.Dial("tcp", tailURL)
    if err != nil {
        panic(err)
    }
    time.Sleep(3 * time.Second)

    return &ChainClient{
        headNode: headClient,
        tailNode: tailClient,
    }, nil    
}


func Client(headURL, tailURL string) {
    client, err := NewChainClient(headURL, tailURL)
    if err != nil {
        fmt.Printf("Failed to create client: %v\n", err)
        return
    }

	var reply struct{}
	var replyMap map[string]storage.Todo

	lecturesCreate := storage.Todo{Task: "predavanja", Completed: false}
	lecturesUpdate := storage.Todo{Task: "predavanja", Completed: true}
	practicals := storage.Todo{Task: "vaje", Completed: false}


    fmt.Print("1. Put: ")
	if err := client.headNode.Call("Node.Put", &lecturesCreate, &reply); err != nil {
		panic(err)
	}
	fmt.Println("Done")

	fmt.Print("2. Get 1: ")
	replyMap = make(map[string]storage.Todo)
	if err := client.tailNode.Call("Node.Get", &lecturesUpdate, &replyMap); err != nil {
		panic(err)
	}
	fmt.Println(replyMap, ": Done")

	fmt.Print("3. Put update: ")
	if err := client.headNode.Call("Node.Put", &lecturesUpdate, &reply); err != nil {
		panic(err)
	}
	fmt.Println("Done")

	fmt.Print("4. Put practicals: ")
	if err := client.headNode.Call("Node.Put", &practicals, &reply); err != nil {
		panic(err)
	}
	fmt.Println("Done")
}