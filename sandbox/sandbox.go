package sandbox

import (
	"fmt"
	"os/exec"

	"golang.design/x/lockfree"
)

type Sandbox struct {
	AvailableBoxIDs *lockfree.Queue
	waitingQueue    *lockfree.Queue
	sandboxCount    int
}

func NewSandbox(count int) *Sandbox {
	availableBoxIDs := lockfree.NewQueue()
	for i := 0; i < count; i++ {
		cmd := exec.Command("isolate", "--init", fmt.Sprintf("-b %v", i))
		err := cmd.Run()
		if err != nil {
			panic(err)
		} else {
			availableBoxIDs.Enqueue(i)
		}
	}
	s := &Sandbox{
		AvailableBoxIDs: availableBoxIDs,
		sandboxCount:    count,
		waitingQueue:    lockfree.NewQueue(),
	}
	return s
}

func (s *Sandbox) Reserve() int {
	if item := s.AvailableBoxIDs.Dequeue(); item != nil {
		boxID := item.(int)
		return boxID
	}

	waitChan := make(chan int)
	s.waitingQueue.Enqueue(waitChan)
	return <-waitChan
}

func (s *Sandbox) Release(boxID int) {
	if item := s.waitingQueue.Dequeue(); item != nil {
		waitChan := item.(chan int)
		waitChan <- boxID
		close(waitChan)
	} else {
		s.AvailableBoxIDs.Enqueue(boxID)
	}
}

func (s *Sandbox) AvailableCount() int {
	return int(s.AvailableBoxIDs.Length())
}

func (s *Sandbox) WaitingCount() int {
	return int(s.waitingQueue.Length())
}

func (s *Sandbox) ProcessingCount() int {
	return s.sandboxCount - int(s.AvailableCount())
}

func (s *Sandbox) Cleanup() {
	for i := 0; i < s.sandboxCount; i++ {
		cmd := exec.Command("isolate", "-b", fmt.Sprintf("%v", i), "--cleanup")
		fmt.Printf("Cleaning up box %v\n", i)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error cleaning up box %v: %v\n", i, err)
		}
	}
}