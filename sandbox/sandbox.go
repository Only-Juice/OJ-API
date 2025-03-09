package sandbox

import (
	"fmt"
	"os/exec"
	"sync"
)

type Sandbox struct {
	mu              sync.Mutex
	cond            *sync.Cond
	AvailableBoxIDs []int
	waitingCount    int
}

func NewSandbox(count int) *Sandbox {
	availableBoxIDs := make([]int, count)
	for i := 0; i < count; i++ {
		cmd := exec.Command("isolate", "--init", fmt.Sprintf("-b %v", i))
		err := cmd.Run()
		if err != nil {
			panic(err)
		} else {
			availableBoxIDs[i] = i
		}
	}
	s := &Sandbox{
		AvailableBoxIDs: availableBoxIDs,
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *Sandbox) Reserve() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	for len(s.AvailableBoxIDs) == 0 {
		s.waitingCount++
		s.cond.Wait()
		s.waitingCount--
	}

	boxID := s.AvailableBoxIDs[0]
	s.AvailableBoxIDs = s.AvailableBoxIDs[1:]
	return boxID
}

func (s *Sandbox) Release(boxID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AvailableBoxIDs = append(s.AvailableBoxIDs, boxID)
	s.cond.Signal()
}

func (s *Sandbox) AvailableCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.AvailableBoxIDs)
}

func (s *Sandbox) WaitingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waitingCount
}

func (s *Sandbox) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, boxID := range s.AvailableBoxIDs {
		cmd := exec.Command("isolate", "--cleanup", fmt.Sprintf("-b %v", boxID))
		fmt.Printf("Cleaning up box %v\n", boxID)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error cleaning up box %v: %v\n", boxID, err)
		}
	}
}
