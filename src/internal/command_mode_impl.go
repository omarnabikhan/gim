package internal

import (
	"fmt"
	"io"
	"strings"

	gc "github.com/gbin/goncurses"
)

func newCommandEditorMode(baseEditor *editorImpl, cursorY int, cursorX int) *commandModeEditor {
	return &commandModeEditor{editorImpl: baseEditor, oldCursorY: cursorY, oldCursorX: cursorX}
}

type commandModeEditor struct {
	*editorImpl

	commandBuffer strings.Builder
	// We maintain the old cursor's position to update after we swap out of COMMAND mode.
	oldCursorY, oldCursorX int
}

func (ce *commandModeEditor) Handle(key gc.Key) error {
	ch := gc.KeyString(key)
	switch ch {
	case ESC_KEY:
		// Cancel the command.
		ce.commandBuffer.Reset()
		ce.userMsg = ""
		ce.swapToNormalMode()
		return nil
	case DELETE_KEY:
		// Delete the last char in the command. If the command is empty, then swap to NORMAL mode.
		if ce.commandBuffer.Len() == 0 {
			ce.userMsg = ""
			ce.swapToNormalMode()
			return nil
		}
		cmd := ce.commandBuffer.String()
		ce.commandBuffer.Reset()
		ce.commandBuffer.WriteString(cmd[:len(cmd)-1])
		ce.updateUserMsg()
		return nil
	case "enter":
		// Trim the beginning ":"
		command := ce.commandBuffer.String()
		ce.commandBuffer.Reset()
		defer func() { ce.swapToNormalMode() }()
		return ce.handleCommandEntered(command)
	default:
		// Add to command buffer and update user message.
		ce.commandBuffer.WriteString(ch)
		ce.updateUserMsg()
		return nil
	}
}

func (ce *commandModeEditor) GetCursorYX() (int, int) {
	return ce.getMaxYForContent() + 2, ce.commandBuffer.Len() + 1
}

func (ce *commandModeEditor) handleCommandEntered(command string) error {
	switch command {
	case "w":
		// Write the contents of the in-memory buffer to disc, and s
		return ce.writeToDisc()
	case "q":
		// Quit the program.
		ce.Close()
		return io.EOF
	case "debug":
		// Toggle debug mode.
		ce.verbose = !ce.verbose
		return nil
	default:
		ce.userMsg = fmt.Sprintf("unrecognized command: %s", command)
		return nil
	}
}

func (ce *commandModeEditor) swapToNormalMode() {
	// Restore the previous cursor before swapping modes.
	ce.cursorY, ce.cursorX = ce.oldCursorY, ce.oldCursorX
	ce.swapEditorMode(NORMAL_MODE)
}

func (ce *commandModeEditor) updateUserMsg() {
	// Print the command, preceded by ":"
	ce.userMsg = fmt.Sprintf(":%s\n", ce.commandBuffer.String())
}
