package internal

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	gc "github.com/gbin/goncurses"

	"github.com/omarnabikhan/gim/src"
	"github.com/omarnabikhan/gim/src/internal/build_version"
)

type EditorMode string

const (
	// rw-rw-rw-
	cReadWriteFileMode = 0666

	// Colors.
	COLOR_DEFAULT = 100
	COLOR_DEBUG   = 101
	COLOR_BG      = 102

	// Color pairs.
	COLOR_PAIR_DEBUG   = 1
	COLOR_PAIR_DEFAULT = 2

	// Editor modes.
	NORMAL_MODE  EditorMode = "NORMAL"
	INSERT_MODE  EditorMode = "INSERT"
	COMMAND_MODE EditorMode = "COMMAND"

	// Escape sequences.
	ESC_KEY    = "\x1b"
	DELETE_KEY = "\x7f"
)

func NewEditor(window *gc.Window, filePath string, verbose bool) (src.Editor, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR, cReadWriteFileMode)
	if err != nil {
		return nil, err
	}

	fileContents, lengthBytes := getFileContentsAndLen(file)
	e := &editorImpl{
		window:       window,
		file:         file,
		fileContents: fileContents,
		userMsg:      fmt.Sprintf(`file "%s" %dL %dB`, file.Name(), len(fileContents), lengthBytes),
		mode:         NORMAL_MODE,
		verbose:      verbose,
	}

	// Register the supported modes.
	e.normalModeEditor = &normalModeEditor{editorImpl: e}
	e.insertModeEditor = &insertModeEditor{editorImpl: e}
	e.commandModeEditor = &commandModeEditor{editorImpl: e}

	gc.InitColor(COLOR_DEFAULT, 900, 900, 900)
	gc.InitColor(COLOR_DEBUG, 887, 113, 63)
	gc.InitColor(COLOR_BG, 170, 170, 170)

	gc.InitPair(COLOR_PAIR_DEBUG, COLOR_DEBUG, COLOR_BG)
	gc.InitPair(COLOR_PAIR_DEFAULT, COLOR_DEFAULT, COLOR_BG)

	// Initial update of window.
	e.sync()
	return e, nil
}

type editorImpl struct {
	window *gc.Window
	file   *os.File

	// Textual elements shown to user.
	fileContents  []string // Each element is a line from the source file without ending in '\n'.
	userMsg       string   // Shown to user at bottom of screen.
	commandBuffer strings.Builder

	// The cursorX is not necessarily the column which the cursor occupies. See the moveCursorHorizontal
	// function for more details.
	// The cursorY does indeed mark which row on the window that the cursor occupies.
	cursorX, cursorY int
	fileLineOffset   int // Which line of the file is being shown at the top of the screen.

	// Mode info.
	mode    EditorMode
	verbose bool

	// Different modes are implemented here.
	normalModeEditor  src.Editor
	insertModeEditor  src.Editor
	commandModeEditor src.Editor
}

var _ src.Editor = (*editorImpl)(nil)

func (e *editorImpl) Handle(key gc.Key) error {
	defer e.sync()
	switch e.mode {
	case NORMAL_MODE:
		return e.normalModeEditor.Handle(key)
	case INSERT_MODE:
		return e.insertModeEditor.Handle(key)
	case COMMAND_MODE:
		return e.commandModeEditor.Handle(key)
	default:
		return nil
	}
}

func (e *editorImpl) swapToInsertMode() {
	e.cursorX = e.normalizeCursorX(e.cursorX)
	e.mode = INSERT_MODE
	e.userMsg = "-- INSERT --"
}

func (e *editorImpl) moveCursorVertical(dy int) {
	newY := e.cursorY + dy
	numLinesInFile := len(e.fileContents)

	// Validation to prevent scrolling past file contents.
	if newY+e.fileLineOffset < 0 {
		// Nothing to scroll up to.
		return
	} else if newY+e.fileLineOffset >= numLinesInFile {
		// Nothing to scroll down to.
		return
	}

	// Handle valid scrolling. Scrolling also wipes the userMsg.
	if newY < 0 {
		// Scroll up one line.
		e.fileLineOffset -= 1
		newY = 0
		e.userMsg = ""
	} else if newY > e.getMaxYForContent() {
		e.fileLineOffset += 1
		// Keep the cursor on the same line (the last line).
		newY = e.getMaxYForContent()
		e.userMsg = ""
	}
	// Else, we're within the displayed content and can simply update the y-pos.
	e.cursorY = newY
}

// The cursor's x-position that is stored here is not the actual position the cursor occupies. Instead,
// it's treated as the max possible position it may occupy, limited by the current line's length.
// For example, say the current line has 40 chars, and the cursor's x-pos is 30. If the cursor moves
// to a line with fewer chars, say 10, the stored x-pos is still 30, even though the cursor would
// actually occupy an x-pos of 9 (the max possible on a line of length 10). This is to preserve the
// x-pos on shorter lines so that when we return to larger lines, the x-pos "pops" back to 30.
func (e *editorImpl) moveCursorHorizontal(dx int) {
	newX := e.cursorX + dx
	if newX < 0 {
		return
	}
	lineLength := len(e.fileContents[e.getCurrLineInd()])
	// Here is a difference between valid cursor x-pos in NORMAL vs INSERT mode:
	// - In NORMAL mode, the x-pos must be a valid char position: meaning, a valid offset.
	// - In INSERT mode, the x-pos must be a valid position _to insert_ a char. Practically speaking,
	//   in INSERT mode, the x-pos may be equal to the length of the current line (since we may
	//   insert a new char here).
	if newX >= lineLength && e.mode == NORMAL_MODE {
		// The newX is past the last char on the current line. That is valid (see the doc comment),
		// though we don't want to go any further than we are now.
		// So, if the x-pos is increasing, do not update it at all (set it to what it is currently).
		// Otherwise, if it's decreasing, set it to the second-to-last char on the current line. We
		// move it to the second-to-last instead of the last since the cursor is on the last char
		// from user's perspective.
		// Additionally, if the current line is empty, don't move it at all.
		if newX >= e.cursorX || lineLength == 0 {
			newX = e.cursorX
		} else {
			newX = lineLength - 2
		}
	}
	if newX > lineLength && e.mode == INSERT_MODE {
		return
	}
	e.cursorX = newX
}

// Write the contents of the in-memory file to disc.
// TODO(omar): Very simple implementation of clear the file, then overwrite full contents. We can do
// better if we know that only some small portion of the file needs to change.
func (e *editorImpl) writeToDisc() error {
	defer e.file.Sync()

	// Clear the contents of the file.
	if err := os.Truncate(e.file.Name(), 0); err != nil {
		return err
	}

	// Write the new contents of file to disc.
	e.file.Seek(0 /*offset*/, io.SeekStart)
	// We collect in a []byte and do a single write for efficiency.
	contents := bytes.Buffer{}
	for _, line := range e.fileContents {
		contents.WriteString(line)
		contents.WriteString("\n")
	}
	n, err := e.file.Write(contents.Bytes())
	if err != nil {
		return err
	}
	// Update the display to say we wrote to disc.
	e.userMsg = fmt.Sprintf("%d bytes written to disc", n)
	return nil
}

func (e *editorImpl) Close() {
	e.file.Close()
}

func (e *editorImpl) sync() {
	defer e.window.Refresh()
	defer func() {
		if e.mode == COMMAND_MODE {
			// Overwrite whatever the cursorY and cursorX are to show the cursor at the bottom of
			// the screen.
			e.window.Move(e.getMaxYForContent()+2, e.commandBuffer.Len()+1)
		} else {
			e.window.Move(e.cursorY, e.normalizeCursorX(e.cursorX))
		}
	}()
	e.updateWindow()
}

func (e *editorImpl) updateWindow() {
	// Update the window atomically by replacing it. This is more efficient than multiple Print calls
	// on the user-visible window, which may result in flashes.
	windowY, windowX := e.window.YX()
	maxY, maxX := e.window.MaxYX()
	newWindow, _ := gc.NewWindow(maxY, maxX, windowY, windowX)
	newWindow.SetBackground(COLOR_PAIR_DEFAULT)
	for i := 0; i <= e.getMaxYForContent(); i++ {
		// We reserve the bottom 2 lines for user messages, and debug messages.
		if i+e.fileLineOffset < len(e.fileContents) {
			line := e.fileContents[e.fileLineOffset+i]
			newWindow.Println(line)
		} else if (e.verbose && i < maxY-2) || (!e.verbose && i < maxY-1) {
			// There are no more file contents, so use a special UI to denote that these lines are
			// not present in the file.
			// We need to reserve either 1 or 2 lines without this UI treatment. 1 if there is no
			// debug message, 2 otherwise.
			newWindow.AttrOn(gc.A_DIM)
			newWindow.Println("~")
			newWindow.AttrOff(gc.A_DIM)
		}
	}
	if e.verbose {
		// Print debug output.
		newWindow.ColorOn(COLOR_PAIR_DEBUG)
		newWindow.Print("DEBUG: ")
		newWindow.Printf("build=%s; ", build_version.GetVersion())
		newWindow.Printf("file len=%d lines; ", len(e.fileContents))
		newWindow.Printf("curr line len=%d chars; ", len(e.fileContents[e.getCurrLineInd()]))
		newWindow.Printf("curr line offset=%d lines; ", e.fileLineOffset)
		newWindow.Printf("cursor=(x=%d,y=%d); ", e.cursorX, e.cursorY)
		newWindow.Printf("mode=%s", e.mode)
		newWindow.Println()
		newWindow.ColorOff(COLOR_PAIR_DEBUG)
	} else {
		// Print a newline anyways so no shifts when user toggles verbosity.
		newWindow.Println()
	}
	if e.mode == COMMAND_MODE {
		// Print the command, preceded by ":"
		newWindow.Printf(":%s\n", e.commandBuffer.String())
	} else {
		newWindow.Println(e.userMsg)
	}

	e.window.Erase()
	e.window.SetBackground(gc.ColorPair(COLOR_PAIR_DEFAULT))
	e.window.Overlay(newWindow)
}

func (e *editorImpl) normalizeCursorX(x int) int {
	// In INSERT mode, it's expected for the cursor to be equal to the length of the current line.
	if e.mode == NORMAL_MODE && e.cursorX >= len(e.fileContents[e.getCurrLineInd()]) {
		// Special handling of x-position. See moveCursorInternal for details.
		x = len(e.fileContents[e.getCurrLineInd()]) - 1
	}
	if x < 0 {
		x = 0
	}
	return x
}

func (e *editorImpl) normalizeCursorY(y int) int {
	if y+e.fileLineOffset >= len(e.fileContents) {
		// Special case: we ran out of file. Instead, move the cursor to the last line of the file.
		y = len(e.fileContents) - e.fileLineOffset - 1
	}
	return y
}

func (e *editorImpl) getCurrLineInd() int {
	return e.fileLineOffset + e.cursorY
}

func (e *editorImpl) getMaxYForContent() int {
	maxY, _ := e.window.MaxYX()
	// We reserve the bottom 2 lines for debug and user messages. Then we subtract 1 more since this
	// is an offset.
	return maxY - 3
}

// Each string is the entire row. The row does NOT contain the ending newline.
func getFileContentsAndLen(file *os.File) ([]string, int) {
	// Make sure file is being read from beginning.
	file.Seek(0 /*offset*/, io.SeekStart)
	contents, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	fileContents := []string{}
	currRow := strings.Builder{}
	for _, b := range contents {
		if b == '\n' {
			// Line break, meaning we update a new row.
			fileContents = append(fileContents, currRow.String())
			currRow = strings.Builder{}
		} else {
			currRow.WriteByte(b)
		}
	}
	return fileContents, len(contents)
}
