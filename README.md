# Glow

Render markdown on the CLI, with _pizzazz_!

<p align="center">
    <img src="https://stuff.charm.sh/glow-github.gif" alt="Glow Logo">
    <a href="https://github.com/charmbracelet/glow/releases"><img src="https://img.shields.io/github/release/charmbracelet/glow.svg" alt="Latest Release"></a>
    <a href="https://pkg.go.dev/github.com/charmbracelet/glow?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="GoDoc"></a>
    <a href="https://github.com/charmbracelet/glow/actions"><img src="https://github.com/charmbracelet/glow/workflows/build/badge.svg" alt="Build Status"></a>
    <a href="http://goreportcard.com/report/github.com/charmbracelet/glow"><img src="http://goreportcard.com/badge/charmbracelet/glow" alt="Go ReportCard"></a>
</p>

![Glow example output](https://github.com/charmbracelet/glow/raw/master/example.png)

## What is it?

Glow is a terminal based markdown reader designed from the ground up to bring
out the beauty of the CLI.

Use it to quickly discover markdown files in a folder (it will automatically
search subdirectories for you), read documentation directly on the command line
and stash markdown files to your own private collection in the Charm Cloud so
you can read them anywhere.

## Installation

Use your fave package manager:

```bash
# macOS or Linux
brew install glow

# Arch Linux (btw)
yay -S glow

# Void Linux
xbps-install -S glow

# Nix
nix-env -iA nixpkgs.glow

# FreeBSD
pkg install glow
```

Or download a binary from the [releases][] page. MacOS, Linux, FreeBSD
binaries are available, as well as Debian and RPM packages. ARM builds are also
available for Linux and FreeBSD.

Or just use `go get`:

```bash
go get github.com/charmbracelet/glow
```

[releases]: https://github.com/charmbracelet/glow/releases


## Usage

Use a markdown source as the primary argument:

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

When `glow` is started without a markdown source, it will try to find a
`README.md` or `README` file in the current working directory.

### Stashing

Glow works with the Charm Cloud to allow you to store any markdown file in your
own private collection. When you run:

`glow stash README.md`

You'll add that markdown file to your stash. Running `glow` without arguments
will let you browse through all your stashed documents. This is a great way to
keep track of documentation that you need to reference.

Stashing is private and your stash will not be exposed publicly.

### Word Wrapping

The `-w` flag lets you set a maximum width at which the output will be wrapped:

```bash
glow -w 60
```

### Paging

The output can be displayed in the user's preferred pager with the `-p` flag.
This defaults to the ANSI-aware `less -r` if `$PAGER` is not explicitly set.

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

Check out the [Glamour Style Section](https://github.com/charmbracelet/glamour/blob/master/styles/gallery/README.md)
to find more styles. Or [make your own](https://github.com/charmbracelet/glamour/tree/master/styles)!

***

For additional usage details see:

```bash
glow --help
```

## License

[MIT](https://github.com/charmbracelet/glow/raw/master/LICENSE)

***

Part of [Charm](https://charm.sh).

<img alt="the Charm logo" src="https://stuff.charm.sh/charm-logotype.png" width="400px">

Charm热爱开源!
