package bot

import (
	"testing"
	"time"

	configmodel "github.com/AzozzALFiras/Nullhand/internal/model/config"
)

func TestIdleLockTimeout(t *testing.T) {
	cases := []struct {
		name string
		cfg  *configmodel.Config
		want time.Duration
	}{
		{"nil config uses default", nil, defaultIdleLock},
		{"zero uses default", &configmodel.Config{}, defaultIdleLock},
		{"custom minutes", &configmodel.Config{IdleLockMinutes: 5}, 5 * time.Minute},
		{"negative disables", &configmodel.Config{IdleLockMinutes: -1}, 0},
	}
	for _, c := range cases {
		if got := idleLockTimeout(c.cfg); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}
