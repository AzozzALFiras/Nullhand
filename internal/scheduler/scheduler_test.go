package scheduler

import (
	"testing"
	"time"
)

func TestTaskMatchesNow(t *testing.T) {
	mon := time.Monday
	wed := time.Wednesday
	sat := time.Saturday

	tests := []struct {
		name string
		task Task
		h, m int
		dow  time.Weekday
		want bool
	}{
		{
			name: "every day at 9am — matches Mon 09:00",
			task: Task{Hour: 9, Minute: 0},
			h:    9, m: 0, dow: mon, want: true,
		},
		{
			name: "every day at 9am — does not match 09:01",
			task: Task{Hour: 9, Minute: 0},
			h:    9, m: 1, dow: mon, want: false,
		},
		{
			name: "Mon-only at 9am — matches Mon",
			task: Task{Hour: 9, Minute: 0, Days: []time.Weekday{mon}},
			h:    9, m: 0, dow: mon, want: true,
		},
		{
			name: "Mon-only at 9am — does NOT match Wed",
			task: Task{Hour: 9, Minute: 0, Days: []time.Weekday{mon}},
			h:    9, m: 0, dow: wed, want: false,
		},
		{
			name: "weekday at 9am — matches Mon",
			task: Task{Hour: 9, Minute: 0, Days: []time.Weekday{mon, time.Tuesday, wed, time.Thursday, time.Friday}},
			h:    9, m: 0, dow: mon, want: true,
		},
		{
			name: "weekday at 9am — does NOT match Sat",
			task: Task{Hour: 9, Minute: 0, Days: []time.Weekday{mon, time.Tuesday, wed, time.Thursday, time.Friday}},
			h:    9, m: 0, dow: sat, want: false,
		},
		{
			name: "extra time — matches second time",
			task: Task{Hour: 9, Minute: 0, ExtraTimes: []string{"17:30"}},
			h:    17, m: 30, dow: mon, want: true,
		},
		{
			name: "extra time — does NOT match unrelated minute",
			task: Task{Hour: 9, Minute: 0, ExtraTimes: []string{"17:30"}},
			h:    17, m: 31, dow: mon, want: false,
		},
		{
			name: "extra time + Days filter — matches",
			task: Task{Hour: 9, Minute: 0, ExtraTimes: []string{"17:30"}, Days: []time.Weekday{mon}},
			h:    17, m: 30, dow: mon, want: true,
		},
		{
			name: "extra time + Days filter — wrong day",
			task: Task{Hour: 9, Minute: 0, ExtraTimes: []string{"17:30"}, Days: []time.Weekday{mon}},
			h:    17, m: 30, dow: wed, want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := taskMatchesNow(&tc.task, tc.h, tc.m, tc.dow)
			if got != tc.want {
				t.Errorf("taskMatchesNow(%+v, %d:%02d, %s) = %v, want %v",
					tc.task, tc.h, tc.m, tc.dow, got, tc.want)
			}
		})
	}
}
