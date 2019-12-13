# Gold Style Reference

## Block Elements

Block elements contain other elements and are rendered around them. All block
elements support the following style settings:

| Attribute        | Value  | Description                                        |
| ---------------- | ------ | -------------------------------------------------- |
| prefix           | string | Printed before the block's first element           |
| suffix           | string | Printed after the block's last element             |
| styled_prefix    | string | Printed before the block's first element           |
| styled_suffix    | string | Printed after the block's last element             |
| indent           | number | Specifies the indentation of the block             |
| margin           | number | Specifies the margin around the block              |
| color            | color  | Defines the default text color for the block       |
| background_color | color  | Defines the default background color for the block |

Elements inside a block inherit the block's following style settings:

| Attribute        | Value | Description                                        |
| ---------------- | ----- | -------------------------------------------------- |
| color            | color | Defines the default text color for the block       |
| background_color | color | Defines the default background color for the block |
| bold             | bool  | Increases text intensity                           |
| faint            | bool  | Decreases text intensity                           |
| italic           | bool  | Prints the text in italic                          |
| crossed_out      | bool  | Enables strikethrough as text decoration           |
| underline        | bool  | Enables underline as text decoration               |
| overlined        | bool  | Enables overline as text decoration                |
| blink            | bool  | Enables blinking text                              |
| conceal          | bool  | Conceals / hides the text                          |
| inverse          | bool  | Swaps fore- & background colors                    |

### document

The `document` element represents the markdown's body.

#### Example

Style:

```
"document": {
    "indent": 2,
    "background_color": "234",
    "prefix": "\n",
    "suffix": "\n"
}
```

---

### paragraph

The `paragraph` element represents a paragraph in the document.

#### Example

Style:

```
"paragraph": {
    "margin": 4,
    "color": "15",
    "background_color": "235"
}
```

---

### heading

The `heading` element represents a heading.

### h1 - h6

The `h1` to `h6` elements represent headings. `h1` defines the most important
heading, `h6` the least important heading. Undefined attributes are inherited
from the `heading` element.

#### Example

Markdown:

```
# h1

## h2

### h3
```

Style:

```
"heading": {
    "color": "15",
    "background_color": "57"
},
"h1": {
    "styled_prefix": "=> ",
    "styled_suffix": " <=",
    "margin": 2,
    "bold": true,
    "background_color": "69"
},
"h2": {
    "styled_prefix": "## ",
    "margin": 4
},
"h3": {
    "styled_prefix": "### ",
    "margin": 6
}
```

Output:

![Heading Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/heading.png)

---

### block_quote

The `block_quote` element represents a quote.

#### Example

Style:

```
"block_quote": {
    "color": "200"
}
```

Output:

![Block Quote Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/block_quote.png)

---

### list

The `list` element represents a list in the document.

| Attribute    | Value  | Description                                |
| ------------ | ------ | ------------------------------------------ |
| level_indent | number | Specifies the indentation for nested lists |

#### Example

Style:

```
"list": {
    "color": "15",
    "background_color": "52",
    "level_indent": 4
}
```

---

### code_block

The `code_block` element represents a block of code.

| Attribute | Value  | Description                                                     |
| --------- | ------ | --------------------------------------------------------------- |
| theme     | string | Defines the [Chroma][chroma] theme used for syntax highlighting |

[chroma]: https://github.com/alecthomas/chroma

#### Example

Style:

```
"code_block": {
    "color": "200",
    "theme": "solarized-dark"
}
```

Output:

![Code Block Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/code_block.png)

---

### table

The `table` element represents a table of data.

#### Example

Markdown:

```
| Label  | Value |
| ------ | ----- |
| First  | foo   |
| Second | bar   |
```

Style:

```
"table": {
    "margin": 4
}
```

Output:

![Table Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/table.png)

## Inline Elements

All inline elements support the following style settings:

| Attribute        | Value  | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| prefix           | string | Printed before the element                            |
| suffix           | string | Printed after the element                             |
| styled_prefix    | string | Printed before the element                            |
| styled_suffix    | string | Printed after the element                             |
| color            | color  | Defines the default text color for the document       |
| background_color | color  | Defines the default background color for the document |
| bold             | bool   | Increases text intensity                              |
| faint            | bool   | Decreases text intensity                              |
| italic           | bool   | Prints the text in italic                             |
| crossed_out      | bool   | Enables strikethrough as text decoration              |
| underline        | bool   | Enables underline as text decoration                  |
| overlined        | bool   | Enables overline as text decoration                   |
| blink            | bool   | Enables blinking text                                 |
| conceal          | bool   | Conceals / hides the text                             |
| inverse          | bool   | Swaps fore- & background colors                       |

### text

The `text` element represents a block of text.

#### Example

Style:

```
"text": {
    "bold": true,
    "color": "15",
    "background_color": "57"
}
```

---

### item

The `item` element represents an item in a list.

#### Example

Markdown:

```
- First Item
    - Nested List Item
- Second Item
```

Style:

```
"item": {
    "prefix": "• "
}
```

Output:

![List Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/list.png)

---

### enumeration

The `enumeration` element represents an item in an ordered list.

#### Example

Markdown:

```
1. First Item
2. Second Item
```

Style:

```
"enumeration": {
    "prefix": ". "
}
```

Output:

![Enumeration Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/enumeration.png)

---

### task

The `task` element represents a task item.

| Attribute | Value  | Description                 |
| --------- | ------ | --------------------------- |
| ticked    | string | Prefix for finished tasks   |
| unticked  | string | Prefix for unfinished tasks |

#### Example

Markdown:

```
- [x] Finished Task
- [ ] Outstanding Task
```

Style:

```
"task": {
    "ticked": "✓ ",
    "unticked": "✗ "
}
```

Output:

![Task Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/task.png)

---

### link

The `link` element represents a link.

#### Example

Markdown:

```
This is a [link](https://charm.sh).
```

Style:

```
"link": {
    "color": "123",
    "underline": true,
    "prefix": "(",
    "suffix": ")"
}
```

Output:

![Link Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/link.png)

---

### link_text

The `link_text` element represents the text associated with a link.

#### Example

Style:

```
"link_text": {
    "color": "123",
    "bold": true
}
```

---

### image

The `image` element represents an image.

#### Example

Markdown:

```
![Image](https://charm.sh/logo.png).
```

Style:

```
"image": {
    "color": "123",
    "prefix": "[Image: ",
    "suffix": "]"
}
```

Output:

![Image Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/image.png)

---

### image_text

The `image_text` element represents the text associated with an image.

#### Example

Style:

```
"image_text": {
    "color": "8"
}
```

---

### code

The `code` element represents an inline code segment.

#### Example

Style:

```
"code": {
    "color": "200"
}
```

Output:

![Code Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/code.png)

---

### emph

The `emph` element represents an emphasized text.

#### Example

Markdown:

```
This text is *emphasized*.
```

Style:

```
"emph": {
    "italic": true
}
```

Output:

![Emph Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/emph.png)

---

### strong

The `strong` element represents important text.

#### Example

Markdown:

```
This text is **strong**.
```

Style:

```
"strong": {
    "bold": true
}
```

Output:

![Strong Example](https://github.com/charmbracelet/gold/raw/master/styles/examples/strong.png)

---

### hr

The `hr` element represents a horizontal rule.

#### Example

Markdown:

```
---
```

Style:

```
"hr": {
    "prefix": "---"
}
```

## html_block
## html_span
