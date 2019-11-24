# Gold

Render markdown on the CLI, with _pizzazz_!

## Usage

Supply a JSON stylesheet with the `-s` flag. Use a markdown source as the argument.

Read from file:
```
./gold -s dark.json README.md
```

Read from stdin:
```
./gold -s dark.json -
```

Fetch README from GitHub:
```
./gold -s dark.json https://github.com/charmbracelet/gold
```

When `gold` is started without any markdown source, it will try to find a `README.md`
or `README` file in the current working directory.

## Colors

Currently `gold` uses the [Aurora ANSI colors](https://godoc.org/github.com/logrusorgru/aurora#Index).

## Example Output

![Gold Dark Theme](https://github.com/charmbracelet/gold/raw/master/cmd/gold/gold_dark.png)
