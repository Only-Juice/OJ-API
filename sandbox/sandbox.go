package sandbox

import (
	"OJ-API/models"
	"OJ-API/utils"
	"fmt"
	"os/exec"
	"time"

	"golang.design/x/lockfree"
)

type Sandbox struct {
	AvailableBoxIDs *lockfree.Queue // Sandbox that can use
	waitingQueue    *lockfree.Queue // Sandbox that executing code
	jobQueue        *lockfree.Queue // Storing Unjudge job
	sandboxCount    int             // How many sandbox
}

type Job struct {
	Repo     string
	CodePath []byte
	UQR      models.UserQuestionTable
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
		jobQueue:        lockfree.NewQueue(),
	}
	return s
}

func (s *Sandbox) Reserve(timeout time.Duration) (int, bool) {
	if item := s.AvailableBoxIDs.Dequeue(); item != nil {
		return item.(int), true
	}

	waitChan := make(chan int, 1)
	s.waitingQueue.Enqueue(waitChan)

	select {
	case boxID := <-waitChan:
		return boxID, true
	case <-time.After(timeout):
		return -1, false
	}
}

func (s *Sandbox) Release(boxID int) {
	item := s.waitingQueue.Dequeue()

	if item != nil {
		if waitChan, ok := item.(chan int); ok {
			select {
			case waitChan <- boxID:
				close(waitChan)
			default:
				s.AvailableBoxIDs.Enqueue(boxID)
			}
			return
		}
	}
	/* Release Isolate resource */

	cmd := exec.Command("isolate", "--init", fmt.Sprintf("-b %v", boxID))
	cmd.Run()

	s.AvailableBoxIDs.Enqueue(boxID)
}

func (s *Sandbox) AvailableCount() int {
	return int(s.AvailableBoxIDs.Length())
}

func (s *Sandbox) WaitingCount() int {
	return int(s.jobQueue.Length())
}

func (s *Sandbox) ProcessingCount() int {
	return s.sandboxCount - int(s.AvailableCount())
}

func (s *Sandbox) IsJobEmpty() bool {
	return s.jobQueue.Length() == 0
}

func (s *Sandbox) ReserveJob(repo string, codePath []byte, uqtid models.UserQuestionTable) {

	job := &Job{
		Repo:     repo,
		CodePath: codePath,
		UQR:      uqtid,
	}
	s.jobQueue.Enqueue(job)
}

func (s *Sandbox) ReleaseJob() *Job {

	if int(s.jobQueue.Length()) == 0 {
		return nil // queue 是空的
	}

	item := s.jobQueue.Dequeue()

	job, ok := item.(*Job)
	if !ok {
		utils.Warn("[Sandbox] Dequeued item is not of type *Job")
		return nil
	}

	return job
}

func (s *Sandbox) Cleanup() {
	for i := 0; i < s.sandboxCount; i++ {
		cmd := exec.Command("isolate", "-b", fmt.Sprintf("%v", i), "--cleanup")
		utils.Debugf("Cleaning up box %v", i)
		err := cmd.Run()
		if err != nil {
			utils.Errorf("Error cleaning up box %v: %v", i, err)
		}

	}
	for {
		ok := s.AvailableBoxIDs.Dequeue()
		if ok == nil {
			break
		}
	}

	for {
		ok := s.waitingQueue.Dequeue()
		if ok == nil {
			break
		}
	}
}
