# Pager text search (`/`, `n`, `N`)

## Problem

Glow's TUI pager (`-t`/`-p`) has no way to search within a rendered document.
Users familiar with `less`, `vim`, or `tmux` copy-mode expect `/` to start a
search, `enter` to confirm it, and `n`/`N` to jump between matches.

## Scope

In scope: incremental text search inside the pager view (`ui/pager.go`),
forward/backward navigation between matches, match-count feedback, and
highlighting of matches in the rendered viewport.

Out of scope (YAGNI for v1): regex search, case-sensitivity toggle, search
history, persisting the last query across documents/sessions, searching the
stash/file-listing view (it already has its own `/`-filter, unrelated to this
feature).

## Background: what already exists

- `ui/stash.go` already implements a `/`-triggered inline filter using
  `bubbles/v2/textinput`, with its own `filterState` (`unfiltered` /
  `filtering` / `filterApplied`). This is the reference pattern for the input
  UX (`newStashModel`, `handleFiltering`).
- `bubbles/v2/viewport.Model` (v2.1.1) already ships with match-highlighting
  primitives we can build directly on:
  - `SetHighlights(matches [][]int)` — takes byte-offset ranges (as returned
    by `regexp.FindAllStringIndex` or equivalent) measured against
    `viewport.GetContent()`, converts them to line/column highlight info, and
    auto-scrolls to the nearest match relative to the current scroll
    position.
  - `HighlightNext()` / `HighlightPrevious()` — cycle the "selected" match
    with wraparound (confirmed in source: `(idx + 1) % len(highlights)`),
    and auto-scroll to keep the newly selected match visible
    (`EnsureVisible`).
  - `ClearHighlights()` — removes all highlight state.
  - `HighlightStyle` / `SelectedHighlightStyle` (`lipgloss.Style` fields on
    `Model`) — control how all matches vs. the current match are rendered.

  This means match tracking, wraparound, and scroll-to-match are handled by
  the library; our job is producing the byte-offset matches and wiring up
  keys/state/UI feedback.

## UX design

### Entering search

- `/` in the pager (browse state) enters a new pager state,
  `pagerStateSearch`, and focuses a `textinput.Model` (`m.searchInput`)
  rendered inline in the status bar with a `Find:` prompt — same visual
  treatment as the stash filter input (`stashInputPromptStyle`).
- Any previous highlights are cleared immediately when `/` is pressed (a new
  search always starts fresh; no query prefill).
- While in `pagerStateSearch`, keystrokes go to the text input, not to
  viewport navigation (mirrors `stashModel.handleFiltering`).

### Confirming / cancelling

- `enter`: read `m.searchInput.Value()`. If empty, just return to browse
  state with no highlights. Otherwise:
  1. Take `content := m.viewport.GetContent()`.
  2. Find all byte-offset ranges of case-insensitive literal matches of the
     query in `content` (equivalent to
     `regexp.MustCompile("(?i)" + regexp.QuoteMeta(query)).FindAllStringIndex(content, -1)`,
     implemented directly since this is the simplest correct way to get
     case-insensitive literal byte offsets without hand-rolling Unicode case
     folding).
  3. If no matches: show a transient status message ("No matches") via the
     existing `showStatusMessage` mechanism, clear the query, return to
     browse state.
  4. If matches exist: call `m.viewport.SetHighlights(matches)`, store the
     query on the model (`m.searchQuery`) so it can be re-run after a
     re-render, set `m.searching = true`, return to browse state.
- `esc` while typing search: discard the query, return to browse state,
  leave any prior highlight state untouched (there is none, since `/`
  cleared it) — consistent with "esc backs out one level."

### Navigating matches

- `n` (browse state, `m.searching == true`): `m.viewport.HighlightNext()`.
- `N` (browse state, `m.searching == true`): `m.viewport.HighlightPrevious()`.
- If `m.searching == false`, `n`/`N` are no-ops (not bound to anything else
  currently, so simply fall through).

### Clearing an active search

- `esc` in browse state, when `m.searching == true`: clear highlights
  (`m.viewport.ClearHighlights()`), set `m.searching = false`, clear
  `m.searchQuery`. This takes priority over the existing `esc` behavior in
  the pager only when a search is active; existing `esc`/`q` behavior
  (`pagerStateStatusMessage` → browse) is unaffected otherwise.
- Highlights/search state are also cleared whenever the document is unloaded
  or reloaded: `m.unload()`, and the `r` / `editorFinishedMsg` / `reloadMsg`
  reload paths, since match offsets are meaningless once content changes.

### Surviving resize

- On `tea.WindowSizeMsg`, the pager already fully re-renders via
  `renderWithGlamour`, producing a new `contentRenderedMsg` with fresh
  content (and thus invalidating old byte offsets). If `m.searching == true`
  when a `contentRenderedMsg` arrives, after calling `m.setContent(...)`,
  re-run the same match-finding logic against the new content using
  `m.searchQuery` and call `SetHighlights` again, so the current search
  transparently survives a terminal resize instead of silently going stale.

### Status bar / help feedback

- While `m.searching == true`, the status bar's note area shows
  `"n/N: <count> matches"` — replacing the document `Note` field
  temporarily, similar to how `pagerStateStatusMessage` temporarily replaces
  it today. (Exact copy can be refined during implementation; the point is
  visible confirmation search is active and how many hits exist.)
- Add a line to the pager help view (`?`): `/  search`, and fold `n`/`N`
  into the existing up/down-style rows as "next/prev match" — exact
  placement decided during implementation to fit the existing two-column
  layout in `helpView()`.

### Styling

- Add two new fields to `ui/styles.go`'s `Styles` struct:
  `searchHighlightStyle` and `searchSelectedHighlightStyle`
  (`lipgloss.Style`, not the `func(...string) string` pattern used
  elsewhere, since these are assigned directly to
  `viewport.Model.HighlightStyle` / `SelectedHighlightStyle`).
- Suggested treatment: dim background for all matches, a brighter
  accent-colored background (e.g. using the existing `fuchsia` or
  `yellowGreen` tones already in the palette) for the current match — to be
  finalized visually during implementation, adapting for light/dark
  background like the rest of `Styles`.
- Applied once in `newPagerModel` (or `setSize`) after `viewport.New()`.

## Data model changes (`ui/pager.go`)

```go
type pagerModel struct {
    // ...existing fields...
    searchInput textinput.Model
    searchQuery string
    searching   bool
}

const (
    pagerStateBrowse pagerState = iota
    pagerStateStatusMessage
    pagerStateSearch
)
```

## Error handling

- Empty query on `enter`: no-op, return to browse.
- Zero matches: status message, no highlight state retained.
- No regex compilation errors possible since the query is treated as a
  literal string, not user-supplied regex — avoids exposing regex syntax
  errors to the user for a "just search my text" feature.

## Testing

- Unit test the match-finding helper directly (pure function: `content,
  query string -> [][]int`), covering: no matches, single match, multiple
  matches, case-insensitivity, overlapping-looking substrings, matches
  adjacent to ANSI escape sequences from glamour-rendered content (to
  confirm we are searching/matching against the same content string passed
  to `viewport.SetHighlights`, per the library's documented contract).
- Manual TUI verification (`run` skill / building `glow -t` locally) for:
  `/` → type → `enter` → highlights appear and jump to nearest match; `n`/`N`
  cycle with wraparound; `esc` clears; resizing the terminal while a search
  is active preserves highlighting; searching a term with zero matches shows
  feedback and doesn't crash.
