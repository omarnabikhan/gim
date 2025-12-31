package internal

import "github.com/omarnabikhan/gim/src"

type EditorMode interface {
	src.Editor

	// Each mode has a different implementation of how the cursor viewed.
	GetCursorYX() (int, int)
}
