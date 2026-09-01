# Summary: `-f` / `--follow` mode

Implemented `tail -f` for markdown (plan: `plans/follow-mode.md`): `glow -f
file.md` renders the file, then appends newly written content to stdout as the
file grows. Output is append-only — no repainting — so scrollback survives and
piping works.

## Changes

- **`follow.go`** (new) — the feature:
  - fsnotify watches the file's **parent directory**, so atomic-rename saves
    from editors don't kill the watch; events are debounced 100ms.
  - An offset reader pulls only appended bytes into a pending buffer.
  - A chunker flushes at the last blank line outside a fenced code block, with
    CommonMark-correct fence tracking (backtick/tilde, fence length, ≤3-space
    indent, no backticks in a backtick fence's info string).
  - A trailing block with no terminating blank line flushes after an **8s idle
    timeout**. If that splits an open code fence, the opening fence line is
    re-emitted with the next chunk so the remainder still renders as code
    (goldmark treats the unclosed fence as code-to-EOF).
  - A truncated or rewritten file (in-place edit, atomic save) gets a rendered
    `---` divider and a full re-render below prior output.
  - Non-markdown files render per complete line, wrapped as code blocks.
- **`main.go`** — registers `-f`/`--follow`; validates: exactly one local
  file, with clear errors for stdin, URLs, directories, and combining with
  `--pager`/`--tui`.
- **`follow_test.go`** (new) — unit tests for boundary detection (fences,
  short closing fence, tilde fences, indented pseudo-fences) and the flush
  paths (partial block held, fence held open, fence reopen after idle flush,
  plain-text line flushing).

## Verification

- `go build`, `go vet`, `go test ./...` all pass.
- `golangci-lint run`: zero issues in the new code (remaining findings are
  pre-existing in `ui/` and untouched parts of `main.go`).
- End-to-end script drove the built binary while appending to a watched file:
  partial paragraph stayed hidden until its blank line; a fence with an
  internal blank line held until closed; a rewrite produced divider + fresh
  render; a trailing unterminated paragraph appeared only after the 8s idle
  flush; all validation errors read correctly.

## Known limitations (documented in the plan)

- Append semantics: mid-file edits fall back to divider + full re-render.
- Reference-style links and setext headings don't resolve across chunk
  boundaries.
