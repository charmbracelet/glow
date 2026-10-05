---
title: Glow Mermaid Playground
---

# Glow Mermaid Playground

Glow draws ` ```mermaid ` blocks as diagrams, sized to the terminal width.
Flowcharts (including cyclic ones), sequence, state, class, ER, gantt, pie,
mindmap, timeline, journey, quadrant, xychart and gitGraph diagrams are
rendered; the rare remainder falls back to the source. The icons in the
labels are [Nerd Font][nf] glyphs and need a Nerd Font-patched terminal
font to show up.

[nf]: https://www.nerdfonts.com

## CI pipeline

Top-down flowchart, dotted smoke tests, a thick fast lane, spaced edge
labels, a decision diamond and a dashed retry loop:

```mermaid
flowchart TD
    L[" lint"] --> T[" test"]
    T --> B[" build"]
    B -.-> S[" smoke"]
    S --> G{{" pass?"}}
    G -- yes --> P[" deploy"]
    G -- no --> F[" fix"]
    F ==> P
    F -.-> T
```

## Service architecture

Left-to-right flowchart with every shape, fanning out to workers and back
into the store:

```mermaid
flowchart LR
    U(( user)) --> A( auth)
    A --> Q([ queue])
    Q --> W1[" worker A"]
    Q --> W2[" worker B"]
    W1 --> S{{" store"}}
    W2 --> S
```

## Sequence diagram

Participants, solid and dashed messages, every arrowhead style, notes and
a loop frame:

```mermaid
sequenceDiagram
    participant U as  user
    participant A as  auth
    U->>A: login
    A-->>U: token
    Note over U,A: refresh before expiry
    U->U: refresh
    loop until valid
        U->>A: request
        A--xU: denied
    end
    A-)U: welcome
```

## State machine

Start and end markers, aliased states and transitions:

```mermaid
stateDiagram-v2
    [*] --> Idle
    state "Long idle" as Idle
    Idle --> Running : start
    Running --> Paused : pause
    Paused --> Running : resume
    Running --> Idle : stop
    Idle --> [*]
```

## Class diagram

Members, stereotypes, inheritance, realization and aggregation:

```mermaid
classDiagram
    class Animal {
        +String name
        +makeSound()
    }
    class Duck {
        +swim()
    }
    class Flyable {
        <<interface>>
        +fly()
    }
    Animal <|-- Duck
    Duck ..|> Flyable
    Duck "1" o-- "2" Feet : has
```

## ER diagram

Entities with attributes and labelled relationships:

```mermaid
erDiagram
    CUSTOMER {
        string name PK
    }
    ORDER {
        int total
    }
    ITEM {
        string sku
    }
    CUSTOMER ||--o{ ORDER : places
    ORDER ||--|{ ITEM : contains
```

## Gantt chart

Sections, task flags and a milestone on a scaled time axis:

```mermaid
gantt
    title Release plan
    dateFormat YYYY-MM-DD
    section Prep
     Design :done, des1, 2024-01-01, 5d
     Build :crit, 2024-01-06, 10d
    section Ship
     Test :active, 2024-01-16, 4d
     Launch :milestone, 2024-01-20, 0d
```

## Pie chart

```mermaid
pie title Response codes
    "2xx" : 812
    "3xx" : 44
    "4xx" : 130
    "5xx" : 14
```

## Mindmap

```mermaid
mindmap
  root((glow features))
    Rendering
      Markdown
      Diagrams
    TUI
      Pager
      Stash
```

## Timeline

```mermaid
timeline
    title Terminal era
    1970s : VT100 : Bourne shell
    1990s : Linux : xterm
    2020s : GPUs in the shell : ligatures
```

## Journey

```mermaid
journey
    title Reading a README
    section Setup
      Clone repo: 5: Me
      Install glow: 4: Me
    section Reading
      Open file: 8: Me, You
```

## Quadrant chart

```mermaid
quadrantChart
    title Feature planning
    x-axis Low effort --> High effort
    y-axis Low value --> High value
    quadrant-1 Do later
    quadrant-2 Quick wins
    quadrant-3 Drop
    quadrant-4 Bet big
    diagrams: [0.25, 0.8]
    theming: [0.15, 0.5]
```

## XY chart

```mermaid
xychart-beta
    title Issues closed per month
    x-axis [jan, feb, mar, apr, may]
    bar [12, 19, 8, 24, 31]
```

## Git graph

```mermaid
gitGraph
    commit
    commit
    branch feature
    commit
    checkout main
    commit
    merge feature
    commit
```

## Error handling

A deeper flowchart tree with an early exit:

```mermaid
flowchart TD
    R[" request"] --> V[" validate"]
    V --> E{{" format<br/>error?"}}
    E -- yes --> H[" 400 bad request"]
    E -- no --> X[" execute"]
    X --> O({{" 500?"}})
    O -- no --> K[" 200 ok"]
    O -- retry --> R2[" retry 3"]
    R2 --> H2[" 503 give up"]
```

## Unsupported diagrams

The rare remainder, like C4 diagrams, sankeys and packet charts, degrade
to the source plus a note explaining why:

```mermaid
C4Context
    person(user, User)
```
