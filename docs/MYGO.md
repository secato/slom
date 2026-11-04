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

## Look and feel (the gamer redesign)
- `internal/ui/theme.go`: every color is in `pal` by meaning (accent = will be applied, amber = Steam in the way, cyan = info); `gamerTheme()` forces a dark `DarkTheme()` copy with `c.SetTheme` at the top of `View`. No hex literals in `view.go`, except the header wash and the "not saved yet" pill.
- Fonts are embedded (`go:embed`, OFL, licenses in `internal/ui/assets/`) and registered once in `setup()` with `ui.RegisterFont`: Rajdhani Bold for titles (`display()`), JetBrains Mono for commands (`mono()`). Use `Font("Rajdhani, system-ui")` so a failed registration falls back. Icons are embedded SVGs parsed once with `ui.ParseSVG`, drawn with `ui.Icon` in the text color.
- Custom widgets are `ButtonBase` (`ghost`, `bigButton`, game rows), `TabsBase` + `TabsParts.Tab(i)` (bottom border via `BorderWidth(0,0,2,0).BorderColor`), `ui.Grid(c).Columns(2)` with `ColumnSpan(2)` for wide controls, `Row(c).Wrap()` for the command tokens, `ui.RichText` with `ui.Span` for a token that is part cyan, part white.
- Glow is `Shadow(0,0,blur,0,color)`; wrap the stock `Switch` in a `Box(...).Radius(99)` to give it one.
- Tester gotcha: if a test changes the model directly (`a.saveAsPreset`), call `tt.Frame()` before `Click`, because rows may have moved (the sidebar groups pending games).
- Game rows are `ButtonBase`s with `Clicked()`, not `ui.List`, because the list has section headers; they stay focusable and keyboard-activatable but have no arrow-key navigation.
- Cards: `ui.Slider(c, &float64, lo, hi)` + `Opacity(0.4)` for "not set"; `Row(c).Wrap()` for pill rows; `Row(c).Justify(ui.SpaceBetween)` for tick labels; `ColumnSpan(2)` makes a card take the whole grid row. Hover/Clicked helpers in `view.go` (`ghost`, `bigButton`) and `cards.go` (`choice`) are the pattern for custom buttons. The command box labels itself "Launch options preview: <line>" for assistive tech and `scripts/e2e.py`.
