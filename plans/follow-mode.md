# Plan: `-f` / `--follow` mode

`tail -f` for markdown: append-only output to stdout, no repainting. Scrollback
stays intact and output works when piped.

## Scope

- Only valid for a single local file argument — error out for stdin, URLs, and
  directories (`cannot follow non-file sources`).
- `-f` is a free short flag (taken: `-p -t -s -w -a -l -n -m`).

## Design

### Core loop

- fsnotify watches the file's **parent directory** (survives atomic renames
  from editors) and filters events for the target filename.
- Keep a read offset. On write events, read from offset to EOF and append the
  new bytes to an input buffer. Nothing is printed yet.
- Debounce write bursts (~100ms) since saves often produce multiple events.

### Chunker (flush only at safe boundaries)

Markdown can't be rendered mid-block, so the buffer is only flushed at safe
boundaries. State to track is small:

- **Fenced code blocks:** track opening ``` / ~~~ (fence char and length, per
  CommonMark rules) and hold everything until the matching closing fence.
- **Outside a fence:** a blank line is the natural terminator — it ends a
  paragraph, list, or table.

A flushable chunk = everything up to the last blank line that isn't inside an
open fence. Render the chunk with glamour, print it, keep the remainder
buffered. Trim glamour's leading/trailing blank-line padding so consecutive
chunks read as one continuous document.

### Startup

Render the existing file content as the first chunk, then start following —
matching `tail -f` showing the tail before waiting.

### Truncation / rewrite

If file size < offset, the file was truncated or rewritten (e.g., an in-place
edit + save). Do what `tail -f` does: print a visible divider, re-render the
whole file below it, reset the offset. Scrolls, never repaints.

## Caveats (documented behavior)

1. **Append semantics, not edit semantics.** Ideal for files being appended to
   (build logs, LLM output streaming, accumulated notes). Mid-file edits fall
   back to the truncation path above.
2. **Trailing partial block.** A paragraph not yet followed by a blank line is
   ambiguous — the next write might continue it, and we can't unprint.
   Decision: flush complete blocks eagerly; a trailing unterminated block is
   flushed after an 8-second idle timeout. If the idle flush splits an open
   code fence, the opening fence line is re-emitted with the next chunk so
   the remainder still renders as code.
3. **Cross-chunk features degrade slightly.** Reference-style links defined in
   a later chunk and setext headings won't resolve across chunk boundaries,
   since each chunk renders independently. Rare in appended markdown; worth a
   line in the docs.

## Pipeline summary

fsnotify → offset reader → fence-aware blank-line chunker → per-chunk glamour
render → append to stdout, with truncation handled by divider-plus-rerender.
