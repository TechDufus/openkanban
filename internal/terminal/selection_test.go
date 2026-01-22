package terminal

import (
	"testing"
)

func TestSelectionState_Lifecycle(t *testing.T) {
	s := NewSelectionState()

	if s.Mode != SelectionIdle {
		t.Error("new selection should be idle")
	}
	if s.IsActive() {
		t.Error("new selection should not be active")
	}

	// Start selection
	s.Start(Position{Row: 0, Col: 5})
	if s.Mode != SelectionSelecting {
		t.Error("after Start, mode should be SelectionSelecting")
	}
	if !s.IsActive() {
		t.Error("during selection, IsActive should be true")
	}

	// Update selection
	s.Update(Position{Row: 2, Col: 10})
	if s.Cursor.Row != 2 || s.Cursor.Col != 10 {
		t.Errorf("cursor should be at (2,10), got (%d,%d)", s.Cursor.Row, s.Cursor.Col)
	}

	// Finish selection
	s.Finish()
	if s.Mode != SelectionSelected {
		t.Error("after Finish with movement, mode should be SelectionSelected")
	}
	if !s.IsActive() {
		t.Error("completed selection should be active")
	}

	// Clear
	s.Clear()
	if s.Mode != SelectionIdle {
		t.Error("after Clear, mode should be SelectionIdle")
	}
	if s.IsActive() {
		t.Error("after Clear, should not be active")
	}
}

func TestSelectionState_ClickWithoutDrag(t *testing.T) {
	s := NewSelectionState()

	// Click without moving (anchor == cursor)
	s.Start(Position{Row: 1, Col: 5})
	s.Finish()

	if s.Mode != SelectionIdle {
		t.Error("click without drag should return to idle")
	}
}

func TestSelectionState_Bounds(t *testing.T) {
	s := NewSelectionState()

	// Selection from top-left to bottom-right
	s.Start(Position{Row: 1, Col: 5})
	s.Update(Position{Row: 3, Col: 15})
	s.Finish()

	start, end := s.Bounds()
	if start.Row != 1 || start.Col != 5 {
		t.Errorf("start should be (1,5), got (%d,%d)", start.Row, start.Col)
	}
	if end.Row != 3 || end.Col != 15 {
		t.Errorf("end should be (3,15), got (%d,%d)", end.Row, end.Col)
	}

	// Selection from bottom-right to top-left (reversed)
	s.Clear()
	s.Start(Position{Row: 5, Col: 20})
	s.Update(Position{Row: 2, Col: 10})
	s.Finish()

	start, end = s.Bounds()
	if start.Row != 2 || start.Col != 10 {
		t.Errorf("normalized start should be (2,10), got (%d,%d)", start.Row, start.Col)
	}
	if end.Row != 5 || end.Col != 20 {
		t.Errorf("normalized end should be (5,20), got (%d,%d)", end.Row, end.Col)
	}
}

func TestSelectionState_Contains(t *testing.T) {
	s := NewSelectionState()
	s.Start(Position{Row: 2, Col: 5})
	s.Update(Position{Row: 4, Col: 15})
	s.Finish()

	tests := []struct {
		pos      Position
		expected bool
		name     string
	}{
		{Position{Row: 2, Col: 5}, true, "start position"},
		{Position{Row: 4, Col: 15}, true, "end position"},
		{Position{Row: 3, Col: 10}, true, "middle"},
		{Position{Row: 2, Col: 10}, true, "first row after start"},
		{Position{Row: 4, Col: 10}, true, "last row before end"},
		{Position{Row: 1, Col: 10}, false, "before start row"},
		{Position{Row: 5, Col: 10}, false, "after end row"},
		{Position{Row: 2, Col: 4}, false, "start row before start col"},
		{Position{Row: 4, Col: 16}, false, "end row after end col"},
	}

	for _, tc := range tests {
		result := s.Contains(tc.pos)
		if result != tc.expected {
			t.Errorf("%s: Contains(%v) = %v, want %v", tc.name, tc.pos, result, tc.expected)
		}
	}
}

func TestSelectionState_ContainsInactive(t *testing.T) {
	s := NewSelectionState()

	if s.Contains(Position{Row: 0, Col: 0}) {
		t.Error("inactive selection should not contain anything")
	}
}

func TestSelectionState_UpdateOnlyDuringSelecting(t *testing.T) {
	s := NewSelectionState()

	// Update before starting should be no-op
	s.Update(Position{Row: 5, Col: 5})
	if s.Cursor.Row != 0 || s.Cursor.Col != 0 {
		t.Error("update before start should be no-op")
	}

	// Start and finish
	s.Start(Position{Row: 1, Col: 1})
	s.Update(Position{Row: 2, Col: 2})
	s.Finish()

	// Update after finishing should be no-op
	s.Update(Position{Row: 10, Col: 10})
	if s.Cursor.Row != 2 || s.Cursor.Col != 2 {
		t.Error("update after finish should be no-op")
	}
}
