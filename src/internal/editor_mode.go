package internal

import (
	gc "github.com/gbin/goncurses"

	"github.com/omarnabikhan/gim/src"
)

type EditorMode interface {
	src.Editor

	// Each mode has a different implementation of how the cursor viewed.
	GetCursorYX() (int, int)

	GetChar(ch rune, y int, x int) gc.Char
}
