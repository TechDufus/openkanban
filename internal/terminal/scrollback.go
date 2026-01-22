package terminal

import (
	"sync"

	"github.com/hinshun/vt10x"
)

// ScrollbackBuffer is a ring buffer for storing terminal scrollback history.
// It stores lines that have scrolled off the top of the terminal screen.
type ScrollbackBuffer struct {
	lines    [][]vt10x.Glyph // Circular buffer of lines
	head     int             // Index where next line will be written
	count    int             // Number of lines currently stored
	capacity int             // Maximum number of lines
	mu       sync.RWMutex
}

// NewScrollbackBuffer creates a new scrollback buffer with the given capacity.
func NewScrollbackBuffer(capacity int) *ScrollbackBuffer {
	if capacity <= 0 {
		capacity = 10000 // Default
	}
	return &ScrollbackBuffer{
		lines:    make([][]vt10x.Glyph, capacity),
		capacity: capacity,
	}
}

// Push adds a line to the scrollback buffer.
// If the buffer is full, the oldest line is overwritten.
func (sb *ScrollbackBuffer) Push(line []vt10x.Glyph) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Copy the line to avoid holding references to the original
	lineCopy := make([]vt10x.Glyph, len(line))
	copy(lineCopy, line)

	sb.lines[sb.head] = lineCopy
	sb.head = (sb.head + 1) % sb.capacity

	if sb.count < sb.capacity {
		sb.count++
	}
}

// Len returns the number of lines in the buffer.
func (sb *ScrollbackBuffer) Len() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.count
}

// Get returns the line at the given index (0 = oldest line in buffer).
// Returns nil if index is out of bounds.
func (sb *ScrollbackBuffer) Get(index int) []vt10x.Glyph {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if index < 0 || index >= sb.count {
		return nil
	}

	// Calculate actual position in circular buffer
	// If buffer is full, oldest line is at head
	// If buffer is not full, oldest line is at 0
	var actualIndex int
	if sb.count < sb.capacity {
		actualIndex = index
	} else {
		actualIndex = (sb.head + index) % sb.capacity
	}

	return sb.lines[actualIndex]
}

// GetRange returns lines from startIndex to endIndex (exclusive).
// Useful for rendering a viewport of scrollback history.
func (sb *ScrollbackBuffer) GetRange(startIndex, endIndex int) [][]vt10x.Glyph {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > sb.count {
		endIndex = sb.count
	}
	if startIndex >= endIndex {
		return nil
	}

	result := make([][]vt10x.Glyph, endIndex-startIndex)
	for i := startIndex; i < endIndex; i++ {
		var actualIndex int
		if sb.count < sb.capacity {
			actualIndex = i
		} else {
			actualIndex = (sb.head + i) % sb.capacity
		}
		result[i-startIndex] = sb.lines[actualIndex]
	}

	return result
}

// Clear removes all lines from the buffer.
func (sb *ScrollbackBuffer) Clear() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	for i := range sb.lines {
		sb.lines[i] = nil
	}
	sb.head = 0
	sb.count = 0
}

// Capacity returns the maximum number of lines the buffer can hold.
func (sb *ScrollbackBuffer) Capacity() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.capacity
}
