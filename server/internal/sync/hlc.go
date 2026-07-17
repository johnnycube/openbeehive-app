// Package sync holds the sync building blocks. hlc.go implements a
// Hybrid Logical Clock: monotonic, wall-clock-close timestamps that are
// totally ordered across all devices and sort lexicographically.
package sync

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Format: "<ms:15>:<counter:5>:<nodeID>" — feste Breite => String-Sort = Zeit-Sort.
type HLC struct {
	mu      sync.Mutex
	lastMS  int64
	counter int64
	nodeID  string
}

func NewHLC(nodeID string) *HLC { return &HLC{nodeID: nodeID} }

func nowMS() int64 { return time.Now().UnixMilli() }

// Now returns a fresh local timestamp (for local writes).
func (h *HLC) Now() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	pt := nowMS()
	if pt > h.lastMS {
		h.lastMS, h.counter = pt, 0
	} else {
		h.counter++
	}
	return h.format(h.lastMS, h.counter)
}

// Recv updates the clock when receiving a foreign timestamp and
// guarantees the local clock is afterwards > the received stamp.
func (h *HLC) Recv(remote string) {
	rms, rc, _ := parse(remote)
	h.mu.Lock()
	defer h.mu.Unlock()
	pt := nowMS()
	switch {
	case pt > h.lastMS && pt > rms:
		h.lastMS, h.counter = pt, 0
	case rms > h.lastMS:
		h.lastMS, h.counter = rms, rc+1
	case h.lastMS > rms:
		h.counter++
	default:
		if rc > h.counter {
			h.counter = rc
		}
		h.counter++
	}
}

func (h *HLC) format(ms, c int64) string {
	return fmt.Sprintf("%015d:%05d:%s", ms, c, h.nodeID)
}

// Compare: -1 a<b, 0 equal, 1 a>b. A plain string comparison suffices.
func Compare(a, b string) int { return strings.Compare(a, b) }

func parse(s string) (ms, c int64, node string) {
	p := strings.SplitN(s, ":", 3)
	if len(p) != 3 {
		return 0, 0, ""
	}
	ms, _ = strconv.ParseInt(p[0], 10, 64)
	c, _ = strconv.ParseInt(p[1], 10, 64)
	return ms, c, p[2]
}
