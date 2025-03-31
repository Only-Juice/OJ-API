package sandbox

import (
	"OJ-API/models"
	"container/heap"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type Sandbox struct {
	mu              sync.Mutex
	AvailableBoxIDs map[int]bool // map of boxID to availability
	waitingQueue    PriorityQueue
	sandboxCount    int
}

type WaitItem struct {
	waitChan chan int
	priority time.Time // Timestamp for priority
	index    int       // Index in the heap
}

// PriorityQueue implements a priority queue for WaitItem.
type PriorityQueue []*WaitItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Earlier time has higher priority
	return pq[i].priority.Before(pq[j].priority)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*WaitItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // Avoid memory leak
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

func NewSandbox(count int) *Sandbox {
	availableBoxIDs := make(map[int]bool, count)
	for i := 0; i < count; i++ {
		cmd := exec.Command("isolate", "--init", fmt.Sprintf("-b %v", i))
		err := cmd.Run()
		if err != nil {
			panic(err)
		} else {
			availableBoxIDs[i] = true
		}
	}
	s := &Sandbox{
		AvailableBoxIDs: availableBoxIDs,
		sandboxCount:    count,
		waitingQueue:    make(PriorityQueue, 0),
	}
	heap.Init(&s.waitingQueue)
	return s
}

func (s *Sandbox) Reserve(userQuestion models.UserQuestionTable) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	for boxID := range s.AvailableBoxIDs {
		delete(s.AvailableBoxIDs, boxID)
		return boxID
	}

	waitChan := make(chan int)
	priority := userQuestion.JudgeTime
	item := &WaitItem{
		waitChan: waitChan,
		priority: priority,
	}
	heap.Push(&s.waitingQueue, item)
	s.mu.Unlock()
	boxID := <-waitChan
	s.mu.Lock()
	return boxID
}

func (s *Sandbox) Release(boxID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.waitingQueue.Len() > 0 {
		item := heap.Pop(&s.waitingQueue).(*WaitItem)
		item.waitChan <- boxID
		close(item.waitChan)
	} else {
		s.AvailableBoxIDs[boxID] = true
	}
}

func (s *Sandbox) AvailableCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.AvailableBoxIDs)
}

func (s *Sandbox) WaitingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waitingQueue.Len()
}

func (s *Sandbox) ProcessingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sandboxCount - len(s.AvailableBoxIDs)
}

func (s *Sandbox) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for boxID := range s.AvailableBoxIDs {
		cmd := exec.Command("isolate", "--cleanup", fmt.Sprintf("-b %v", boxID))
		fmt.Printf("Cleaning up box %v\n", boxID)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error cleaning up box %v: %v\n", boxID, err)
		}
	}
}
