# Gold

Render markdown on the CLI, with _pizzazz_!


## Installation

Use your fave package manager:

```bash
# MacOS
brew install gold

# Arch Linux (btw)
pacman -S gold
```

Or just use `go get`:

```bash
go get github.com/charmbracelet/gold
```


## Usage

Use a markdown source as the primary argument:

```bash
# Read from file
gold README.md

# Read from stdin
gold -

# Fetch README from GitHub
gold github.com/charmbracelet/gold

# Fetch markdown from HTTP
gold https://host.tld/file.md
```

When `gold` is started without a markdown source, it will try to find a
`README.md` or `README` file in the current working directory.

### Word Wrapping

The `-w` flag lets you set a maximum width at which the output will be wrapped:

```bash
gold -w 60
```

### Styles

You can choose a style with the `-s` flag (`dark` being the default):

```bash
gold -s [dark|light]
```

Alternatively you can also supply a custom JSON stylesheet:

```bash
gold -s mystyle.json
```

Check out the [Glamour Style Section](https://github.com/charmbracelet/glamour/blob/master/styles/gallery/README.md)
to find more styles. Or [make your own](https://github.com/charmbracelet/glamour/tree/master/styles)!

***

For additional usage details see:

```bash
gold --help
```


## Example Output

![Glamour Dark Style](https://github.com/charmbracelet/gold/raw/master/example.png)


## Authors

* [Christian Muehlhaeuser](https://github.com/muesli)
* [Toby Padilla](https://github.com/toby)
* [Christian Rocha](https://github.com/meowgorithm)

Part of [Charm](https://charm.sh). For more info see: `ssh charm.sh`


## License

[MIT](https://github.com/charmbracelet/gold/raw/master/LICENSE)
