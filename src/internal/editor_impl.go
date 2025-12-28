package internal

import (
	"io"
	"os"
	"strings"

	src "text_editor"

	gc "github.com/gbin/goncurses"
)

const (
	// rw-rw-rw-
	cReadWriteFileMode = 0666

	// Dimensions.
	cMaxWidth    = 100
	cMaxHeight   = 10
	cDebugHeight = 5

	// Colors.
	cDebugColor = 1
)

func NewEditor(window *gc.Window, filePath string) (src.Editor, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR, cReadWriteFileMode)
	if err != nil {
		return nil, err
	}
	e := &editorImpl{window: window, file: file}

	gc.InitPair(cDebugColor, gc.C_RED, gc.C_BLACK)
	window.Resize(cMaxHeight+cDebugHeight, cMaxWidth)
	// Initial update of window.
	e.sync()
	return e, nil
}

type editorImpl struct {
	window *gc.Window
	file   *os.File

	// Mutable state.
	lineLengths      []int // How many chars on are on the ith line of the file.
	verbose          bool
	cursorX, cursorY int
}

var _ src.Editor = (*editorImpl)(nil)

func (e *editorImpl) Handle(key gc.Key) error {
	switch k := gc.KeyString(key); k {
	case "j":
		e.moveCursor(1 /*dy*/, 0 /*dx*/)
	case "k":
		e.moveCursor(-1 /*dy*/, 0 /*dx*/)
	case "l":
		e.moveCursor(0 /*dy*/, 1 /*dx*/)
	case "h":
		e.moveCursor(0 /*dy*/, -1 /*dx*/)
	case "0":
		e.cursorX = 0
	case "v":
		e.verbose = !e.verbose
	}
	e.sync()
	return nil
}

func (e *editorImpl) Close() {
	e.file.Close()
}

func (e *editorImpl) moveCursor(dy int, dx int) {
	cursorY, cursorX := e.window.CursorYX()
	newY, newX := cursorY+dy, cursorX+dx
	if newY < 0 || newY >= cMaxHeight || newX < 0 || newX >= cMaxWidth {
		// Don't go off-screen.
		return
	}
	if newX >= e.lineLengths[newY] {
		// Don't go past the last char in the current line.
		newX = e.lineLengths[newY] - 1
		if newX < 0 {
			newX = 0
		}
	}
	e.cursorY, e.cursorX = newY, newX
}

func (e *editorImpl) sync() {
	defer e.window.Refresh()
	defer func() {
		e.window.Move(e.cursorY, e.cursorX)
	}()
	// Will clear the STDOUT file and write whatever is viewable.
	e.clearDisplay()
	e.updateWindow()
	//e.updateWindowStub()
}

func (e *editorImpl) clearDisplay() {
	e.window.Erase()
}

func (e *editorImpl) updateWindowStub() {
	e.window.Print("dummy")
}

func (e *editorImpl) updateWindow() {
	contents := e.getFileContents()
	lineLengths := make([]int, len(contents))
	for i, row := range contents {
		e.window.Println(row)
		lineLengths[i] = len(row)
	}
	e.lineLengths = lineLengths
	e.window.Println()
	if e.verbose {
		// Print debug output.
		e.window.ColorOn(cDebugColor)
		e.window.Println("DEBUG:")
		e.window.Printf("file has %d lines\n", len(contents))
		e.window.Printf("current line has %d chars\n", len(contents[e.cursorY]))
		e.window.Printf("cursor is at (x=%d,y=%d)\n", e.cursorX, e.cursorY)
		e.window.ColorOff(cDebugColor)
	}
}

// Each string is the entire row. The row does NOT contain the ending newline.
func (e *editorImpl) getFileContents() []string {
	// Make sure file is being read from beginning.
	e.file.Seek(0 /*offset*/, io.SeekStart)
	contents, err := io.ReadAll(e.file)
	if err != nil {
		panic(err)
	}

	fileContents := make([]string, cMaxHeight)
	currRowInd := 0
	currRow := strings.Builder{}
	for _, b := range contents {
		if b == '\n' {
			// Line break, meaning we update a new row.
			fileContents[currRowInd] = currRow.String()
			currRowInd++
			currRow = strings.Builder{}
		} else {
			currRow.WriteByte(b)
		}
	}
	return fileContents
}
