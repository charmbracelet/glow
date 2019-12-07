# Gold

Render markdown on the CLI, with _pizzazz_!

## Usage

Use a markdown source as the argument:

Read from file:
```
./gold README.md
```

Read from stdin:
```
./gold -
```

Fetch README from GitHub:
```
./gold github.com/charmbracelet/gold
```

Fetch markdown from an HTTP source:
```
./gold https://host.tld/file.md
```

When `gold` is started without any markdown source, it will try to find a
`README.md` or `README` file in the current working directory.

You can supply a JSON stylesheet with the `-s` flag:
```
./gold -s mystyle.json
```

## Example Output

![Gold Dark Style](https://github.com/charmbracelet/gold/raw/master/styles/gallery/dark.png)

Check out the [Gold Style Gallery](https://github.com/charmbracelet/gold/blob/master/styles/gallery/README.md)!

## Colors

Currently `gold` uses the [Aurora ANSI colors](https://godoc.org/github.com/logrusorgru/aurora#Index).
