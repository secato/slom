# Spike: a "Test" button that launches without saving

**Decision for v1: not implemented.** The spec said to try `steam://run/<id>//<options>/`
and drop Test if it does not carry wrappers and env vars.

## Evidence
- Documentation and community sources describe the URL's `//<args>` part as launch
  parameters passed to the application, i.e. game arguments. None says Steam substitutes
  `%command%` or runs wrappers (gamescope, mangohud) or env vars from the URL; Steam applies
  those only from the saved Launch Options.
- `steam -applaunch <id> <args>` likewise passes arguments to the game.
- Launching outside Steam (running Proton ourselves) was ruled out in the spec: it would not
  match what Steam really does, so a "test" could mislead.

## Not verified empirically
Running the real experiment starts the user's Steam and touches their session, so it was
not done automatically. `scripts/spike-test-launch.sh <appid>` runs it safely (its wrapper
records what Steam passed and never starts the game). Run it with Steam open:

- If it prints `RESULT: works`, Test can be built: in `internal/ui`, a "Test" button next to
  Apply that calls `install.Start("steam://run/<id>//<url-encoded options>/")` (Steam must
  be running, the opposite of Apply), with a note that the launch is temporary and Apply
  makes it permanent.
- Otherwise nothing to do; the UI already has no Test button.
