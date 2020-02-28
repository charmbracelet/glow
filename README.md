# Glow

[![Latest Release](https://img.shields.io/github/release/charmbracelet/glow.svg)](https://github.com/charmbracelet/glow/releases)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/charmbracelet/glow)
[![Build Status](https://github.com/charmbracelet/glow/workflows/build/badge.svg)](https://github.com/charmbracelet/glow/actions)
[![Go ReportCard](http://goreportcard.com/badge/charmbracelet/glow)](http://goreportcard.com/report/charmbracelet/glow)

Render markdown on the CLI, with _pizzazz_!

![Glow example output](https://github.com/charmbracelet/glow/raw/master/example.png)


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

Or download a binary from the [releases][] page. Windows, MacOS, and Linux
(including ARM) binaries are available, as well as Debian and RPM packages.

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


## Authors

* [Christian Muehlhaeuser](https://github.com/muesli)
* [Toby Padilla](https://github.com/toby)
* [Christian Rocha](https://github.com/meowgorithm)

Part of [Charm](https://charm.sh). For more info see `ssh charm.sh`. Charm热爱开源!

## License

[MIT](https://github.com/charmbracelet/glow/raw/master/LICENSE)
