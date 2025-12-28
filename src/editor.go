package src

import "github.com/gbin/goncurses"

// Editor - The main interface that represents the program. At any point there will be just one
// instantiation of Editor. The program will pass runes that the user provides (likely via STDIN),
// and handles the manipulation of internal state and publishing of that state (via printing to the
// console the new state of the file).
type Editor interface {
	Handle(key goncurses.Key) error
	Close()
}
