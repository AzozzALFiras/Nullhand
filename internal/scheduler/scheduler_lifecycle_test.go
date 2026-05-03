package scheduler

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAddSpecAssignsUniqueIDs(t *testing.T) {
	s := New()
	id1 := s.AddSpec(Task{ChatID: 1, UserID: 1, Label: "a", Hour: 9, Action: func() {}})
	id2 := s.AddSpec(Task{ChatID: 1, UserID: 1, Label: "b", Hour: 10, Action: func() {}})
	if id1 == id2 || id1 == "" || id2 == "" {
		t.Fatalf("AddSpec must return unique non-empty IDs, got %q and %q", id1, id2)
	}
}

func TestListReturnsSnapshot(t *testing.T) {
	s := New()
	s.AddSpec(Task{Label: "x", Hour: 9, Action: func() {}})
	tasks := s.List()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in list, got %d", len(tasks))
	}
	// Mutating the returned slice must not affect internal state.
	tasks[0].Label = "tampered"
	if s.List()[0].Label == "tampered" {
		t.Fatal("List() must return a copy, not aliased internal state")
	}
}

func TestCancelRemovesTask(t *testing.T) {
	s := New()
	id := s.AddSpec(Task{Label: "x", Hour: 9, Action: func() {}})
	if !s.Cancel(id) {
		t.Fatal("Cancel must return true for existing ID")
	}
	if len(s.List()) != 0 {
		t.Fatal("Cancel must remove the task")
	}
	if s.Cancel(id) {
		t.Fatal("Cancel must return false for already-removed ID")
	}
}

func TestClearWipesAll(t *testing.T) {
	s := New()
	for i := 0; i < 3; i++ {
		s.AddSpec(Task{Label: "x", Hour: 9, Action: func() {}})
	}
	s.Clear()
	if len(s.List()) != 0 {
		t.Fatal("Clear must drop every task")
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.json")

	// First scheduler writes a task to disk.
	s1 := New()
	s1.EnablePersistence(path)
	id := s1.AddSpec(Task{
		ChatID: 42, UserID: 7, Label: "morning report",
		Hour: 8, Minute: 30,
		ActionText: "screenshot",
		Days:       []time.Weekday{time.Monday, time.Friday},
		Action:     func() {},
	})
	if id == "" {
		t.Fatal("AddSpec returned empty ID")
	}
	// AddSpec persists asynchronously — give it a beat.
	time.Sleep(100 * time.Millisecond)

	// Second scheduler loads the task and verifies it survived.
	s2 := New()
	s2.EnablePersistence(path)
	hydrated := false
	err := s2.LoadFrom(func(actionText string, chatID, userID int64, taskID string) (func(), string) {
		hydrated = true
		if actionText != "screenshot" || chatID != 42 || userID != 7 {
			t.Errorf("hydrator got wrong fields: action=%q chat=%d user=%d", actionText, chatID, userID)
		}
		// Empty label preserves the persisted Label.
		return func() {}, ""
	})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !hydrated {
		t.Fatal("persisted task did not invoke hydrator on reload")
	}
	loaded := s2.List()
	if len(loaded) != 1 {
		t.Fatalf("expected 1 task after reload, got %d", len(loaded))
	}
	got := loaded[0]
	if got.Hour != 8 || got.Minute != 30 || got.Label != "morning report" {
		t.Errorf("reload lost fields: %+v", got)
	}
	if len(got.Days) != 2 {
		t.Errorf("reload lost Days filter: %v", got.Days)
	}
}

func TestLoadFromDropsUnresolvable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.json")

	s1 := New()
	s1.EnablePersistence(path)
	s1.AddSpec(Task{ChatID: 1, Label: "good", Hour: 9, ActionText: "ok", Action: func() {}})
	s1.AddSpec(Task{ChatID: 1, Label: "bad", Hour: 10, ActionText: "missing-recipe", Action: func() {}})
	time.Sleep(100 * time.Millisecond)

	s2 := New()
	s2.EnablePersistence(path)
	err := s2.LoadFrom(func(actionText string, chatID, userID int64, taskID string) (func(), string) {
		if actionText == "missing-recipe" {
			return nil, ""
		}
		return func() {}, ""
	})
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got := len(s2.List()); got != 1 {
		t.Fatalf("expected 1 task after dropping unresolvable, got %d", got)
	}
	if s2.List()[0].ActionText != "ok" {
		t.Errorf("wrong task survived: action=%q", s2.List()[0].ActionText)
	}
}
