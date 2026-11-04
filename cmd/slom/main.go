// slom edits the launch options of your installed Steam games with labeled controls:
// Proton, DLSS, gamescope, MangoHud and GameMode, with a preview of the whole command.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/store"
	slomui "github.com/secato/slom/internal/ui"
)

// version is set at build time (-ldflags "-X main.version=v1.2.3"); "dev" for a plain go build.
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println("slom", version)
		return
	}
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
