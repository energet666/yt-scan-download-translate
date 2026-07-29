# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Long-running service that polls YouTube playlists every minute, downloads new videos via `yt-dlp`, and (optionally per playlist) overlays a Russian voice-over track fetched by `vot-cli-live`, muxed with `ffmpeg`.

## Commands

```bash
go run .                                  # run from source (also does first-run setup)
go build -o yt-scan-download-translate .  # build the binary install_service.sh expects
go vet ./...
gofmt -l .
```

There are no tests in this repository.

`vendor/` is **gitignored but present locally**, so local builds silently run in `-mod=vendor` mode while a fresh clone resolves `github.com/energet666/goytdlp` from the network. Consequences: after changing `go.mod` locally, run `go mod vendor` or the build fails with an inconsistent-vendor error; and edits made inside `vendor/` affect local builds only and are never committed.

`install_service.sh` (gitignored, root-only, Linux) generates a systemd unit from the built binary. It is not usable on macOS.

## Runtime prerequisites

`yt-dlp`, `ffmpeg`, and `vot-cli-live` must be on `PATH` — `checkDependencies()` in [main.go](main.go) exits with status 1 at startup if any is missing.

## Working directory is significant

Every path in the program is relative to the process CWD, which must be the repo root:

- `templates/` — read-only seed files, must exist next to the binary.
- `.private/` — live state, seeded from `templates/` on first run by `createConfigsIfNotExist()`; gitignored. Holds `config.json` (playlists), `downloaded.json` (processed video IDs), `yt-dlp.conf`.
- `download/`, `download_tmp/` — output and temp dirs, set by `--paths` inside `templates/yt-dlp.conf`, themselves relative to CWD.

Running from a subdirectory (e.g. `ffmpeg/`) breaks all of these. Note the systemd unit sets `WorkingDirectory` for exactly this reason.

## Architecture

Single binary, three packages:

- `main` — polling loop, config/state I/O, first-run interactive wizard.
- `votclilive` — wraps the `vot-cli-live` CLI (5-minute timeout).
- `ffmpeg` — wraps one `ffmpeg` invocation.
- External `github.com/energet666/goytdlp` wraps `yt-dlp` (`ScanPlaylist`, `Download`, `GetFilename`).

**Flow:** `main()` → `runScanCycle(ctx)` once per minute → for each playlist, `ScanPlaylist` → for each video not in `downloaded.json`, `handleVideo()`.

**State:** `downloaded.json` is rewritten immediately after each successful video, not at cycle end, so a crash never causes a re-download. It is the only dedup mechanism — deleting it re-downloads everything in every configured playlist.

**Cancellation:** `signal.NotifyContext` (SIGINT/SIGTERM) is checked between playlists and between videos, but `handleVideo` takes no context — an in-flight download or ffmpeg run completes before shutdown.

### handleVideo translation state machine

This is the non-obvious part of the codebase ([main.go](main.go)):

1. Download the video. If the playlist has `translate: false`, stop here — the original stays in `download/`.
2. Call `votclilive.Download(..., "live")` up to `maxLiveRetries` (3) times. Success requires *both* a nil error *and* the `.mp3` actually existing on disk — `vot-cli-live` can exit 0 without producing output, so the `os.Stat` check is load-bearing. Partial `.mp3` files are removed between attempts.
3. If all live attempts fail, fall back once to `"tts"` voice style.
4. `ErrNoSpeech` at any point short-circuits: no translation, no ffmpeg, original video kept, and the video is still marked as processed.
5. `ffmpeg.AddAudio` muxes original audio at `volume=0.2` with the translated track at `1.0` (video stream copied, audio re-encoded to AAC). The `amix` **must** keep `normalize=0` — the default `normalize=1` divides every input by the input count, which silently costs 6 dB on the translated voice; `alimiter` after the mix is what makes `normalize=0` safe from clipping.
6. On success both the `.mp3` and the **original video file are deleted** — only the muxed output survives.

Output filename encodes which path was taken: `<original>.[VOT-CLI-LIVE].mp4` or `<original>.[VOT-CLI-TTS].mp4`.

### Known fragilities

- No-speech detection greps `vot-cli-live` output for the literal Russian substring `"нет речи"` across combined stdout+stderr ([votclilive/download.go](votclilive/download.go)). Any upstream wording or locale change silently turns "no speech" into a generic failure, which then burns all 3 retries plus the TTS fallback.
- `goytdlp.ScanPlaylist` does **not** pass `--config-location`, unlike `Download` and `GetFilename`. Options in `.private/yt-dlp.conf` (cookies, proxy, auth) therefore do not apply to playlist scanning.
- Playlists and videos are processed strictly sequentially; one slow video blocks the whole cycle.

## Conventions

- Logging is `slog` with a text handler at `LevelDebug`; log messages and CLI prompts are English, while some inline code comments and all docs (`README.md`, `IMPROVEMENTS.md`) are Russian. Match the surrounding file.
- CLI wrappers build `[]string` args explicitly and stream child stdout/stderr to the parent's, rather than shelling out to a string.

[IMPROVEMENTS.md](IMPROVEMENTS.md) tracks the roadmap as a checklist; the notable unchecked items are goroutine-based parallelism and command-line flags.
