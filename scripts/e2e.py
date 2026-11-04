#!/usr/bin/env python3
"""End-to-end test of the real slom window, driven through MyGo's automation plugin.

It opens a REAL window, clicks through the UI and, at the end, presses Apply, which writes
the real localconfig.vdf of the first Steam account (Steam must be closed). The script
restores the file afterwards and checks it is byte-identical. slom's own config and
backups go to a temp dir, so nothing of the user's slom data is touched.

  go build -tags e2e -o /tmp/slom-e2e ./cmd/slom
  python3 scripts/e2e.py /tmp/slom-e2e "<game name>" [appid]
"""
import glob, json, os, shutil, socket, subprocess, sys, tempfile, time

binary, game = sys.argv[1], sys.argv[2]
appid = sys.argv[3] if len(sys.argv) > 3 else None
if subprocess.run(["pgrep", "-x", "steam"], capture_output=True).returncode == 0:
    sys.exit("Steam is running: close it first (this test presses Apply)")

cfg = glob.glob(os.path.expanduser("~/.local/share/Steam/userdata/*/config/localconfig.vdf"))[0]
original = open(cfg, "rb").read()
tmp = tempfile.mkdtemp(prefix="slom-e2e-")
sock = os.path.join(tmp, "auto.sock")
env = dict(os.environ, MYGO_AUTOMATION=sock, XDG_CONFIG_HOME=tmp + "/config", XDG_STATE_HOME=tmp + "/state")
proc = subprocess.Popen([binary], env=env, stderr=subprocess.PIPE)
for _ in range(100):
    if os.path.exists(sock):
        break
    time.sleep(0.1)
else:
    proc.kill(); sys.exit("automation socket never appeared")

s = socket.socket(socket.AF_UNIX); s.connect(sock); f = s.makefile("rw")
n = 0
def call(method, **params):
    global n; n += 1
    f.write(json.dumps({"id": n, "method": method, "params": params}) + "\n"); f.flush()
    r = json.loads(f.readline())
    if "error" in r: raise RuntimeError(f"{method}: {r['error']}")
    return r["result"]
def nodes(): return call("tree")["nodes"]
def preview():
    """The launch line, read from the label of the command box ("Launch options preview: <line>")."""
    for x in nodes():
        if x.get("label", "").startswith("Launch options preview"):
            return x["label"].split(": ", 1)[1].strip()
    raise RuntimeError("no launch options preview on screen")
def find(label, role=None):
    for _ in range(50):
        for x in nodes():
            if x.get("label") == label and (role is None or x["role"] == role):
                return x
        time.sleep(0.1)
    raise RuntimeError(f"no element labeled {label!r}; have {[x.get('label') for x in nodes()][:40]}")
def press(label, role=None): call("act", id=find(label, role)["id"], action="press"); call("idle")
def check(cond, msg):
    print(("ok   " if cond else "FAIL ") + msg)
    if not cond: raise SystemExit(1)

ok = False
try:
    for _ in range(100):  # the window opens a moment after the socket
        if call("windows")["windows"]:
            break
        time.sleep(0.1)
    else:
        raise RuntimeError("the app never opened a window; stderr: " + (proc.stderr.read1(2000).decode() if proc.poll() is not None else "(still running)"))
    call("idle")
    print("window:", call("windows")["windows"][0]["title"])
    press(game)
    check(preview() != "", "selecting a game shows its launch options preview")
    for x in nodes():  # the tab label may carry a dot
        if x["role"] in ("tab", "radio") and x.get("label", "").startswith("Gamescope"):
            call("act", id=x["id"], action="press"); call("idle"); break
    press("Run inside gamescope", "switch")
    press("Always capture the mouse", "switch")
    line = preview()
    check("gamescope --force-grab-cursor --" in line, f"switches change the preview: {line!r}")
    # Copy -> clipboard
    press("Copy")
    time.sleep(0.5)
    clip = subprocess.run(["wl-paste", "-n"], capture_output=True, text=True).stdout
    check("gamescope --force-grab-cursor --" in clip and clip.rstrip().endswith("%command%"), f"Copy puts the line on the clipboard: {clip!r}")
    png = call("capture")
    open(os.path.join(tmp, "window.png"), "wb").write(__import__("base64").b64decode(png["png"]))
    print("window capture:", os.path.join(tmp, "window.png"), png["width"], "x", png["height"])
    # Apply -> real localconfig.vdf
    press("Apply " + game)  # the bolt button of the pending game in the sidebar
    time.sleep(1)
    data = open(cfg, "rb").read().decode()
    check("gamescope --force-grab-cursor -- %command%" in data, "Apply wrote the launch options into the real localconfig.vdf")
    backups = glob.glob(tmp + "/state/slom/backups/*.bak")
    check(len(backups) == 1 and open(backups[0], "rb").read() == original, "a backup of the original file was made")
    ok = True
finally:
    proc.terminate()
    try: proc.wait(5)
    except Exception: proc.kill()
    open(cfg, "wb").write(original)  # put the real file back
    check(open(cfg, "rb").read() == original, "real localconfig.vdf restored byte-identically") if ok else print("restored localconfig.vdf")
    shutil.copytree(tmp, "/tmp/slom-e2e-last", dirs_exist_ok=True)
    shutil.rmtree(tmp, ignore_errors=True)
print("E2E PASSED" if ok else "E2E FAILED")
