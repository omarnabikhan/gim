package main

import (
	"os"
	"os/signal"
	"syscall"

	gc "github.com/gbin/goncurses"

	"text_editor/internal"
)

func main() {
	// TODO(omar): Proper flag management. For now, arg 1 is the file name to edit (arg 0 is program
	// name always).

	window, _ := gc.Init()
	defer gc.End()

	gc.Echo(false)
	gc.CBreak(true)
	gc.StartColor()

	editor, err := internal.NewEditor(window, os.Args[1])
	if err != nil {
		panic(err)
	}
	defer editor.Close()

	// Also cleanup on process exit.
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)
		<-signalChan
		editor.Close()
		gc.End()
		os.Exit(0)
	}()

	for {
		key := window.GetChar()
		if gc.KeyString(key) == "q" {
			break
		}
		err = editor.Handle(key)
		if err != nil {
			panic(err)
		}
	}
}
