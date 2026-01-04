package internal

func newHighlightToCursor(startY int, startX int) *Highlight {
	return &Highlight{
		startY: startY,
		startX: startX,
		endY:   -1,
		endX:   -1,
	}
}

type Highlight struct {
	startY, startX int
	// And end pos of -1 implies that the end is wherever the cursor currently is.
	endY, endX int
}
