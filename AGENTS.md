# AGENTS.md — althosts

A small, local-only CLI that manages multiple `/etc/hosts` profiles and
switches between them safely. Everything stays local — no remote sync, daemons, or GUI.

macOS note: althosts uses the normal `sudo` flow. README documents Apple's
`/etc/pam.d/sudo_local` Touch ID setup for sudo, but althosts must not modify
that system-wide sudo configuration itself.

## Stack

- Language: **Go 1.26+** (`go.mod`: `module github.com/xshoji/althosts`)
- CLI framework: **cobra** (`github.com/spf13/cobra`)
- Config / combined files: **YAML** (`gopkg.in/yaml.v3`)
- State file: **JSON** (`encoding/json`)
- Target OS: **macOS first**, Linux works, Windows path is wired but not exercised

## Layout

```
cmd/althosts/main.go        # entry point — only calls cli.NewRootCmd().Execute()
internal/
  cli/                      # cobra subcommands (1 file per command)
    root.go                 # NewRootCmd: wires --home, builds subcommands via mk()
    context.go              # ctx <-> *app helpers (withApp / appFrom)
    elevate.go              # self-re-exec under sudo (ALTHOSTS_ELEVATED guard)
    init.go list.go create.go edit.go show.go diff.go validate.go
    use.go doctor.go
  home/                     # resolves althosts home, exposes dir layout
  config/                   # config.yaml load/save (Config struct + defaults)
  state/                    # state.json (active profile, applied_at, hosts_path)
  profile/                  # Store: list/create/load + render (plain & combined)
  hostsfile/                # atomic hosts file write + writability probe
  dns/                      # OS DNS cache flush (darwin only; no-op elsewhere)
  editor/                   # spawn $VISUAL / $EDITOR / vi
  validate/                 # hosts-file lint (warning / error findings)
```

Runtime layout under `~/.althosts/` (or `$ALTHOSTS_HOME`, or `--home`):

```
profiles/<name>.hosts        # plain profiles
combined/<name>.yaml         # combined definitions (members: [<plain>, ...])
state.json
config.yaml
```

Home resolution priority (see `internal/home/home.go`):
**`--home` flag > `$ALTHOSTS_HOME` > `~/.althosts`.**

## Core invariants — DO NOT BREAK

- **Atomic writes only.** Always go through `hostsfile.Write` (temp file in
  the same directory, then `rename`). Resolves symlinks first
  (`/etc/hosts` → `/private/etc/hosts` on macOS).
- **State is written last.** Update `state.json` only after a successful
  apply.
- **Self-elevation, not auto-sudo.** When `hostsfile.EnsureWritable` returns
  `hostsfile.ErrPermission` and we are not already root and not already
  re-execed (`alreadyElevated()` checks `ALTHOSTS_ELEVATED`), call
  `reExecWithSudoSameArgs(cmd)`. Never silently shell out to `sudo` for
  arbitrary commands.
- **Combined profiles are flat.** `profile.Render` rejects combined members
  whose kind is also `KindCombined`. Do not introduce nesting.
- **Validation gating.** `use` runs `validate.Hosts` and refuses to apply
  when any finding is `SeverityError` unless `--force` is passed. Warnings
  print but do not block.
- **DNS flush is best-effort and platform-gated.** `dns.Flush` is darwin-only
  today; it must remain a silent no-op on other GOOS. Errors must not fail
  the apply.
- **Touch ID for sudo is documentation-only.** README may point users to
  Apple's `/etc/pam.d/sudo_local` setup, but althosts should not write or manage
  that system-wide sudo configuration.
- **Profile names** are validated by `profile.ValidateName` — no
  ``/\\:*?"<>|`` and no leading `.`. Reuse it for any new code that accepts a
  name from the user.
- **No network calls. No telemetry. No background processes.** Strictly
  local.

## How a subcommand is wired

`internal/cli/root.go` builds each command through `mk(build)`:

1. `build(nil)` is called once to register the command's flags / `Use`.
2. The command's `RunE` is wrapped so that, at execution time,
   `newApp(homeFlag)` resolves the home and loads `config.yaml`, then stores
   `*app` in the cobra context via `withApp`.
3. Inside the command, recover it with `appFrom(cmd.Context())`. Call
   `a.requireHome()` first if the command needs an existing althosts home.

When adding a new subcommand:

- Put it in `internal/cli/<name>.go`, exporting `new<Name>Cmd(_ *app) *cobra.Command`.
- Register it in the `root.AddCommand(...)` block in `root.go`.
- Pull dependencies (`store`, `cfg`, `home`) off `*app`, never construct your
  own `home.Home` / `profile.Store`.
- Use `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` for all output (tests rely
  on this).

## Build, test, run

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .              # must print nothing

go build -o althosts ./cmd/althosts
./althosts --home ./testdata init
./althosts --home ./testdata list
```

For commands that touch `/etc/hosts` (`use`), point `hosts_path` in
`config.yaml` (or use a fresh `--home`) at a throwaway file when developing —
do **not** run `sudo go run` against the real `/etc/hosts`.

Targeted tests live next to their package:

- `internal/validate/validate_test.go`
- `internal/profile/render_test.go` (+ `testhelp_test.go`)

Add tests for new behavior in the same package.

## Coding conventions

- Standard Go formatting (`gofmt`); run `go vet` before sending changes.
- Keep packages small and single-purpose; mirror the existing split
  (no kitchen-sink `util` / `common` packages).
- Wrap errors with `fmt.Errorf("...: %w", err)` and define sentinel errors
  in the package that owns them (see `profile.ErrNotFound`,
  `hostsfile.ErrPermission`).
- All file writes use mode `0o644`; directories `0o755`.
- Public identifiers carry a doc comment starting with the identifier name.

## Don'ts

- Don't add a daemon, background goroutine, or watcher.
- Don't introduce remote sync, HTTP clients, or telemetry.
- Don't shell out to `sed`, `awk`, or other external tools when stdlib will
  do; the only sanctioned external commands are `sudo`, `dscacheutil`,
  `killall mDNSResponder`, and the user's `$EDITOR`.
- Don't bypass `hostsfile.Write`.
- Don't allow combined profiles to reference other combined profiles.
- Don't write outside `home.Root` (except the configured `hosts_path`).
- Don't add new top-level dependencies without a clear reason — the project
  intentionally has only `cobra` + `yaml.v3`.
