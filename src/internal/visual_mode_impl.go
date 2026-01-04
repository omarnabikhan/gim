package internal

import (
	gc "github.com/gbin/goncurses"
)

func newVisualModeEditor(baseEditor *editorImpl, startY int, startX int) *visualModeEditor {
	return &visualModeEditor{editorImpl: baseEditor, cursorStartY: startY, cursorStartX: startX}
}

type visualModeEditor struct {
	*editorImpl
	cursorStartY, cursorStartX int
}

func (ve *visualModeEditor) Handle(key gc.Key) error {
	switch k := gc.KeyString(key); k {
	// TODO(omar): Lot's to reuse here. The movement commands for visual mode are the same as normal.
	// Let's consider a refactor where we have the movements share common code.
	case ESC_KEY:
		ve.swapEditorMode(NORMAL_MODE)
		return nil
	case "j", "down":
		// Move the cursor down.
		ve.moveCursorVertical(1)
		return nil
	case "k", "up":
		// Move the cursor up.
		ve.moveCursorVertical(-1)
		return nil
	case "l", "right":
		// Move the cursor right.
		ve.moveCursorHorizontal(1, false /*pastLastCharAllowed*/)
		return nil
	case "h", "left":
		// Move the cursor left.
		ve.moveCursorHorizontal(-1, false /*pastLastCharAllowed*/)
		return nil
	}
	return nil
}

func (ve *visualModeEditor) GetCursorYX() (int, int) {
	return ve.cursorY, ve.normalizeCursorX()
}

func (ve *visualModeEditor) GetChar(ch rune, y int, x int) gc.Char {
	// If selected, apply special highlight.
	if ve.isSelected(y, x) {
		// In bounds, apply special UI.
		return gc.A_UNDERLINE | gc.Char(ch)
	}
	return gc.Char(ch)
}

func (ve *visualModeEditor) normalizeCursorX() int {
	x := ve.cursorX
	// In INSERT mode, it's expected for the cursor to be equal to the length of the current line.
	if x > len(ve.fileContents[ve.getCurrLineInd()]) {
		// Special handling of x-position. See moveCursorInternal for details.
		x = len(ve.fileContents[ve.getCurrLineInd()])
	}
	if x < 0 {
		x = 0
	}
	return x
}

func (ve *visualModeEditor) isSelected(y int, x int) bool {
	startY, startX, endY, endX := ve.getOrderedBounds()
	if y < startY || y > endY {
		// Not in the y bounds.
		return false
	}
	// Satisfies the y-bounds, so check the x-bounds are satisfied.
	if y == startY && x < startX {
		// On the starting line, must be AFTER the x-pos.
		return false
	}
	if y == endY && x > endX {
		// On the ending line, must be BEFORE the x-pos.
		return false
	}
	// Satisfies both y-bounds and x-bounds.
	return true
}

func (ve *visualModeEditor) getOrderedBounds() (
	int /*startY*/, int, /*startX*/
	int /*endY*/, int, /*endY*/
) {
	if ve.cursorStartY < ve.cursorY {
		return ve.cursorStartY, ve.cursorStartX, ve.cursorY, ve.cursorX
	} else if ve.cursorStartY > ve.cursorY {
		return ve.cursorY, ve.cursorX, ve.cursorStartY, ve.cursorStartX
	}
	// Otherwise, sort on x-pos.
	if ve.cursorStartX < ve.cursorX {
		return ve.cursorStartY, ve.cursorStartX, ve.cursorY, ve.cursorX
	}
	return ve.cursorY, ve.cursorX, ve.cursorStartY, ve.cursorStartX
}
