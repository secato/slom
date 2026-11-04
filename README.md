# Slom - Steam launch options manager

**Set Steam launch options without memorizing them.**

Pick a game, flip switches for Proton, DLSS, gamescope, MangoHud and GameMode, and see the
whole command line as you go. Hover any option for a plain-English explanation. Slom writes
the result into Steam's launch options for you.

![slom](docs/screenshot.png)

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/secato/slom/main/install.sh | sh
```

Downloads the latest release for your CPU, checks its checksum, and adds Slom to your
application menu. `./install.sh --uninstall` removes it again.

## Build it yourself

```sh
go build -o slom ./cmd/slom
./slom
```

Needs Go 1.27+ and Linux (glibc). Close Steam to apply, or use Force apply and Slom closes and reopens it for you.
Slom backs up `localconfig.vdf` before every change.

- **Presets**: save a setup like "Native 1440p 165Hz" once and use it on many games.
- **Imports** what is already in Steam; anything it doesn't know is kept as-is.
- **Copy** the line instead, if you'd rather paste it into Steam yourself.

MIT licensed: copy it, change it, do what you like. Built with [MyGo](https://github.com/egoist/mygo). Working on it with an AI agent? Start at
[AGENTS.md](AGENTS.md).
