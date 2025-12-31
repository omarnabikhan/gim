package internal

import (
	"strings"

	gc "github.com/gbin/goncurses"
)

type insertModeEditor struct {
	*editorImpl
}

func (ie *insertModeEditor) Handle(key gc.Key) error {
	ch := gc.KeyString(key)
	switch ch {
	case ESC_KEY:
		// Swap to NORMAL model
		// Swapping decrements the x-pos by 1.
		ie.mode = NORMAL_MODE
		ie.cursorX = ie.normalizeCursorX(ie.cursorX - 1)
		ie.userMsg = ""
		return nil
	case DELETE_KEY:
		// Delete the char before the cursor.
		ie.deleteChar()
		return nil
	default:
		// Insert a char at the cursor.
		ie.insertChar(ch)
		return nil
	}
}

// Handle the user inputting the delete key.
func (ie *insertModeEditor) deleteChar() {
	currLineInd := ie.getCurrLineInd()
	currLine := ie.fileContents[currLineInd]
	if ie.cursorX == 0 && currLineInd == 0 {
		// Do nothing.
		return
	}
	if ie.cursorX == 0 {
		// If the cursor is at the beginning of the line (x-pos = 0) and not on the first line
		// (y-pos > 0), this is a special case and we:
		// 1. Copy the entire contents of that line to the previous line.
		// 2. Delete the current line (modify number of lines in file).
		// 3. Decrement the cursor's y-pos by 1.
		// 4. Update the cursor's x-pos to be whatever the end of the previous line was.
		prevLine := ie.fileContents[currLineInd-1]
		newLine := strings.Builder{}
		newLine.WriteString(prevLine)
		newLine.WriteString(currLine)
		// Replace the previous line.
		ie.fileContents[currLineInd-1] = newLine.String()
		// Remove the current line.
		ie.fileContents = append(ie.fileContents[:currLineInd], ie.fileContents[currLineInd+1:]...)
		ie.moveCursorVertical(-1)
		ie.cursorX = len(prevLine)
		return
	}
	newLine := strings.Builder{}
	newLine.WriteString(currLine[:ie.cursorX-1])
	newLine.WriteString(currLine[ie.cursorX:])
	ie.fileContents[currLineInd] = newLine.String()
	ie.moveCursorHorizontal(-1)
}

// Handle the user inputting the ch key.
func (ie *insertModeEditor) insertChar(ch string) {
	currLineInd := ie.getCurrLineInd()
	currLine := ie.fileContents[currLineInd]
	switch ch {
	case "down":
		// Move the cursor down.
		ie.moveCursorVertical(1)
		return
	case "up":
		// Move the cursor up.
		ie.moveCursorVertical(-1)
		return
	case "right":
		// Move the cursor right.
		ie.moveCursorHorizontal(1)
		return
	case "left":
		// Move the cursor left.
		ie.moveCursorHorizontal(-1)
		return
	case "enter":
		// Upon pressing the "enter" key, the current line is split before and after the x-pos of
		// the cursor, and:
		// 1. The "before" part stays on the current line.
		// 2. The "after" part (includes cursor's x-pos) is pushed to a new.
		// 3. The cursor's x-pos becomes 0.
		// 4. The cursor's y-pos is incremented by 1.
		before, after := currLine[:ie.cursorX], currLine[ie.cursorX:]
		ie.fileContents[currLineInd] = before
		ie.fileContents = append(
			ie.fileContents[:currLineInd+1],
			append([]string{after}, ie.fileContents[currLineInd+1:]...)...,
		)
		ie.cursorX = 0
		ie.moveCursorVertical(1)
		return
	}
	newLine := strings.Builder{}
	newLine.WriteString(currLine[:ie.cursorX])
	newLine.WriteString(ch)
	newLine.WriteString(currLine[ie.cursorX:])
	ie.fileContents[currLineInd] = newLine.String()
	ie.moveCursorHorizontal(1)
}
