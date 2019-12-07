# Gold Styles

## Elements

### document

The `document` element defines the markdown's body.

#### Attributes

| Attribute        | Value  | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| prefix           | string | Printed before the first element during rendering     |
| suffix           | string | Printed after the last element during rendering       |
| indent           | number | Specifies the indentation of the document             |
| margin           | number | Specifies the margin around the document              |
| color            | color  | Defines the default text color for the document       |
| background_color | color  | Defines the default background color for the document |

#### Example

```
"document": {
    "indent": 2,
    "background_color": "234",
    "prefix": "\n",
    "suffix": "\n"
}
```

### text

The `text` element defines a block of text.

#### Attributes

| Attribute        | Value  | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| prefix           | string | Printed before the text gets printed                  |
| suffix           | string | Printed after the text got printed                    |
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

#### Example

```
"text": {
    "bold": true
    "color": "15",
    "background_color": "57"
}
```

### block_quote
### list
### item
### enumeration
### paragraph
### heading
### h1 - h6
### hr
### emph
### strong
### del
### link
### link_text
### image
### image_text
### html_block
### code_block
### softbreak
### hardbreak
### code
### html_span
### table
### table_cel
### table_head
### table_body
### table_row
