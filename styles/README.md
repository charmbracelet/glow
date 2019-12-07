# Gold Style Reference

## document

The `document` element defines the markdown's body.

### Attributes

| Attribute        | Value  | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| prefix           | string | Printed before the first element during rendering     |
| suffix           | string | Printed after the last element during rendering       |
| indent           | number | Specifies the indentation of the document             |
| margin           | number | Specifies the margin around the document              |
| color            | color  | Defines the default text color for the document       |
| background_color | color  | Defines the default background color for the document |

### Example

```
"document": {
    "indent": 2,
    "background_color": "234",
    "prefix": "\n",
    "suffix": "\n"
}
```

## paragraph

The `paragraph` element defines a paragraph in the document.

### Attributes

| Attribute        | Value  | Description                                         |
| ---------------- | ------ | --------------------------------------------------- |
| prefix           | string | Printed before the first element in a paragraph     |
| suffix           | string | Printed after the last element in a paragraph       |
| indent           | number | Specifies the indentation of paragraphs             |
| margin           | number | Specifies the margin around paragraphs              |
| color            | color  | Defines the default text color for paragraphs       |
| background_color | color  | Defines the default background color for paragraphs |

### Example

```
"paragraph": {
    "margin": 4,
    "color": "15",
    "background_color": "235"
}
```

## heading

The `heading` element defines a heading.

## h1 - h6

The `h1` to `h6` elements define headings. `h1` defines the most important
heading, `h6` the least important heading. Undefined attributes are inherited
from the `heading` element.

### Attributes

| Attribute        | Value  | Description                                       |
| ---------------- | ------ | ------------------------------------------------- |
| prefix           | string | Printed before a heading                          |
| suffix           | string | Printed after a heading                           |
| indent           | number | Specifies the indentation of headings             |
| margin           | number | Specifies the margin around headings              |
| color            | color  | Defines the default text color for headings       |
| background_color | color  | Defines the default background color for headings |
| bold             | bool   | Increases text intensity                          |
| faint            | bool   | Decreases text intensity                          |
| italic           | bool   | Prints the text in italic                         |
| crossed_out      | bool   | Enables strikethrough as text decoration          |
| underline        | bool   | Enables underline as text decoration              |
| overlined        | bool   | Enables overline as text decoration               |
| blink            | bool   | Enables blinking text                             |
| conceal          | bool   | Conceals / hides the text                         |
| inverse          | bool   | Swaps fore- & background colors                   |

### Example

```
"heading": {
    "prefix": "\n",
    "margin": 4,
    "color": "15",
    "background_color": "57"
},
"h1": {
    "prefix": "\n=> ",
    "suffix": " <=",
    "margin": 2,
    "bold": true,
    "background_color": "69"
},
"h2": {
    "prefix": "\n# "
}
```

## text

The `text` element defines a block of text.

### Attributes

| Attribute        | Value  | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| prefix           | string | Printed before a text gets printed                    |
| suffix           | string | Printed after a text got printed                      |
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

### Example

```
"text": {
    "bold": true,
    "color": "15",
    "background_color": "57"
}
```

## list

The `list` element defines a list in the document.

### Attributes

| Attribute        | Value  | Description                                    |
| ---------------- | ------ | ---------------------------------------------- |
| prefix           | string | Printed before a list                          |
| suffix           | string | Printed after a list                           |
| indent           | number | Specifies the indentation of lists             |
| margin           | number | Specifies the margin around lists              |
| color            | color  | Defines the default text color for lists       |
| background_color | color  | Defines the default background color for lists |

### Example

```
"list": {
    "margin": 4,
    "color": "15",
    "background_color": "52"
}
```

## item

The `item` element defines an item in lists.

### Attributes

| Attribute        | Value  | Description                                    |
| ---------------- | ------ | ---------------------------------------------- |
| prefix           | string | Printed before an item                         |
| suffix           | string | Printed after an item                          |
| color            | color  | Defines the default text color for items       |
| background_color | color  | Defines the default background color for items |
| bold             | bool   | Increases text intensity                       |
| faint            | bool   | Decreases text intensity                       |
| italic           | bool   | Prints the text in italic                      |
| crossed_out      | bool   | Enables strikethrough as text decoration       |
| underline        | bool   | Enables underline as text decoration           |
| overlined        | bool   | Enables overline as text decoration            |
| blink            | bool   | Enables blinking text                          |
| conceal          | bool   | Conceals / hides the text                      |
| inverse          | bool   | Swaps fore- & background colors                |

### Example

```
"item": {
    "prefix": "• "
}
```

## enumeration

The `enumeration` element defines an item in ordered lists.

### Attributes

| Attribute        | Value  | Description                                    |
| ---------------- | ------ | ---------------------------------------------- |
| prefix           | string | Printed before an item                         |
| suffix           | string | Printed after an item                          |
| color            | color  | Defines the default text color for items       |
| background_color | color  | Defines the default background color for items |
| bold             | bool   | Increases text intensity                       |
| faint            | bool   | Decreases text intensity                       |
| italic           | bool   | Prints the text in italic                      |
| crossed_out      | bool   | Enables strikethrough as text decoration       |
| underline        | bool   | Enables underline as text decoration           |
| overlined        | bool   | Enables overline as text decoration            |
| blink            | bool   | Enables blinking text                          |
| conceal          | bool   | Conceals / hides the text                      |
| inverse          | bool   | Swaps fore- & background colors                |

### Example

```
"enumeration": {
    "prefix": ". "
}
```

## block_quote
## hr
## emph
## strong
## del
## link
## link_text
## image
## image_text
## html_block
## code_block
## softbreak
## hardbreak
## code
## html_span
## table
## table_cel
## table_head
## table_body
## table_row
