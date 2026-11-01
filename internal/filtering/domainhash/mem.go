package domainhash

import (
	"bytes"
	"os"
	"strconv"
)

const (
	// DefaultMemBudgetBytes is the memory budget used when the amount of
	// available memory cannot be detected.
	DefaultMemBudgetBytes int64 = 32 << 20

	// MinMemBudgetBytes is the lowest memory budget returned by
	// [DetectMemBudget].
	MinMemBudgetBytes int64 = 8 << 20

	// defaultBatchBytes is the maximum number of bytes the builder buffers in
	// memory before flushing a batch.
	defaultBatchBytes int64 = 16 << 20

	// meminfoPath is the path to the Linux memory information file.
	meminfoPath = "/proc/meminfo"
)

// DetectMemBudget returns the number of bytes that the blocklist builder may
// use for its in-memory batch.  marginBytes is the amount of memory the rest of
// the process is expected to need, and it is subtracted from the amount of
// memory currently available on the system.  If the available memory cannot be
// determined, [DefaultMemBudgetBytes] is returned.  The returned value is never
// less than [MinMemBudgetBytes].
func DetectMemBudget(marginBytes int64) (budget int64) {
	if marginBytes < 0 {
		marginBytes = 0
	}

	available, ok := availableMemory()
	if !ok {
		return DefaultMemBudgetBytes
	}

	budget = available - marginBytes
	if budget < MinMemBudgetBytes {
		return MinMemBudgetBytes
	}

	return budget
}

// availableMemory returns the amount of memory available for new allocations,
// in bytes, as reported by the operating system.  ok is false if it cannot be
// determined.
func availableMemory() (n int64, ok bool) {
	data, err := os.ReadFile(meminfoPath)
	if err != nil {
		return 0, false
	}

	return parseMemAvailable(data)
}

// parseMemAvailable parses the MemAvailable field of the contents of
// /proc/meminfo and returns it in bytes.  ok is false if there is no usable
// MemAvailable field in data.
func parseMemAvailable(data []byte) (n int64, ok bool) {
	for len(data) > 0 {
		var line []byte
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			line, data = data[:i], data[i+1:]
		} else {
			line, data = data, nil
		}

		rest, found := bytes.CutPrefix(line, []byte("MemAvailable:"))
		if !found {
			continue
		}

		fields := bytes.Fields(rest)
		if len(fields) == 0 {
			return 0, false
		}

		kb, err := strconv.ParseInt(string(fields[0]), 10, 64)
		if err != nil || kb <= 0 {
			return 0, false
		}

		return kb * 1024, true
	}

	return 0, false
}
