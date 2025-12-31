package internal

import (
	"fmt"
	"io"

	gc "github.com/gbin/goncurses"
)

type commandModeEditor struct {
	*editorImpl
}

func (ce *commandModeEditor) Handle(key gc.Key) error {
	ch := gc.KeyString(key)
	switch ch {
	case ESC_KEY:
		// Cancel the command.
		ce.commandBuffer.Reset()
		ce.mode = NORMAL_MODE
		return nil
	case DELETE_KEY:
		// Delete the last char in the command. If the command is empty, then swap to NORMAL mode too.
		if ce.commandBuffer.Len() == 0 {
			ce.mode = NORMAL_MODE
			return nil
		}
		cmd := ce.commandBuffer.String()
		ce.commandBuffer.Reset()
		ce.commandBuffer.WriteString(cmd[:len(cmd)-1])
		return nil
	case "enter":
		// Trim the beginning ":"
		command := ce.commandBuffer.String()
		ce.commandBuffer.Reset()
		defer func() { ce.mode = NORMAL_MODE }()
		return ce.handleCommandEntered(command)
	default:
		// Just add to command buffer.
		ce.commandBuffer.WriteString(ch)
		return nil
	}
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
	default:
		ce.userMsg = fmt.Sprintf("unrecognized command: %s", command)
		return nil
	}
}
