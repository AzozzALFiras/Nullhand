package scheduler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Task is a recurring task that fires on a configurable schedule.
//
// Action is the closure that actually fires; it is NOT serialized. ActionText
// is the original natural-language action description (e.g. "screenshot",
// "run df -h") and IS serialized so the action can be rehydrated by a
// Hydrator on startup.
//
// Schedule semantics — how the daily check decides whether to fire:
//   - If Days is non-empty, fire only when time.Now().Weekday() is in Days.
//     Days uses Go's time.Sunday=0..Saturday=6 convention.
//   - If Days is empty (nil or zero-length), fire every day (legacy behavior).
//   - Hour:Minute is the primary trigger time. If ExtraTimes is non-empty,
//     the task ALSO fires at every "HH:MM" listed there (same Days filter).
type Task struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Hour       int            `json:"hour"`
	Minute     int            `json:"minute"`
	Days       []time.Weekday `json:"days,omitempty"`        // empty = every day
	ExtraTimes []string       `json:"extra_times,omitempty"` // additional "HH:MM" trigger times
	ChatID     int64          `json:"chat_id"`
	UserID     int64          `json:"user_id"`
	ActionText string         `json:"action_text"`
	Action     func()         `json:"-"`
}

// Hydrator rebuilds an Action closure from the persisted descriptor fields at
// load time. Returning nil signals "no longer resolvable" and the task is
// dropped from the schedule.
type Hydrator func(actionText string, chatID, userID int64, taskID string) (func(), string)

// Scheduler manages a list of daily tasks and fires them at the right time.
// It is safe for concurrent use. Optionally persists task descriptors to a
// JSON file so schedules survive restarts.
type Scheduler struct {
	mu        sync.Mutex
	tasks     []*Task
	counter   int
	stop      chan struct{}
	savePath  string  // empty = no persistence
}

// New creates an idle Scheduler. Call Start() to begin firing tasks.
func New() *Scheduler {
	return &Scheduler{stop: make(chan struct{})}
}

// EnablePersistence configures the scheduler to read/write the task list from
// path. Pass an empty string to disable. Subsequent Add/Cancel/Clear calls
// will save the file automatically.
func (s *Scheduler) EnablePersistence(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savePath = path
}

// Add registers a new daily task and returns its assigned ID. actionText is
// the natural-language descriptor that a Hydrator can re-resolve on reload.
//
// This is the legacy single-time-per-day entry point; use AddSpec for cron-
// style tasks (specific weekdays, multiple times per day).
func (s *Scheduler) Add(chatID, userID int64, label, actionText string, hour, minute int, action func()) string {
	return s.AddSpec(Task{
		ChatID:     chatID,
		UserID:     userID,
		Label:      label,
		ActionText: actionText,
		Hour:       hour,
		Minute:     minute,
		Action:     action,
	})
}

// AddSpec registers a task using a fully-populated Task struct (minus ID,
// which is generated). Use this for tasks with Days or ExtraTimes set.
func (s *Scheduler) AddSpec(t Task) string {
	s.mu.Lock()
	s.counter++
	t.ID = fmt.Sprintf("task_%03d", s.counter)
	taskCopy := t
	s.tasks = append(s.tasks, &taskCopy)
	s.mu.Unlock()
	s.persistAsync()
	return t.ID
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
	found := false
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			found = true
			break
		}
	}
	s.mu.Unlock()
	if found {
		s.persistAsync()
	}
	return found
}

// Clear removes all registered tasks.
func (s *Scheduler) Clear() {
	s.mu.Lock()
	s.tasks = s.tasks[:0]
	s.mu.Unlock()
	s.persistAsync()
}

// LoadFrom reads task descriptors from the configured persistence file and
// rehydrates them via the supplied Hydrator. Tasks the hydrator can't resolve
// are silently dropped. Existing in-memory tasks are replaced.
//
// Safe to call before Start(). Does nothing if persistence is not enabled or
// the file doesn't exist yet.
func (s *Scheduler) LoadFrom(hydrate Hydrator) error {
	s.mu.Lock()
	path := s.savePath
	s.mu.Unlock()
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("scheduler: read %s: %w", path, err)
	}
	var persisted []Task
	if err := json.Unmarshal(data, &persisted); err != nil {
		return fmt.Errorf("scheduler: parse %s: %w", path, err)
	}

	s.mu.Lock()
	s.tasks = s.tasks[:0]
	maxID := 0
	for _, t := range persisted {
		fn, label := hydrate(t.ActionText, t.ChatID, t.UserID, t.ID)
		if fn == nil {
			log.Printf("scheduler: dropping unhydratable task %s (action=%q)", t.ID, t.ActionText)
			continue
		}
		if label != "" {
			t.Label = label
		}
		t.Action = fn
		taskCopy := t
		s.tasks = append(s.tasks, &taskCopy)
		// Keep the counter ahead of any restored ID so new ones don't collide.
		var n int
		if _, err := fmt.Sscanf(t.ID, "task_%d", &n); err == nil && n > maxID {
			maxID = n
		}
	}
	if maxID > s.counter {
		s.counter = maxID
	}
	s.mu.Unlock()
	return nil
}

// persistAsync writes the current task list to the configured savePath.
// Failures are logged but not fatal — scheduling keeps working in memory.
// Runs in a goroutine so callers (Add/Cancel/Clear) aren't blocked on disk.
func (s *Scheduler) persistAsync() {
	s.mu.Lock()
	if s.savePath == "" {
		s.mu.Unlock()
		return
	}
	path := s.savePath
	snap := make([]Task, len(s.tasks))
	for i, t := range s.tasks {
		snap[i] = *t
	}
	s.mu.Unlock()

	go func() {
		if err := writeTasksFile(path, snap); err != nil {
			log.Printf("scheduler: persist %s: %v", path, err)
		}
	}()
}

// writeTasksFile serializes tasks to a JSON file atomically (write to temp,
// then rename). Creates the parent directory if missing.
func writeTasksFile(path string, tasks []Task) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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
	dow := t.Weekday()
	s.mu.Lock()
	// Copy matching tasks before unlocking so Action() is not called under lock.
	var fire []*Task
	for _, task := range s.tasks {
		if !taskMatchesNow(task, h, m, dow) {
			continue
		}
		fire = append(fire, task)
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

// taskMatchesNow reports whether task t should fire at the given hour, minute,
// and weekday. Honours Days (filter) and ExtraTimes (additional fire points).
func taskMatchesNow(t *Task, h, m int, dow time.Weekday) bool {
	// Day-of-week filter.
	if len(t.Days) > 0 {
		match := false
		for _, d := range t.Days {
			if d == dow {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	// Primary time.
	if t.Hour == h && t.Minute == m {
		return true
	}
	// Extra times — each "HH:MM".
	for _, raw := range t.ExtraTimes {
		eh, em, ok := parseHHMM(raw)
		if ok && eh == h && em == m {
			return true
		}
	}
	return false
}

// parseHHMM parses "HH:MM" into hour and minute. Returns ok=false on bad input.
func parseHHMM(s string) (int, int, bool) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}
