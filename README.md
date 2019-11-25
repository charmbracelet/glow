# Gold

Render markdown on the CLI, with _pizzazz_!

## What is it?

Gold is a Golang library that allows you to use JSON based stylesheets to
render Markdown files in the terminal. Just like CSS, you can define color and
style attributes on Markdown elements. The difference is that you use ANSI
color and terminal codes instead of CSS properties and hex colors.

## Usage

See [cmd/gold](cmd/gold/).

## Colors

Currently `gold` uses the [Aurora ANSI colors](https://godoc.org/github.com/logrusorgru/aurora#Index).
