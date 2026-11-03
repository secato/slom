#!/usr/bin/env bash
# Spike for the "Test" button: does Steam honor wrappers and env vars passed in a
# steam://run/<appid>//<options>/ URL?
#
# WARNING: this asks Steam to launch a game. The wrapper records what Steam gave it and exits
# WITHOUT running the game, but that only happens if Steam honors the wrapper. If Steam instead
# treats the URL options as plain game arguments, THE REAL GAME STARTS (pick an offline game,
# never one with anticheat). Steam may also show a confirmation dialog for custom-argument
# launches; then the result is inconclusive (answer the dialog and rerun). Steam itself must
# already be running. Usage: scripts/spike-test-launch.sh <appid>
set -euo pipefail

appid="${1:?usage: $0 <appid>   (an installed, offline game with no anticheat; it may really start)}"
dir="$(mktemp -d)"
marker="$dir/marker"
wrapper="$dir/wrap.sh"
trap 'rm -rf "$dir"' EXIT

cat >"$wrapper" <<WRAP
#!/usr/bin/env bash
{
  echo "wrapper ran"
  echo "SLOM_SPIKE=\${SLOM_SPIKE:-<unset>}"
  echo "args: \$*"
} >"$marker"
exit 0
WRAP
chmod +x "$wrapper"

read -r -p "This may really start app $appid. Continue? [y/N] " ok
[[ "$ok" == [yY]* ]] || { echo "Aborted."; exit 3; }

if ! pgrep -x steam >/dev/null; then
  echo "Steam is not running. Start it, then run this again." >&2
  exit 2
fi

# What the Test button would send: env var + wrapper + %command% (URL-encoded spaces).
opts="SLOM_SPIKE=1%20$wrapper%20%command%"
echo "Asking Steam to run app $appid with: SLOM_SPIKE=1 $wrapper %command%"
steam "steam://run/$appid//$opts/" >/dev/null 2>&1 &

for _ in $(seq 1 40); do
  if [[ -s "$marker" ]]; then
    echo "--- marker written by the wrapper ---"
    cat "$marker"
    if grep -q '^SLOM_SPIKE=1$' "$marker" && grep -q '^args: .\+' "$marker"; then
      echo "RESULT: works. Steam substituted %command%, ran the wrapper and passed the env var."
    else
      echo "RESULT: partial. The wrapper ran but not everything was honored (see above)."
    fi
    exit 0
  fi
  sleep 1
done
echo "RESULT: no wrapper ran within 40 s. Either Steam did not treat the URL options as launch options with %command% (the game may have started with them as plain arguments: close it), or a confirmation dialog is waiting: answer it and rerun before concluding."
exit 1
