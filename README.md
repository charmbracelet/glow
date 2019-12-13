# Gold

Render markdown on the CLI, with _pizzazz_!

## Usage

Use a markdown source as the primary argument:

Read from file:

```console
gold README.md
```

Read from stdin:

```console
gold -
```

Fetch README from GitHub:

```console
gold github.com/charmbracelet/gold
```

Fetch markdown from an HTTP source:

```console
gold https://host.tld/file.md
```

When `gold` is started without any markdown source, it will try to find a
`README.md` or `README` file in the current working directory.

### Word Wrapping

The `-w` flag lets you set a maximum width, at which the output will be wrapped:

```console
gold -w 60
```

### Styles

You can choose a style with the `-s` flag (`dark` being the default):

```console
gold -s [dark|light]
```

Alternatively you can also supply a custom JSON stylesheet:

```console
gold -s mystyle.json
```

Check out the [Gold Style Gallery](https://github.com/charmbracelet/gold/blob/master/styles/gallery/README.md)
to find more available styles!

## Example Output

![Gold Dark Style](https://github.com/charmbracelet/gold/raw/master/styles/gallery/dark.png)

## Colors

Currently `gold` uses the [Aurora ANSI colors](https://godoc.org/github.com/logrusorgru/aurora#Index).
