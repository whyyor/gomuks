# whyyor's gomuks terminal fork

Personal fork of [gomuks](https://github.com/gomuks/gomuks), focused on the
**terminal client** (`gomuks-terminal`, the TUI in `tui/`). It talks to a local
gomuks backend daemon, which syncs with Beeper (WhatsApp, Telegram, Slack and
Instagram bridges). This file is the context needed to rebuild, install and
keep changing it, written for a coding agent as much as for a human.

## Layout on this machine

| What | Where |
|---|---|
| Source (this repo) | `~/Projects/gomuks` |
| Installed TUI | `~/.local/opt/gomuks/gomuks-terminal` |
| Launch command | `gomuks-tui` → symlink to the installed TUI; usually opened with `hgo msg` (`~/Configration/scripts/hgo`) |
| Backend daemon | `~/.local/opt/gomuks/gomuks`, **upstream build `2ae6cc06` (Aug 10), not built from this fork** |
| Daemon service | launchd agent `org.nixos.gomuks`, defined in `~/Configration/nix/flake.nix` |
| TUI config | `~/Library/Application Support/gomuks/terminal.yaml` |
| Daemon DB | `~/Library/Application Support/gomuks/gomuks.db` (read-only queries are fine) |
| Logs | `~/Library/Logs/gomuks/` (`terminal.log`, `gomuks.log`) |
| Notifier bundles | `~/.local/opt/gomuks/notifiers/*.app` (see below) |
| Notifier icon sources | `~/Configration/icons/gomuks-notifiers/*.png` |
| Terminal | Ghostty, config `~/Configration/ghostty/config` |

## Branches and remotes

- `origin` = upstream `gomuks/gomuks`. Its push URL is set to `no_push`: **never push upstream.**
- `fork` = `git@github.com:whyyor/gomuks.git`.
- `fix/rpc-reconnect` (checked out) and `keshav-dev` carry the fork's work and are
  kept at the same commit. After committing:

  ```sh
  git push                                   # fix/rpc-reconnect -> fork
  git branch -f keshav-dev HEAD && git push fork keshav-dev
  ```

- Try risky changes on a separate branch first, and merge only after the user has tested them.

## Rebuild and install the TUI

Always build with **`-tags goolm`** (pure-Go olm; libolm headers aren't
installed, so a build without it fails). `build-terminal.sh` does **not** pass
that tag, so use this instead:

```sh
cd ~/Projects/gomuks
COMMIT=$(git rev-parse HEAD)
MAUTRIX_VER=$(go list -m maunium.net/go/mautrix | awk '{print $2}')
BT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
go build -tags goolm -ldflags "-s -w \
  -X 'go.mau.fi/gomuks/version.Commit=$COMMIT' \
  -X 'go.mau.fi/gomuks/version.BuildTime=$BT' \
  -X 'maunium.net/go/mautrix.GoModVersion=$MAUTRIX_VER'" \
  -o /tmp/gomuks-terminal-new ./cmd/gomuks-terminal

cd ~/.local/opt/gomuks
cp gomuks-terminal gomuks-terminal.prev        # rollback copy
mv /tmp/gomuks-terminal-new gomuks-terminal
go version -m gomuks-terminal | grep -E "vcs.revision|vcs.modified|-tags"
```

Then restart the TUI (`hgo msg`). Rollback: `cp gomuks-terminal.prev gomuks-terminal`.
Commit before building so `vcs.modified=false` and the binary maps to a commit.

Restarting the daemon (rarely needed): `launchctl kickstart -k gui/$(id -u)/org.nixos.gomuks`.

## Checks before every install

```sh
gofmt -l tui/ pkg/
go vet  -tags goolm ./tui/... ./pkg/rpc/...
go test -tags goolm ./tui/... ./pkg/rpc/...
# notification code has per-OS files; make sure the others still compile
GOOS=linux   go vet -tags goolm ./tui/lib/notification/
GOOS=windows go vet -tags goolm ./tui/lib/notification/
```

## Per-network notification icons

macOS takes a notification's icon and app name from the **sending app bundle**;
there is no per-notification override (terminal-notifier 3.x dropped `-appIcon`
and `-sender`). So each network gets its own copy of terminal-notifier with its
own icon and bundle ID. `tui/lib/notification/notify_darwin.go` picks
`notifiers/<Network>.app`, then `notifiers/gomuks.app`, then brew's
`terminal-notifier`.

Rebuild one (from terminal-notifier source at tag `3.1.0`):

```sh
git clone --depth 1 --branch 3.1.0 https://github.com/julienXX/terminal-notifier.git
cd terminal-notifier
make icon ICON=~/Configration/icons/gomuks-notifiers/WhatsApp.png APP_NAME=WhatsApp
cp -R build/WhatsApp.app ~/.local/opt/gomuks/notifiers/
```

Bundle names are exactly `WhatsApp`, `Telegram`, `Slack`, `Instagram`, `gomuks`
(mapping in `sourceBundles`). Icons are Nerd Font glyphs (JetBrainsMono Nerd
Font, non-Mono variant) in brand colours on a `#1e1f1c` rounded tile, rendered
with ImageMagick.

**Permission gotcha:** a new bundle needs notification permission. Never trigger
its first notification from a background (agent) shell: `launchctl managername`
there says `Background`, macOS can't show the prompt, and it records a refusal.
Launch it into the GUI session with `open -n <bundle>.app --args -title … -message …`,
or have the user enable it in System Settings → Notifications. `tccutil reset
UserNotification` does not work for this on macOS 15.

## Ghostty keybinds the TUI depends on

tcell 2.9 can't parse CSI-u (kitty keyboard) sequences, and some chords have no
legacy encoding, so Ghostty translates them into bytes the TUI handles:

| Ghostty keybind | Sends | TUI action |
|---|---|---|
| `ctrl+backspace`, `alt+backspace` | `\x1b\x7f` | delete word |
| `shift+enter` | `\n` (arrives as `KeyLF`) | newline in composer |
| `ctrl+semicolon` | `\x1f` (`KeyCtrlUnderscore`) | toggle visual mode |
| `ctrl+m` | `\x1e` (`KeyCtrlCarat`); Ctrl+M is byte-identical to Enter | mark room read |
| `macos-option-as-alt = left` | | Option works as Alt |

After editing the Ghostty config, the user reloads with Cmd+R.

## Feature map (where things live)

- **Connection:** auto-reconnect and manual resync (`Ctrl+r`) in `pkg/rpc/websocket.go`;
  a centered status dialog shows reconnecting / resyncing / "sync erroring"
  (the daemon's homeserver sync health) in `tui/view-main.go`.
- **Rendering:** `tui/framebuffer.go` sends tcell only the cells that changed. That
  fixed cursor flicker: mauview's `Flex` clears the whole screen every frame,
  which made tcell rewrite ~17 KB per keystroke. Don't remove it;
  `always_clear_screen` is forced off because of it.
- **Images:** inline images and stickers via kitty graphics unicode placeholders
  (`tui/messages/kittyimg.go`), half-block fallback; previews downscaled to 1024px;
  animated webp stickers use an ffmpeg first-frame fallback.
- **Media actions:** `tui/media-actions.go`: play (mpv/ffplay/afplay; video in the terminal
  via `mpv --vo=kitty`; link posts such as Instagram reels go to IINA), open, download.
  Playback stops when leaving the room. Uploads: `tui/upload.go` (`/upload`, `/paste`, Ctrl+V with an image).
- **Visual mode** (`Ctrl+;` or `Alt+v`, then `j`/`k`): verbs `r` reply, `p` play, `o` open,
  `d` download, `y` copy, `u` copy URL, `e` edit, `x` redact, `1`–`6` quick react.
- **Room list and switcher:** brand-coloured bridge icons (`tui/bridgeicon.go`), unread badges,
  switcher ranked by match quality then recency (`tui/roomrank.go`), fzf-style layout.
- **Composer:** emoji `:shortcode:` with Tab completion (`tui/lib/emoji`), `@mention` completion,
  message editing, Option-word editing, arrows switch rooms at the input's edges.
- **Privacy:** typing is shown but not sent (`disable_typing_notifs: true`). Read receipts
  go out automatically when you open a room or press any key while scrolled to the bottom
  (`MainView.BumpFocus` → `MarkRead`); `Ctrl+M` sends one explicitly from anywhere. Deleted
  messages stay visible with a `(deleted)` marker; incoming read receipts show as `✓ seen`.
- **Look:** Monokai palette (`tui/theme.go`); sender colours from ANSI palette
  colours with your own name in white (`tui/widget/color.go`).

## Lessons (don't repeat these)

- **Don't upgrade tcell past 2.9 casually.** 2.13 was tried and reverted at the user's
  request. It renumbers Ctrl keys and gives them lowercase runes, so every Ctrl binding
  stops matching cbind's parsed keys. It merges both Backspace codes, so with
  `backspace1_removes_word` every Backspace deletes a word. A raw LF arrives as
  `KeyCtrlJ`, and the kitty keyboard protocol switches on.
- **Don't disable mouse reporting.** With it off, Ghostty turns the scroll wheel
  into arrow keys, which switch rooms. Tried and reverted.
- **Async state needs an explicit `Render()`.** Receipts, typing, resync and image
  downloads all showed up late until they called it; mauview only repaints on
  input and sync batches.
- **Check real data before writing rules.** Beeper DMs have 3 joined members (the
  bridge bot counts), so use one hero = DM. Bridges lie about msgtype (reels arrive
  as `m.image`), so prefer the mimetype.
- **Measure before claiming a fix.** For rendering issues, capture the bytes the TUI
  writes with an `expect` script that spawns the TUI in a pty (set `TERM=xterm-ghostty`,
  and drain output with `expect -re`, not `sleep`). Open only **Telegram Saved Messages**
  in test runs: opening a real chat sends a read receipt. Delete capture logs and any
  config copies afterwards; `terminal.yaml` contains the account password.
- Many TUI features upstream were commented-out stubs (message selection, editing,
  tab completion, download/open, typing, preferences loading). If a feature
  "does nothing", check whether the code behind it is actually live.
