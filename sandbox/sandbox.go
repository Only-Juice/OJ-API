package sandbox

import (
	"fmt"
	"os/exec"
	"sync"
)

type Sandbox struct {
	mu				sync.Mutex
	AvailableBoxIDs	[]int
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
	return &Sandbox{
		AvailableBoxIDs: availableBoxIDs,
	}
}

func (s *Sandbox) Reserve() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	boxID := s.AvailableBoxIDs[0]
	s.AvailableBoxIDs = s.AvailableBoxIDs[1:]
	return boxID
}

func (s *Sandbox) Release(boxID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AvailableBoxIDs = append(s.AvailableBoxIDs, boxID)
}

func (s *Sandbox) AvailableCount() int {
	return len(s.AvailableBoxIDs)
}

func (s *Sandbox) Cleanup() {
	for _, boxID := range s.AvailableBoxIDs {
		cmd := exec.Command("isolate", "--cleanup", fmt.Sprintf("-b %v", boxID))
		fmt.Printf("Cleaning up box %v\n", boxID)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error cleaning up box %v: %v\n", boxID, err)
		}
	}
}