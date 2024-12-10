package main

import (
	"Naloga2/socialNetwork"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Task struct {
	Id   uint64
	Data string
}

type InvertedIndex struct {
	mu sync.RWMutex
	keyword map[string][]uint64
}

func ExtractKeywordsFromIndex(content string) []string {
	var keywords []string
	var builder strings.Builder

	for _, r := range content {
		r = unicode.ToLower(r)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else if builder.Len() > 0 {
			if builder.Len() >= 4 {
				keywords = append(keywords, builder.String())
			}
			builder.Reset()
		}
	}

	if builder.Len() >= 4 {
		keywords = append(keywords, builder.String())
	}

	return keywords
}

func (index *InvertedIndex) AddKeywordsToIndex(keywords []string, id uint64) {
	index.mu.Lock()
	defer index.mu.Unlock()
	for _, keyword := range keywords {
		index.keyword[keyword] = append(index.keyword[keyword], id)
	}
}

func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		keyword: make(map[string][]uint64),
	}
}


func (index *InvertedIndex) GetIndex() map[string][]uint64 {
	index.mu.RLock()
	defer index.mu.Unlock()
	return index.keyword
}

func worker(id int, taskChan <-chan Task, index *InvertedIndex, wg *sync.WaitGroup, done <-chan struct{}) {
    defer wg.Done()
    for {
        select {
        case task := <-taskChan:
            if task.Id == 0 && task.Data == "" {
                return
            }
            keywords := ExtractKeywordsFromIndex(task.Data)
            index.AddKeywordsToIndex(keywords, task.Id)
        case <-done:
            // fmt.Printf("Worker %d stopped\n", id)
            return
        }
    }
}

func supervisor(taskChan <-chan Task, producer *socialNetwork.Q, index *InvertedIndex) {
	var wg sync.WaitGroup
	
	maxWorkers := 500
	initialWorkers := 100
	checkPeriod := 1 * time.Millisecond
	scaleUpThresh := 500
	scaleDownThresh := 100

	done := make(chan struct{})
	activeWorkers := make(chan int, maxWorkers)

	for i := 0; i < initialWorkers; i++ {
		wg.Add(1)
		go worker(i, taskChan, index, &wg, done)
		activeWorkers <- i
	}

	ticker := time.NewTicker(checkPeriod)
	defer ticker.Stop()

	for range ticker.C {
		taskChanLength := len(taskChan)

		if taskChanLength > scaleUpThresh && len(activeWorkers) < maxWorkers {
			newWorkerID := len(activeWorkers)
			// fmt.Printf("Scaling up: Adding worker %d\n", newWorkerID)
			wg.Add(1)
			go worker(newWorkerID, taskChan, index, &wg, done)
			activeWorkers <- newWorkerID
		}

		if taskChanLength < scaleDownThresh && len(activeWorkers) > 1 {
			// fmt.Println("Scaling down: Removing a worker")
			<-activeWorkers
			done <- struct{}{}
		}

		if producer.QueueEmpty() && len(taskChan) == 0 {
			break
		}
	}

	close(done)
	wg.Wait()
}


func main() {
	var producer socialNetwork.Q
	producer.New(10000)

	index := NewInvertedIndex()
	taskChan := make(chan Task, 10000)

	start := time.Now()

	// fmt.Println("Starting supervisor")
	go supervisor(taskChan, &producer, index)

	// fmt.Println("Starting producer")
	go producer.Run()

	go func() {
		for task := range producer.TaskChan {
			convertedTask := Task{
				Id:   task.Id,
				Data: task.Data,
			}
			taskChan <- convertedTask
		}
		close(taskChan)
	}()

	
    time.Sleep(5 * time.Second)

	// fmt.Println("Stopping producer")
	producer.Stop()

	for !producer.QueueEmpty() {
		time.Sleep(2000 * time.Millisecond)
	}
	elapsed := time.Since(start)
	
	fmt.Println(len(index.keyword))
	fmt.Printf("Processing rate: %f MReqs/s\n", float64(producer.N)/float64(elapsed.Seconds())/1000000.0)
	fmt.Printf("Average queue length: %.2f %%\n", producer.GetAverageQueueLength())
	fmt.Printf("Max queue length %.2f %%\n", producer.GetMaxQueueLength())
}
