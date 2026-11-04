// slom edits the launch options of your installed Steam games with labeled controls:
// Proton, DLSS, gamescope, MangoHud and GameMode, with a preview of the whole command.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/store"
	slomui "github.com/secato/slom/internal/ui"
)

func main() {
	dir, err := store.DefaultDir()
	if err != nil {
		log.Fatal(err)
	}
	backups, err := store.DefaultBackupDir()
	if err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	app := slomui.New(st, backups)
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "slom — Steam launch options",
			Width:     1180,
			Height:    860,
			MinWidth:  820,
			MinHeight: 560,
			Content:   ui.View(app.View),
		})
		app.Start(win)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
