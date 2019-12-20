# Glow

Render markdown on the CLI, with _pizzazz_!

![Glamour Dark Style](https://github.com/charmbracelet/glow/raw/master/example.png)


## Installation

Use your fave package manager:

```bash
# MacOS
brew install glow

# Arch Linux (btw)
yay -S glow
```

Or just use `go get`:

```bash
go get github.com/charmbracelet/glow
```


## Usage

Use a markdown source as the primary argument:

```bash
# Read from file
glow README.md

# Read from stdin
glow -

# Fetch README from GitHub
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

### Styles

You can choose a style with the `-s` flag (`dark` being the default):

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

Part of [Charm](https://charm.sh). For more info see `ssh charm.sh`


## License

[MIT](https://github.com/charmbracelet/glow/raw/master/LICENSE)
