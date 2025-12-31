package internal

import (
	"fmt"

	gc "github.com/gbin/goncurses"
)

func newNormalEditorMode(baseEditor *editorImpl) *normalModeEditor {
	return &normalModeEditor{editorImpl: baseEditor}
}

type normalModeEditor struct {
	*editorImpl
}

func (ne *normalModeEditor) Handle(key gc.Key) error {
	switch k := gc.KeyString(key); k {
	case "j", "down":
		// Move the cursor down.
		ne.moveCursorVertical(1)
		return nil
	case "k", "up":
		// Move the cursor up.
		ne.moveCursorVertical(-1)
		return nil
	case "l", "right":
		// Move the cursor right.
		ne.moveCursorHorizontal(1, false /*pastLastCharAllowed*/)
		return nil
	case "h", "left":
		// Move the cursor left.
		ne.moveCursorHorizontal(-1, false /*pastLastCharAllowed*/)
		return nil
	case "0":
		// Move the cursor to the beginning of the current line.
		ne.cursorX = 0
		return nil
	case "H":
		// Move the cursor to the highest position without scrolling.
		ne.cursorY = 0
		return nil
	case "M":
		// Move the cursor to the middle of the screen without scrolling.
		ne.cursorY = ne.normalizeCursorY(ne.getMaxYForContent() / 2)
		return nil
	case "L":
		// Move the cursor to the lowest valid position without scrolling.
		ne.cursorY = ne.normalizeCursorY(ne.getMaxYForContent())
		return nil
	case "v":
		// Toggle verbose mode.
		ne.verbose = !ne.verbose
		return nil
	case "o":
		// Insert an empty line after the current line, and swap to INSERT mode.
		currLineInd := ne.getCurrLineInd()
		ne.fileContents = append(
			ne.fileContents[:currLineInd+1],
			append(
				[]string{""},
				ne.fileContents[currLineInd+1:]...,
			)...,
		)
		ne.moveCursorVertical(1)
		ne.cursorX = 0
		ne.swapEditorMode(INSERT_MODE)
		return nil
	case "O":
		// Insert an empty line before the current line, and swap to INSERT mode.
		currLineInd := ne.getCurrLineInd()
		ne.fileContents = append(
			ne.fileContents[:currLineInd],
			append(
				[]string{""},
				ne.fileContents[currLineInd:]...,
			)...,
		)
		ne.cursorX = 0
		ne.swapEditorMode(INSERT_MODE)
		return nil
	case "a":
		// Swap to INSERT mode, and increment the cursor's x-pos.
		ne.swapEditorMode(INSERT_MODE)
		// pastLastChar is allowed since we're now in INSERT mode.
		ne.moveCursorHorizontal(1, true /*pastLastCharAllowed*/)
		return nil
	case "i":
		// Swap to INSERT mode.
		ne.swapEditorMode(INSERT_MODE)
		return nil
	case ":":
		// Swap to COMMAND mode.
		ne.userMsg = ""
		ne.swapEditorMode(COMMAND_MODE)
		return nil
	default:
		// Do nothing.
		ne.userMsg = fmt.Sprintf("unrecognized key %s", k)
		return nil
	}
}

func (ne *normalModeEditor) GetCursorYX() (int, int) {
	return ne.cursorY, ne.normalizeCursorX()
}

func (ne *normalModeEditor) normalizeCursorX() int {
	x := ne.cursorX
	if x >= len(ne.fileContents[ne.getCurrLineInd()]) {
		// Special handling of x-position. See moveCursorInternal for details.
		x = len(ne.fileContents[ne.getCurrLineInd()]) - 1
	}
	if x < 0 {
		x = 0
	}
	return x
}
