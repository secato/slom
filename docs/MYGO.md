# MyGo notes (as used in slom)

Module `github.com/egoist/mygo` v0.7.4, native `ui` package (GPU drawn, no webview, no cgo).
Official docs: https://github.com/egoist/mygo/tree/main/docs/ui. The repo's HEAD docs can be
newer than v0.7.4; when in doubt use `go doc github.com/egoist/mygo/ui <Symbol>`
(example: `Clipboard.WriteText` is in HEAD docs but not in v0.7.4: use
`mygo.Clipboard.Write(transfer.TextData(s))`).

## Patterns used
- Immediate-mode: `App.View(c *ui.Context)` rebuilds every frame from `App` state. Widgets take pointers (`ui.Switch(c, &on)`, `ui.Select(c, &label, labels)`) and report `Changed()`.
- Keys: `c.Key(x)` before a stateful widget; a keyed parent scopes its children (`ui.Column(c.Key("game-"+appid))`) so editing state never leaks between games.
- Selects/switches bind to local copies of values from the model, then `Changed()` calls the model (`setOption`). Select labels differ from values; the view maps label -> value.
- Tooltips: `Element.Tooltip(text)`; they show after 0.6 s hover. A disabled control does not show one, so tooltips sit on the label row.
- `ui.List(c, &state, n, rowFn)` with `state.Key`, `state.Selected = &int`, `Changed()` for selection.
- Modals: `ui.Modal(c, &open, fn)`, `ui.AlertDialog(c, &open, title, msg, buttons...) int` (index of the button clicked).
- Background work: goroutine + `win.Update(func(){...})` (runs on the main thread, redraws). Polling that rarely changes: `mygo.RunOnMain` + `win.Invalidate()` only on change (see `App.pollSteam`).
- Display size: `mygo.Screen.PrimaryDisplay()` (DIPs; multiply by `ScaleFactor` for pixels). Refresh rate is not exposed, so `internal/ui/system.go` asks `hyprctl monitors -j`.

## Testing without a window
`ui.NewTester(view, w, h)`: `Click("text")`, `HasText`, `Texts`, `Find("text") (Rect{X,Y,W,H}, ok)`, `Move`, `Advance(d)` (fake clock; tooltips need ~0.6 s), `Frame`, `Image()` (the last frame as `*image.RGBA`). Text-area contents are not in `Texts()`; assert on the model.
`TestScreenshot` renders `docs/screenshot.png`: `SLOM_SCREENSHOT=$PWD/docs/screenshot.png go test ./internal/ui -run Screenshot` (needs an absolute path, tests run in the package dir).

## Gotchas
- Element values are valid for one build pass only; never store them in `App`.
- `Segmented` labels are also the click targets in tests (`tt.Click("Gamescope •")`, the dot marks groups with settings).
- Run `go tool`-style `mygo vet` is not set up here; `go vet ./...` is the check.
