package sandbox

import (
	"fmt"
	"os/exec"
	"sync"
)

type Sandbox struct {
	mu              sync.Mutex
	AvailableBoxIDs map[int]bool // map of boxID to availability
	waitingQueue    []chan int
	sandboxCount    int
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
		waitingQueue:    make([]chan int, 0),
	}
	return s
}

func (s *Sandbox) Reserve() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	for boxID := range s.AvailableBoxIDs {
		delete(s.AvailableBoxIDs, boxID)
		return boxID
	}

	waitChan := make(chan int)
	s.waitingQueue = append(s.waitingQueue, waitChan)
	s.mu.Unlock()
	boxID := <-waitChan
	s.mu.Lock()
	return boxID
}

func (s *Sandbox) Release(boxID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.waitingQueue) > 0 {
		waitChan := s.waitingQueue[0]
		s.waitingQueue = s.waitingQueue[1:]
		waitChan <- boxID
		close(waitChan)
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
	return len(s.waitingQueue)
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
