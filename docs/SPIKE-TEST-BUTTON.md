# Spike: a "Test" button that launches without saving

**Result: not possible. Test is dropped for v1.** Verified by running the experiment on
2026-11-03 against a real Steam client (Proton 11 game, Steam logged in).

## What was run
`scripts/spike-test-launch.sh 3672400` asked Steam to open
`steam://run/3672400//SLOM_SPIKE=1%20<wrapper>%20%command%/`, where the wrapper only
records what it received and exits. No wrapper ran and no game started. Steam's own
`logs/console-linux.txt` shows why:

```
ExecuteSteamURL: "steam://run/3672400//SLOM_SPIKE=1%20/tmp/.../wrap.sh%20%command%/"
GameAction [AppID 3672400, ActionID 1] : LaunchApp changed task to ShowGameArgs with ""
GameAction [AppID 3672400, ActionID 1] : LaunchApp waiting for user response to ShowGameArgs "SLOM_SPIKE=1 "
```

## Findings
1. Steam treats the URL's `//<options>` as **game arguments** (`ShowGameArgs`), not as Launch
   Options: no `%command%` substitution, so wrappers (gamescope, mangohud) cannot be used.
2. Steam asks the user to **confirm** the custom arguments in a dialog before launching, so
   even plain env-style arguments would need a click in Steam on every Test.
3. The options string ends at the first `/`, so anything containing a path (a wrapper
   script) is truncated.

Launching outside Steam (running Proton ourselves) stays ruled out by the spec: it would not
match what Steam really does and could mislead.

The script stays in `scripts/` for re-checking on future Steam versions. It asks for
confirmation because, if Steam ever treated the URL options as plain arguments without a
dialog, the real game would start; use an offline game without anticheat.
