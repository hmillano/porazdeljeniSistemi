package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/DistributedClocks/GoVector/govec"
)

type message struct {
	ID      int
	Content string
}

var (
	Logger          *govec.GoLog
	opts            govec.GoLogOptions
	N               int
	M               int
	K               int
	id              int
	basePort        int
	alreadyProcessed map[int]bool
	previousSender   map[int]int
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func receive(addr *net.UDPAddr, messageCh chan message, senderCh chan int) {
	conn, err := net.ListenUDP("udp", addr)
	checkError(err)
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		conn.SetDeadline(time.Now().Add(10 * time.Second))
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return
		}

		var receivedData string
		Logger.UnpackReceive("Prejeto sporocilo ", buffer[:n], &receivedData, opts)

		var msgID int
		var content string
		fmt.Sscanf(receivedData, "%d:%s", &msgID, &content)

		// Extract sender ID from the remote address
		senderPort := remoteAddr.Port
		senderID := senderPort - basePort

		messageCh <- message{ID: msgID, Content: content}
		senderCh <- senderID
	}
}

func send(addr *net.UDPAddr, msg message) {
	conn, err := net.DialUDP("udp", nil, addr)
	checkError(err)
	defer conn.Close()

	Logger.LogLocalEvent(fmt.Sprintf("Priprava sporocila z ID: %d", msg.ID), opts)

	data := Logger.PrepareSend(
		fmt.Sprintf("Poslano sporocilo z ID: %d", msg.ID),
		[]byte(fmt.Sprintf("%d:%s", msg.ID, msg.Content)),
		opts,
	)

	_, err = conn.Write(data)
	checkError(err)
}

func getRandomProcesses(excludeIDs map[int]bool, count int, total int) []int {
	rand.Seed(time.Now().UnixNano())
	processesToForwardTo := make(map[int]bool)
	for len(processesToForwardTo) < count {
		randomID := rand.Intn(total)
		if !excludeIDs[randomID] {
			processesToForwardTo[randomID] = true
		}
	}

	keys := make([]int, 0, len(processesToForwardTo))
	for k := range processesToForwardTo {
		keys = append(keys, k)
	}
	return keys
}

func broadcastMessage(msg message, excludeID int) {
	excludeIDs := map[int]bool{excludeID: true, id: true} // Exclude previous sender and self
	receivers := getRandomProcesses(excludeIDs, K, N)
	for _, receiver := range receivers {
		receiverAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", basePort+receiver))
		send(receiverAddr, msg)
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	portPtr := flag.Int("p", 9273, "base port")
	idPtr := flag.Int("id", 0, "process ID")
	nPtr := flag.Int("n", 4, "total number of processes")
	mPtr := flag.Int("m", 3, "number of messages to send")
	kPtr := flag.Int("k", 3, "number of processes to forward to")
	flag.Parse()

	basePort = *portPtr
	id = *idPtr
	N = *nPtr
	M = *mPtr
	K = *kPtr

	Logger = govec.InitGoVector("Proces-"+strconv.Itoa(id), "Log-Proces-"+strconv.Itoa(id), govec.GetDefaultConfig())
	opts = govec.GetDefaultLogOptions()

	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", basePort+id))
	checkError(err)

	messageCh := make(chan message, 10)
	senderCh := make(chan int, 10)
	alreadyProcessed = make(map[int]bool)
	previousSender = make(map[int]int)

	go receive(localAddr, messageCh, senderCh)

	if id == 0 {
		for i := 1; i <= M; i++ {
			msg := message{ID: i, Content: strconv.Itoa(i)}
			alreadyProcessed[msg.ID] = true
			broadcastMessage(msg, -1) // No previous sender for initial messages
		}
	} else {
		for {
			select {
			case msg := <-messageCh:
				sender := <-senderCh
				if !alreadyProcessed[msg.ID] {
					alreadyProcessed[msg.ID] = true
					previousSender[msg.ID] = sender
					broadcastMessage(msg, sender)
				}
			case <-time.After(45 * time.Second):
				return
			}
		}
	}
}
