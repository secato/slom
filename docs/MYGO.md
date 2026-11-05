# MyGo notes (as used in slom)

Module `github.com/egoist/mygo` v0.7.4, native `ui` package (GPU drawn, no webview, no cgo).
Docs: https://github.com/egoist/mygo/tree/main/docs/ui. The repo's HEAD docs can be newer than
v0.7.4; check the real API with `go doc github.com/egoist/mygo/ui <Symbol>` before using one
(`Clipboard.WriteText` is in HEAD docs but not here: use `mygo.Clipboard.Write(transfer.TextData(s))`).

## Patterns
- Immediate mode: `App.View(c *ui.Context)` rebuilds every frame from `App` state. Widgets take pointers (`ui.Switch(c, &on)`, `ui.Select(c, &label, labels)`) and report `Changed()`. Bind to a local copy of the model value, then call the model on `Changed()` (`setOption`). Select labels differ from values; the view maps label to value.
- Element values are valid for one build pass; never keep them in `App`.
- Keys: `c.Key(x)` before a stateful widget. A keyed parent scopes its children (`ui.Column(c.Key("game-"+appid))`), so editing state never leaks between games; give repeated cards their own key (`card(c, key, …)`).
- Tooltips: `Element.Tooltip(text)`, after 0.6 s hover. A disabled control shows none, so tooltips sit on the label row.
- Dialogs: `ui.Modal(c, &open, fn)`; `ui.AlertDialog(c, &open, title, msg, buttons...)` returns the index clicked. Toasts: `c.Toast(msg)` (`App.toast`, `App.notify`).
- Background work: goroutine, then `win.Update(func(){…})` (main thread, redraws). Polling that rarely changes: `mygo.RunOnMain` and `win.Invalidate()` only on change (`App.pollSteam`).
- Custom widgets are bases with our own look: `ButtonBase` (`ghost`, `choice`, game rows, apply buttons), `TabsBase` with `TabsParts.Tab(i)` (bottom border: `BorderWidth(0,0,2,0).BorderColor`). Query `Hovered()` before `Children` to restyle on hover; read `Clicked()` after `Children`.
- Layout: `ui.Grid(c).Columns(2)` with `ColumnSpan(2)`; `Row(c).Wrap()` for pill rows and command tokens; `Justify(ui.SpaceBetween)`; `ui.Scroll(c).MaxHeight(n)`; `MinWidth(0)` on a growing column so long content does not overflow.
- Look: `Gradient`, `Shadow(0,0,blur,0,color)` for glow (wrap a `Switch` in `Box(...).Radius(99)` to glow it), `Opacity(0.4)` for "not set", `ui.RichText` with `ui.Span` for a token of two colors, `ui.Icon` with an embedded SVG (`ui.ParseSVG` once) in the text color.
- Fonts are embedded (`go:embed`, OFL licenses next to them in `internal/ui/assets/`) and registered once in `setup()` with `ui.RegisterFont`. Use `Font("Rajdhani, system-ui")` so a failed registration falls back.
- Theme: at the top of every frame `View` sets `pal = a.look(c)` (dark or light palette, with the desktop's accent) and then `c.SetTheme(gamerTheme())`, a `DarkTheme()` or `LightTheme()` copy built from `pal`. `c.Theme().Dark` read before `SetTheme` is the system's appearance (MyGo recomputes the default theme every frame and follows the desktop's changes), and `c.Preferences().Accent` is the portal accent (alpha 0 = none). Colors live in `pal`, named by meaning; add a color to both palettes, not as a literal in `view.go`. Tests: the `Tester` is light unless `SetDark(true)`.
- Game rows are `ButtonBase`, not `ui.List`, because the list has section headers; they are focusable and activate with Enter or Space but have no arrow-key navigation.

## Testing with `ui.NewTester(view, w, h)`
Methods: `Click("text")`, `HasText`, `Texts`, `Find("text") (Rect{X,Y,W,H}, ok)`, `Move`, `Advance(d)` (fake clock), `Frame`, `Image()`. Text-area contents are not in `Texts()`: assert on the model.
- `Click` finds the first element with that text or label. The sidebar repeats category names, so open a tab with the `openTab` helper, not `Click("Gamescope")`.
- After changing the model directly (`a.saveAsPreset`, `a.setOption`), call `tt.Frame()` before the next `Click`: rows move, and the click would land on the old layout.
- A toast, or the tooltip of the button just pressed, covers the dock: `tt.Move(5, 5)` and `tt.Advance(10*time.Second)` before the next click there.
- A disabled button only reflects the model after a `Frame`.
