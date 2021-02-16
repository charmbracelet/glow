# Glow

Render markdown on the CLI, with _pizzazz_!

<p align="center">
    <img src="https://stuff.charm.sh/glow/glow-banner-github.gif" alt="Glow Logo">
    <a href="https://github.com/charmbracelet/glow/releases"><img src="https://img.shields.io/github/release/charmbracelet/glow.svg" alt="Latest Release"></a>
    <a href="https://pkg.go.dev/github.com/charmbracelet/glow?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="GoDoc"></a>
    <a href="https://github.com/charmbracelet/glow/actions"><img src="https://github.com/charmbracelet/glow/workflows/build/badge.svg" alt="Build Status"></a>
    <a href="http://goreportcard.com/report/github.com/charmbracelet/glow"><img src="http://goreportcard.com/badge/charmbracelet/glow" alt="Go ReportCard"></a>
</p>

<p align="center">
    <img src="https://stuff.charm.sh/glow/glow-1.3-trailer-github.gif" width="600" alt="Glow UI Demo">
</p>

## What is it?

Glow is a terminal based markdown reader designed from the ground up to bring
out the beauty—and power—of the CLI.

Use it to discover markdown files, read documentation directly on the command
line and stash markdown files to your own private collection so you can read
them anywhere. Glow will find local markdown files in subdirectories or a local
Git repository.

By the way, all data stashed is encrypted end-to-end: only you can decrypt it.
More on that below.

## Installation

Use your fave package manager:

```bash
# macOS or Linux
brew install glow

# macOS (with MacPorts)
sudo port install glow

# Arch Linux (btw)
yay -S glow

# Void Linux
xbps-install -S glow

# Nix
nix-env -iA nixpkgs.glow

# FreeBSD
pkg install glow

# Solus
eopkg install glow

# Windows (with Scoop)
scoop install glow

# Android (with termux)
pkg install glow
```

Or download a binary from the [releases][releases] page. MacOS, Linux, Windows,
FreeBSD, and OpenBSD binaries are available, as well as Debian, RPM, and Alpine
packages. ARM builds are also available for Linux, FreeBSD, and OpenBSD.

Or just build it yourself (requires Go 1.13+):

```bash
git clone https://github.com/charmbracelet/glow.git
cd glow
go build
```

[releases]: https://github.com/charmbracelet/glow/releases


## The TUI

Simply run `glow` without arguments to start the textual user interface and
browse local and stashed markdown. Glow will find local markdown files in the
current directory and below or, if you’re in a Git repository, Glow will search
the repo.

Markdown files can be read with Glow's high-performance pager. Most of the
keystrokes you know from `less` are the same, but you can press `?` to list
the hotkeys.

### Stashing

Glow works with the Charm Cloud to allow you to store any markdown files in
your own private collection. You can stash a local document from the Glow TUI by
pressing `s`.

Stashing is private, its contents will not be exposed publicly, and it's
encrypted end-to-end. More on encryption below.

## The CLI

In addition to a TUI, Glow has a CLI for working with Markdown. To format a
document use a markdown source as the primary argument:

```bash
# Read from file
glow README.md

# Read from stdin
glow -

# Fetch README from GitHub / GitLab
glow github.com/charmbracelet/glow

# Fetch markdown from HTTP
glow https://host.tld/file.md
```

### Stashing

You can also stash documents from the CLI:

```bash
glow stash README.md
```

Then, when you run `glow` without arguments will you can browse through your
stashed documents. This is a great way to keep track of things that you need to
reference often.

### Word Wrapping

The `-w` flag lets you set a maximum width at which the output will be wrapped:

```bash
glow -w 60
```

### Paging

CLI output can be displayed in your preferred pager with the `-p` flag. This defaults
to the ANSI-aware `less -r` if `$PAGER` is not explicitly set.

### Styles

You can choose a style with the `-s` flag. When no flag is provided `glow` tries
to detect your terminal's current background color and automatically picks
either the `dark` or the `light` style for you.

```bash
glow -s [dark|light]
```

Alternatively you can also supply a custom JSON stylesheet:

```bash
glow -s mystyle.json
```

For additional usage details see:

```bash
glow --help
```

Check out the [Glamour Style Section](https://github.com/charmbracelet/glamour/blob/master/styles/gallery/README.md)
to find more styles. Or [make your own](https://github.com/charmbracelet/glamour/tree/master/styles)!

## The Config File

If you find yourself supplying the same flags to `glow` all the time, it's
probably a good idea to create a config file. Run `glow config`, which will open
it in your favorite $EDITOR. Alternatively you can manually put a file named
`glow.yml` in the default config path of you platform. If you're not sure where
that is, please refer to `glow --help`.

Here's an example config:

```yaml
# style name or JSON path (default "auto")
style: "light"
# show local files only; no network (TUI-mode only)
local: true
# mouse support (TUI-mode only)
mouse: true
# use pager to display markdown
pager: true
# word-wrap at width
width: 80
```

## 🔒 Encryption: How It Works

Encryption works by issuing symmetric keys (basically a generated password) and
encrypting it with the local SSH public key generated by the open-source
[charm][charmlib] library. That encrypted key is then sent up to our server.
We can’t read it since we don’t have your private key. When you want to decrypt
something or view your stash, that key is downloaded from our server and
decrypted locally using the SSH private key. When you link accounts, the
symmetric key is encrypted for each new public key. This happens on your
machine and not our server, so we never see any unencrypted data.

[charmlib]: https://github.com/charmbracelet/charm

## License

[MIT](https://github.com/charmbracelet/glow/raw/master/LICENSE)

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源! / Charm loves open source!

