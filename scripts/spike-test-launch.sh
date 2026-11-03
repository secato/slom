#!/usr/bin/env bash
# Spike for the "Test" button: does Steam honor wrappers and env vars passed in a
# steam://run/<appid>//<options>/ URL?
#
# It starts nothing but a tiny wrapper that records what Steam gave it and exits WITHOUT
# running the game, so no game really starts. Steam itself must already be running (open
# it first). Usage: scripts/spike-test-launch.sh <appid>
set -euo pipefail

appid="${1:?usage: $0 <appid>   (the app id of any installed game; the game will not really start)}"
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
echo "RESULT: does not work. After 40 s the wrapper never ran, so Steam did not treat the URL options as launch options with %command%."
exit 1
