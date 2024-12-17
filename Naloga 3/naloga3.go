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
	Logger    *govec.GoLog
	opts      govec.GoLogOptions
	N         int
	M         int
	K         int
	id        int
	basePort  int
	processed map[int]bool
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func receive(addr *net.UDPAddr, messageCh chan message) {
	conn, err := net.ListenUDP("udp", addr)
	checkError(err)
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		conn.SetDeadline(time.Now().Add(10 * time.Second))
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return
		}

		var receivedData string
		Logger.UnpackReceive("Prejeto sporocilo ", buffer[:n], &receivedData, opts)

		var msgID int
		var content string
		fmt.Sscanf(receivedData, "%d:%s", &msgID, &content)

		messageCh <- message{ID: msgID, Content: content}
	}
}

// func send(addr *net.UDPAddr, msg message) {
// 	conn, err := net.DialUDP("udp", nil, addr)
// 	checkError(err)
// 	defer conn.Close()

// 	Logger.LogLocalEvent("Priprava sporocila ", opts)
// 	data := Logger.PrepareSend("Poslano sporocilo ", []byte(fmt.Sprintf("%d:%s", msg.ID, msg.Content)), opts)
// 	_, err = conn.Write(data)
// 	checkError(err)
// }

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


func getRandomRecipients(excludeID int, count int, total int) []int {
	rand.Seed(time.Now().UnixNano())
	recipients := make(map[int]bool)
	for len(recipients) < count {
		randomID := rand.Intn(total)
		if randomID != excludeID {
			recipients[randomID] = true
		}
	}

	keys := make([]int, 0, len(recipients))
	for k := range recipients {
		keys = append(keys, k)
	}
	return keys
}

func broadcastMessage(msg message) {
	recipients := getRandomRecipients(id, K, N)
	for _, recipient := range recipients {
		recipientAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", basePort+recipient))
		send(recipientAddr, msg)
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	portPtr := flag.Int("p", 9273, "Base port")
	idPtr := flag.Int("id", 0, "process ID")
	nPtr := flag.Int("n", 2, "total number of processes")
	mPtr := flag.Int("m", 1, "number of messages to send")
	kPtr := flag.Int("k", 2, "number of messages to forward")
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
	processed = make(map[int]bool)

	go receive(localAddr, messageCh)

	if id == 0 {
		for i := 1; i <= M; i++ {
			msg := message{ID: i, Content: strconv.Itoa(i)}
			processed[msg.ID] = true
			fmt.Printf("Process %d initiated message: %s\n", id, msg.Content)
			broadcastMessage(msg)
		}
	} else {
			for {
				select {
				case msg := <- messageCh:
					if !processed[msg.ID] {
						processed[msg.ID] = true
						fmt.Printf("Process %d received message: %s\n", id, msg.Content)
						broadcastMessage(msg)
					}
				case <-time.After(45 * time.Second):
					fmt.Printf("Process %d timed out. No new messages received.\n", id)
					return
				}
			}
		}
}
