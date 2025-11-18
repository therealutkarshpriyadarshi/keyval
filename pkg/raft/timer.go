package raft

import (
	"math/rand"
	"sync"
	"time"
)

// ElectionTimer represents a randomized election timeout timer
// The timeout is randomized to reduce the likelihood of split votes
type ElectionTimer struct {
	mu      sync.Mutex
	minTime time.Duration
	maxTime time.Duration
	timer   *time.Timer
	c       chan time.Time
	running bool
	stopped bool
}

// NewElectionTimer creates a new election timer with randomized timeout
// between min and max duration
func NewElectionTimer(min, max time.Duration) *ElectionTimer {
	if min <= 0 || max <= min {
		panic("invalid timer bounds")
	}

	return &ElectionTimer{
		minTime: min,
		maxTime: max,
		c:       make(chan time.Time, 1),
		running: false,
		stopped: false,
	}
}

// Start starts the election timer
func (t *ElectionTimer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running || t.stopped {
		return
	}

	t.running = true
	timeout := t.randomTimeout()
	t.timer = time.AfterFunc(timeout, func() {
		select {
		case t.c <- time.Now():
		default:
			// Channel full, skip this tick
		}
	})
}

// Stop stops the election timer
func (t *ElectionTimer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return
	}

	t.stopped = true
	t.running = false

	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
}

// Reset resets the election timer with a new randomized timeout
func (t *ElectionTimer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return
	}

	// Stop existing timer
	if t.timer != nil {
		t.timer.Stop()
	}

	// Start new timer with randomized timeout
	timeout := t.randomTimeout()
	t.timer = time.AfterFunc(timeout, func() {
		select {
		case t.c <- time.Now():
		default:
			// Channel full, skip this tick
		}
	})

	t.running = true
}

// C returns the channel that receives timeout events
func (t *ElectionTimer) C() <-chan time.Time {
	return t.c
}

// randomTimeout returns a random timeout duration between min and max
func (t *ElectionTimer) randomTimeout() time.Duration {
	diff := t.maxTime - t.minTime
	random := time.Duration(rand.Int63n(int64(diff)))
	return t.minTime + random
}
