package main

import (
	"bufio"
	"os"
	"text_editor/internal"
)

func main() {
	editor, err := internal.NewEditor("/tmp/text_editor_test.txt")
	if err != nil {
		panic(err)
	}
	defer editor.Close()
	reader := bufio.NewReader(os.Stdin)

	for {
		ch, _, _ := reader.ReadRune()
		if ch == '\n' {
			continue
		}
		editor.Handle(ch)
	}
}
