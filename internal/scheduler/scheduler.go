package scheduler

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Task is a recurring daily task that fires at a specific hour:minute.
type Task struct {
	ID     string
	Label  string
	Hour   int
	Minute int
	ChatID int64
	Action func()
}

// Scheduler manages a list of daily tasks and fires them at the right time.
// It is safe for concurrent use.
type Scheduler struct {
	mu      sync.Mutex
	tasks   []*Task
	counter int
	stop    chan struct{}
}

// New creates an idle Scheduler. Call Start() to begin firing tasks.
func New() *Scheduler {
	return &Scheduler{stop: make(chan struct{})}
}

// Add registers a new daily task and returns its assigned ID.
func (s *Scheduler) Add(chatID int64, label string, hour, minute int, action func()) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	id := fmt.Sprintf("task_%03d", s.counter)
	s.tasks = append(s.tasks, &Task{
		ID:     id,
		Label:  label,
		Hour:   hour,
		Minute: minute,
		ChatID: chatID,
		Action: action,
	})
	return id
}

// List returns a snapshot of all currently registered tasks.
func (s *Scheduler) List() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Task, len(s.tasks))
	for i, t := range s.tasks {
		out[i] = *t
	}
	return out
}

// Cancel removes the task with the given ID. Returns true if found and removed.
func (s *Scheduler) Cancel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes all registered tasks.
func (s *Scheduler) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = s.tasks[:0]
}

// Start begins the background goroutine that checks tasks every minute.
func (s *Scheduler) Start() {
	go s.loop()
}

// Stop halts the background goroutine.
func (s *Scheduler) Stop() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

func (s *Scheduler) loop() {
	// Align to the next whole minute before starting the ticker.
	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	select {
	case <-time.After(time.Until(nextMinute)):
	case <-s.stop:
		return
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	// Fire tasks for the aligned minute, then enter normal loop.
	s.check(time.Now())

	for {
		select {
		case t := <-ticker.C:
			s.check(t)
		case <-s.stop:
			return
		}
	}
}

func (s *Scheduler) check(t time.Time) {
	h, m := t.Hour(), t.Minute()
	s.mu.Lock()
	// Copy matching tasks before unlocking so Action() is not called under lock.
	var fire []*Task
	for _, task := range s.tasks {
		if task.Hour == h && task.Minute == m {
			fire = append(fire, task)
		}
	}
	s.mu.Unlock()

	for _, task := range fire {
		go func(t *Task) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("scheduler: task %s panicked: %v", t.ID, r)
				}
			}()
			t.Action()
		}(task)
	}
}
