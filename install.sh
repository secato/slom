#!/bin/sh
# Install slom on Linux: downloads the binary of a GitHub release, checks its checksum, and
# adds a desktop entry and an icon, so slom shows up in your application menu.
#
#   curl -fsSL https://raw.githubusercontent.com/secato/slom/main/install.sh | sh
#   ./install.sh --version v0.1.0      a specific release instead of the latest
#   ./install.sh --prefix /usr/local   somewhere else than ~/.local (needs permission)
#   ./install.sh --uninstall           remove what this script installed
#
# Environment: SLOM_REPO (owner/name, default secato/slom), SLOM_BASE_URL (where the release
# files are, for mirrors and tests).
set -eu

REPO="${SLOM_REPO:-secato/slom}"
PREFIX="$HOME/.local"
VERSION="latest"
UNINSTALL=0

die() { echo "install.sh: $*" >&2; exit 1; }

while [ $# -gt 0 ]; do
	case "$1" in
	--version) [ $# -ge 2 ] || die "--version needs a tag, such as v0.1.0"; VERSION="$2"; shift 2 ;;
	--prefix) [ $# -ge 2 ] || die "--prefix needs a directory"; PREFIX="$2"; shift 2 ;;
	--uninstall) UNINSTALL=1; shift ;;
	-h | --help) sed -n '2,/^set -eu/p' "$0" | sed '$d; s/^# \{0,1\}//'; exit 0 ;;
	*) die "unknown option $1 (try --help)" ;;
	esac
done

BIN_DIR="$PREFIX/bin"
SHARE_DIR="$PREFIX/share"
BIN="$BIN_DIR/slom"
ICON="$SHARE_DIR/icons/hicolor/scalable/apps/slom.svg"
DESKTOP="$SHARE_DIR/applications/slom.desktop"

refresh_caches() {
	command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$SHARE_DIR/applications" >/dev/null 2>&1 || true
	command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t "$SHARE_DIR/icons/hicolor" >/dev/null 2>&1 || true
}

if [ "$UNINSTALL" = 1 ]; then
	rm -f "$BIN" "$ICON" "$DESKTOP"
	refresh_caches
	echo "slom removed from $PREFIX. Its settings in ~/.config/slom and its backups in ~/.local/state/slom are kept."
	exit 0
fi

[ "$(uname -s)" = "Linux" ] || die "Slom runs on Linux only"
case "$(uname -m)" in
x86_64 | amd64) ARCH=amd64 ;;
aarch64 | arm64) ARCH=arm64 ;;
*) die "no build for $(uname -m)" ;;
esac

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL --retry 2 -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -q -O "$2" "$1"; }
else
	die "needs curl or wget"
fi
if command -v sha256sum >/dev/null 2>&1; then
	check() { sha256sum -c -; }
elif command -v shasum >/dev/null 2>&1; then
	check() { shasum -a 256 -c -; }
else
	die "needs sha256sum"
fi

if [ -n "${SLOM_BASE_URL:-}" ]; then
	BASE="$SLOM_BASE_URL"
elif [ "$VERSION" = latest ]; then
	BASE="https://github.com/$REPO/releases/latest/download"
else
	BASE="https://github.com/$REPO/releases/download/$VERSION"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT INT TERM

echo "Downloading Slom ($VERSION, linux/$ARCH)..."
fetch "$BASE/slom-linux-$ARCH" "$TMP/slom-linux-$ARCH" || die "could not download $BASE/slom-linux-$ARCH"
fetch "$BASE/slom.svg" "$TMP/slom.svg" || die "could not download $BASE/slom.svg"
fetch "$BASE/checksums.txt" "$TMP/checksums.txt" || die "could not download $BASE/checksums.txt"

echo "Checking the download..."
(cd "$TMP" && grep -E " (slom-linux-$ARCH|slom\.svg)\$" checksums.txt | check >/dev/null) || die "checksum mismatch: nothing was installed"

mkdir -p "$BIN_DIR" "$(dirname "$ICON")" "$(dirname "$DESKTOP")"
install -m 755 "$TMP/slom-linux-$ARCH" "$BIN"
install -m 644 "$TMP/slom.svg" "$ICON"

# Exec needs quotes when the path has spaces.
case "$BIN" in *" "*) EXEC="\"$BIN\"" ;; *) EXEC="$BIN" ;; esac
cat >"$DESKTOP" <<DESKTOP_ENTRY
[Desktop Entry]
Type=Application
Name=Slom
GenericName=Steam launch options
Comment=Set Steam launch options with labeled controls: Proton, DLSS, gamescope, MangoHud
Exec=$EXEC
Icon=slom
Terminal=false
Categories=Game;
Keywords=steam;launch options;proton;gamescope;mangohud;dlss;
StartupWMClass=slom
DESKTOP_ENTRY
chmod 644 "$DESKTOP"
refresh_caches

echo "Installed: $BIN"
echo "Desktop entry: $DESKTOP"
case ":$PATH:" in
*":$BIN_DIR:"*) ;;
*) echo "Note: $BIN_DIR is not in your PATH; start Slom from the application menu, or add it." ;;
esac
