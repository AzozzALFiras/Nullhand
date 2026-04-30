package system

// Stats holds a snapshot of system resource usage. Populated by the per-OS
// GetStats implementations.
type Stats struct {
	CPUUsed   string // e.g. "3.1% user + 1.2% sys"
	CPUIdle   string // e.g. "95.4%"
	MemUsed   string // e.g. "8192.0M"
	MemUnused string // e.g. "1234.5M"
	ActiveApp string // foreground window title
}
