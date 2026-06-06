# althosts

> `althosts` — switch `/etc/hosts` profiles safely from the command line.

`althosts` is a small CLI that manages multiple local `/etc/hosts` profiles and
switches between them on demand. Everything stays local — no remote sync, daemons, or GUI.

**macOS nicety:** althosts uses the normal `sudo` flow, so if you enable Touch
ID for `sudo` with Apple's documented macOS configuration, applying hosts
profiles can use the same biometric prompt as other sudo commands.

## Install

```bash
brew install xshoji/tap/althosts
```


## Quick start

```bash
althosts init                       # create ~/.althosts and seed `default`
althosts create dev                 # snapshot current /etc/hosts as "dev"
althosts edit dev                   # open in $EDITOR
althosts validate dev
althosts diff dev                   # diff against current /etc/hosts
althosts use dev                    # apply (auto self-elevates with sudo if needed)
althosts list                       # see what's active
althosts use default                # back to the original hosts

althosts combine dev-full default dev  # build a combined profile from plain ones
althosts use dev-full                  # apply the combined profile
althosts remove dev                    # delete a profile (refused if active)
```

`althosts init` automatically saves a snapshot of the current `/etc/hosts` as
a profile named `default`, so you can always switch back with
`althosts use default`. Pass `--no-default` to skip the seed.

## Touch ID for sudo on macOS

On supported macOS versions, you can enable Touch ID for `sudo` by creating
`/etc/pam.d/sudo_local` from Apple's template and uncommenting the standard PAM
line:

```text
auth       sufficient     pam_tid.so
```

This is the Apple-documented `sudo_local` mechanism, not a modification of the
system `/etc/pam.d/sudo` file. Apple describes this in
[What's new for enterprise in macOS Sonoma](https://support.apple.com/en-us/HT213893):

> Touch ID can be allowed for `sudo` with a configuration that persists across
> software updates using `/etc/pam.d/sudo_local`. See
> `/etc/pam.d/sudo_local.template` for details.

In other words, this is the normal local configuration Apple provides for
`sudo`, not an althosts-specific setting. althosts does not write this file;
once you enable Touch ID for sudo, `althosts use <name>` benefits from the
same system-wide sudo behavior when it self-elevates.

One way to apply Apple's template manually is:

```bash
$ sudo cp /etc/pam.d/sudo_local.template /etc/pam.d/sudo_local
$ sudo vi /etc/pam.d/sudo_local   # uncomment the pam_tid.so line
$ cat /etc/pam.d/sudo_local
# Managed by althosts. Re-run `althosts touchid --disable` to remove.
auth       sufficient     pam_tid.so
```

## Layout

```
~/.althosts/
├── profiles/
│   ├── default.hosts
│   ├── dev.hosts
│   └── staging.hosts
├── combined/
│   └── dev-full.yaml          # combined definition (YAML)
├── state.json                 # currently-active profile
└── config.yaml
```

The home directory can be relocated:

```bash
ALTHOSTS_HOME=~/dotfiles/althosts althosts list
althosts --home ./testdata list
```

Priority is `--home` > `$ALTHOSTS_HOME` > `~/.althosts`.

## Combined profiles

A combined profile concatenates multiple plain profiles when applied. Create
one with the `combine` command:

```bash
althosts combine dev-full default dev ads-block
```

This writes `~/.althosts/combined/dev-full.yaml`:

```yaml
members:
  - default
  - dev
  - ads-block
```

Then activate it like any plain profile:

```bash
althosts use dev-full
```

The hosts content is generated dynamically at apply / show / diff time.
Combined profiles cannot reference other combined profiles. The same `edit`,
`show`, `diff`, `validate`, and `remove` commands work for combined profiles —
the kind is detected automatically from the name.

## Commands

| Command | Description |
|---|---|
| `init [--no-default]` | Create `~/.althosts` and seed a `default` profile from current `/etc/hosts` |
| `list [--current] [--json]` | List profiles and combined definitions (active is marked `*`) |
| `create <name> [--from-current\|--empty\|--from <name-or-path>]` | Create a plain profile. `--from` accepts an existing profile name or a local file path |
| `combine <name> <member> [<member>...]` | Create a combined profile from existing plain profile members |
| `edit <name>` | Open profile or combined definition in `$EDITOR` (kind auto-detected, auto-applies if active) |
| `show <name>` | Print the rendered hosts content (works for plain and combined) |
| `diff <name>` | Diff rendered profile against current `/etc/hosts` |
| `validate <name>` | Lint hosts content; warnings are non-blocking, errors block `use` |
| `use <name> [--no-flush] [--force]` | Apply profile (auto self-elevates with sudo when writing `/etc/hosts`) |
| `remove <name>` | Remove a profile or combined definition (refused if currently active) |
| `doctor` | Diagnose the environment |

## Configuration

`~/.althosts/config.yaml`:

```yaml
hosts_path: /etc/hosts
flush_dns: true
editor: ""        # empty -> $VISUAL -> $EDITOR -> vi
```

`use` self-elevates with `sudo` only when writing the platform default
`/etc/hosts`. See [Why `sudo`?](#why-sudo) for details on when and how
elevation kicks in.

On macOS, `dscacheutil -flushcache` is run automatically after each `use`
unless `flush_dns: false` or `--no-flush` is passed.

## Why `sudo`?

`/etc/hosts` is owned by `root`, so technically *something* with root
privileges must perform the write — there is no way around that.

But forcing users to type `sudo althosts ...` and think about which
operations need elevation is a poor experience. althosts therefore
detects when an operation actually requires root and only then prompts
you interactively to elevate — everything else runs as your normal user.

**Flow when you run `althosts use <name>`:**

1. Render + validate as your normal user (uses your `$HOME`, `$EDITOR`, etc.).
2. Probe `hosts_path` for writability. If writable, just write — **no sudo**.
3. Only if `/etc/hosts` needs root, re-exec under `/usr/bin/sudo` with the
   same args plus `--home <resolved>` (sudo resets `HOME`, so we pin it).
   An `ALTHOSTS_ELEVATED=1` marker prevents elevation loops.

**Design choices:**

- Render / validate / editor stay in user context — running them as root
  would touch the wrong `HOME` and give the editor root privileges.
- Elevate on demand, not via a setuid binary or daemon — minimum surface.
- Custom `hosts_path` (non-default) **never** triggers sudo; it must be
  writable by you.

If you've enabled Touch ID for sudo (`/etc/pam.d/sudo_local`), the same
biometric prompt is used.



## Development

### Build

```bash
go build -ldflags="-s" -trimpath -o althosts main.go
```

### Test

```bash
go test -v ./...
```

## Release

The release flow for this repository is automated with GitHub Actions.
Pushing Git tags triggers the release job.

```
# Release
git tag v0.0.1 && git push --tags


# Delete tag
v="v0.0.1"; git tag -d "${v}" && git push origin :"${v}"

# Delete tag and recreate new tag and push
v="v0.0.1"; git tag -d "${v}" && git push origin :"${v}"; git tag "${v}"; git push --tags
```


## License

MIT — see [LICENSE](LICENSE).
