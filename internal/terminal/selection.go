package terminal

import (
	"strings"

	"github.com/hinshun/vt10x"
)

// SelectionMode represents the current state of text selection.
type SelectionMode int

const (
	SelectionIdle SelectionMode = iota
	SelectionSelecting
	SelectionSelected
)

// Position represents a coordinate in the terminal.
type Position struct {
	Row int // Logical row (negative = scrollback, positive = live screen)
	Col int
}

// SelectionState manages mouse text selection in the terminal.
type SelectionState struct {
	Mode   SelectionMode
	Anchor Position // Where selection started (click position)
	Cursor Position // Current end of selection (drag position)
}

// NewSelectionState creates a new selection state in idle mode.
func NewSelectionState() *SelectionState {
	return &SelectionState{
		Mode: SelectionIdle,
	}
}

// Start begins a new selection at the given position.
func (s *SelectionState) Start(pos Position) {
	s.Mode = SelectionSelecting
	s.Anchor = pos
	s.Cursor = pos
}

// Update extends the selection to the new cursor position.
func (s *SelectionState) Update(pos Position) {
	if s.Mode == SelectionSelecting {
		s.Cursor = pos
	}
}

// Finish completes the selection (mouse release).
func (s *SelectionState) Finish() {
	if s.Mode == SelectionSelecting {
		// Only transition to Selected if there's an actual selection
		if s.Anchor != s.Cursor {
			s.Mode = SelectionSelected
		} else {
			// Click without drag - clear selection
			s.Mode = SelectionIdle
		}
	}
}

// Clear cancels any active selection.
func (s *SelectionState) Clear() {
	s.Mode = SelectionIdle
	s.Anchor = Position{}
	s.Cursor = Position{}
}

// IsActive returns true if there's an active or completed selection.
func (s *SelectionState) IsActive() bool {
	return s.Mode == SelectionSelecting || s.Mode == SelectionSelected
}

// Bounds returns the normalized selection bounds (start before end).
func (s *SelectionState) Bounds() (start, end Position) {
	// Normalize: start should be before end
	if s.Anchor.Row < s.Cursor.Row ||
		(s.Anchor.Row == s.Cursor.Row && s.Anchor.Col <= s.Cursor.Col) {
		return s.Anchor, s.Cursor
	}
	return s.Cursor, s.Anchor
}

// Contains checks if a position is within the selection.
func (s *SelectionState) Contains(pos Position) bool {
	if !s.IsActive() {
		return false
	}

	start, end := s.Bounds()

	// Before selection start
	if pos.Row < start.Row || (pos.Row == start.Row && pos.Col < start.Col) {
		return false
	}

	// After selection end
	if pos.Row > end.Row || (pos.Row == end.Row && pos.Col > end.Col) {
		return false
	}

	return true
}

// ExtractText extracts selected text from scrollback buffer and live screen.
// scrollbackLines: lines from scrollback buffer (oldest first)
// liveScreen: function to get a glyph from live screen (col, row)
// liveRows: number of rows in live screen
// scrollbackLen: total lines in scrollback
func (s *SelectionState) ExtractText(
	scrollbackLines [][]vt10x.Glyph,
	liveScreen func(col, row int) vt10x.Glyph,
	liveRows int,
	scrollbackLen int,
) string {
	if !s.IsActive() {
		return ""
	}

	start, end := s.Bounds()
	var result strings.Builder

	for row := start.Row; row <= end.Row; row++ {
		var line []vt10x.Glyph

		// Determine if this row is in scrollback or live screen
		// Logical row: negative = scrollback (counted from end), 0+ = live screen
		if row < 0 {
			// Scrollback: row -1 is the most recent scrollback line
			scrollbackIndex := scrollbackLen + row
			if scrollbackIndex >= 0 && scrollbackIndex < len(scrollbackLines) {
				line = scrollbackLines[scrollbackIndex]
			}
		} else {
			// Live screen
			if liveScreen != nil && row < liveRows {
				// Build line from live screen
				line = nil // Will extract char by char below
			}
		}

		// Determine column range for this row
		startCol := 0
		endCol := -1 // Will be set based on line length

		if row == start.Row {
			startCol = start.Col
		}

		if row == end.Row {
			endCol = end.Col
		}

		// Extract characters
		if row < 0 && line != nil {
			// From scrollback
			if endCol < 0 || endCol >= len(line) {
				endCol = len(line) - 1
			}
			for col := startCol; col <= endCol && col < len(line); col++ {
				ch := line[col].Char
				if ch == 0 {
					ch = ' '
				}
				result.WriteRune(ch)
			}
		} else if row >= 0 && liveScreen != nil {
			// From live screen - need to know line width
			// Assume reasonable width, trim trailing spaces
			maxCol := 200 // Reasonable max
			if endCol >= 0 && endCol < maxCol {
				maxCol = endCol + 1
			}
			var lineChars []rune
			for col := startCol; col < maxCol; col++ {
				glyph := liveScreen(col, row)
				ch := glyph.Char
				if ch == 0 {
					ch = ' '
				}
				lineChars = append(lineChars, ch)
			}
			// Trim trailing spaces for non-last rows
			if row < end.Row {
				lineChars = trimTrailingSpaces(lineChars)
			}
			for _, ch := range lineChars {
				result.WriteRune(ch)
			}
		}

		// Add newline between rows (but not after last row)
		if row < end.Row {
			result.WriteRune('\n')
		}
	}

	return result.String()
}

// trimTrailingSpaces removes trailing spaces from a rune slice.
func trimTrailingSpaces(runes []rune) []rune {
	end := len(runes)
	for end > 0 && runes[end-1] == ' ' {
		end--
	}
	return runes[:end]
}
