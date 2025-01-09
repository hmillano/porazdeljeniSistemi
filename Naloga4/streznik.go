// streznik.go

package main

import (
	"fmt"
	"naloga4/storage"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type NodeType int

const (
    HEAD NodeType = iota
    MIDDLE
    TAIL
)

type ChainNode struct {
    nodeType    NodeType
    todoStore   storage.TodoStorage
    nextNode    *rpc.Client
    prevNode    *rpc.Client
    nextNodeURL string
    prevNodeURL string
    mu          sync.Mutex
}

func NewChainNode(nodeType NodeType, prevNodeURL, nextNodeURL string) (*ChainNode, error) {
    node := &ChainNode{
        nodeType:    nodeType,
        todoStore:   *storage.NewTodoStorage(),
        nextNodeURL: nextNodeURL,
        prevNodeURL: prevNodeURL,
    }

    if nodeType != TAIL && nextNodeURL != "" {
        var client *rpc.Client
        var err error
        for {
            client, err = rpc.Dial("tcp", nextNodeURL)
            if err != nil {
                fmt.Printf("Failed to connect to next node: %v. Retrying in 2 seconds...\n", err)
                time.Sleep(2 * time.Second)
                continue
            }
            break
        }
        node.nextNode = client
    }

    if nodeType != HEAD && prevNodeURL != "" {
        var client *rpc.Client
        var err error
        for {
            client, err = rpc.Dial("tcp", prevNodeURL)
            if err != nil {
                fmt.Printf("Failed to connect to previous node: %v. Retrying in 2 seconds...\n", err)
                time.Sleep(2 * time.Second)
                continue
            }
            break
        }
        node.prevNode = client
    }

    return node, nil
}


func (chainNode *ChainNode) Put(todo *storage.Todo, reply *struct{}) error {
    if chainNode.nodeType == HEAD {
        chainNode.mu.Lock()
        defer chainNode.mu.Unlock()
    }

    localTodo := &storage.Todo{
        Task:      todo.Task,
        Completed: todo.Completed,
        Commited:  false,
    }

    var ret struct{}
    err := chainNode.todoStore.Put(localTodo, &ret)
    if err != nil {
        return fmt.Errorf("Local storage failed: %v", err)
    }

    if chainNode.nodeType != TAIL && chainNode.nextNode != nil {
        go func() {
            var nextReply struct{}
            err := chainNode.nextNode.Call("ChainNode.Put", todo, &nextReply)
            if err != nil {
                fmt.Printf("Failed to forward to next node: %v\n", err)
            }
        }()
    }

    return nil    
}

func (chainNode *ChainNode) Commit(todo *storage.Todo, reply *struct{}) error {
    chainNode.mu.Lock()
    defer chainNode.mu.Unlock()

    localTodo := &storage.Todo{
        Task:      todo.Task,
        Completed: todo.Completed,
        Commited:  true,
    }

    var ret struct{}
    err := chainNode.todoStore.Commit(localTodo, &ret)
    if err != nil {
        return fmt.Errorf("Failed to commit locally: %v", err)
    }

    if chainNode.nodeType != HEAD && chainNode.prevNode != nil {
        var prevReply struct{}
        err := chainNode.prevNode.Call("ChainNode.Commit", localTodo, &prevReply)
        if err != nil {
            return fmt.Errorf("Failed to propagate commit to previous node: %v", err)
        }
    }

    return nil
}


func (chainNode *ChainNode) Get(todo *storage.Todo, dict *map[string]storage.Todo, reply *struct{}) error {
    if chainNode.nodeType != TAIL {
        return fmt.Errorf("Only tail node can call Get RPC!")
    }

    return chainNode.todoStore.Get(todo, dict)
}


func StartServer(nodeType NodeType, listenAddr string, prevNodeURL string, nextNodeURL string) error {
    fmt.Println("StartServer function start")
    node, err := NewChainNode(nodeType, prevNodeURL, nextNodeURL)
    if err != nil {
        return fmt.Errorf("Failed to create chain node: %v", err)
    }

    rpc.Register(node)

    listener, err := net.Listen("tcp", listenAddr)
    if err != nil {
        return fmt.Errorf("Failed to listen: %v", err)
    }

    fmt.Printf("Chain node type %v listening at %v\n", nodeType, listenAddr)

    rpc.Accept(listener)
    return nil
}