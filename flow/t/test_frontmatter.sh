#!/bin/bash

set -uo pipefail

# Test framework setup
i=0 x=-1 a=-1 y=-1 n=0 all=-1 yes=-1 no=0 on=-1 off=0
[[ ${V-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || V=y
[[ ${X-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || X=n

it () { echo -e "$(printf '%.0s=' {1..79})\nTEST $((++i)) $*\n$(printf '%.0s-' {1..79})"; } 1>&2
ok () { echo TEST $i $((( $? )) && echo ❌ || echo ✅; echo " V=$V X=$X"); } 1>&2
vv () { ((i==V || V<0)) && cat -u || cat -u > /dev/null; }

# Build glow
go build .
cp -avn ./glow ./glow.orig >&2 || true
timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3


# Test 1: Simple frontmatter handling
it should handle simple frontmatter correctly; (
  ((i==X || X<0)) && set -x; set -e

  # Create test doc with frontmatter
  cat > /tmp/fm_simple.md <<'EOF'
---
title: Test Document
author: Test Author
---

# Main Content

This is the main content.
EOF

  # Compare output with and without streaming
  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_simple.md | ./glow -w0 --flow=-1 - 2>&1) <(cat /tmp/fm_simple.md | ./glow -w0 --flow=0 - 2>&1)

) | vv; ok

# Test 2: Frontmatter split across chunk boundaries with small flow
it should handle frontmatter split across chunks with flow=32; (
  ((i==X || X<0)) && set -x; set -e

  # Create doc with frontmatter that would split at 32 bytes
  cat > /tmp/fm_split.md <<'EOF'
---
title: This is a very long title that will definitely span across chunk boundaries
author: Author Name
date: 2024-01-01
tags: [test, streaming, markdown]
---

# Content After Frontmatter

Regular content here.
EOF

  # Test with small flow size that would split frontmatter
  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_split.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fm_split.md | ./glow -w0 --flow=32 - 2>&1)

) | vv; ok

# Test 3: Large frontmatter exceeding typical flow buffer
it should handle large frontmatter exceeding flow buffer; (
  ((i==X || X<0)) && set -x; set -e

  # Generate large frontmatter (2KB)
  {
    echo "---"
    for i in {1..50}; do
      echo "field_$i: This is a long value for field number $i to create bulk"
    done
    echo "---"
    echo ""
    echo "# Content"
    echo "After large frontmatter"
  } > /tmp/fm_large.md

  # Test with flow smaller than frontmatter
  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_large.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fm_large.md | ./glow -w0 --flow=512 - 2>&1)

) | vv; ok

# Test 4: Malformed frontmatter (missing closing ---)
it should handle malformed frontmatter gracefully; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fm_malformed.md <<'EOF'
---
title: Unclosed Frontmatter
author: Test

# This looks like content but frontmatter never closed

More content here.
EOF

  # Should handle gracefully, treating entire doc as content
  ./glow -w0 --flow=64 /tmp/fm_malformed.md > /dev/null 2>&1
  echo "Exit code: $?"

) | vv; ok

# Test 5: Empty frontmatter
it should handle empty frontmatter; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fm_empty.md <<'EOF'
---
---

# Main Content

Document with empty frontmatter.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_empty.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fm_empty.md | ./glow -w0 --flow=32 - 2>&1)

) | vv; ok

# Test 6: No frontmatter document
it should handle documents without frontmatter; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fm_none.md <<'EOF'
# Document Without Frontmatter

Just regular markdown content here.
No frontmatter at all.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_none.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fm_none.md | ./glow -w0 --flow=32 - 2>&1)

) | vv; ok

# Test 7: Frontmatter with --- in content
it should handle documents with --- in content body; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fm_dashes.md <<'EOF'
---
title: Test
---

# Content

Some text here.

---

More content after horizontal rule.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_dashes.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fm_dashes.md | ./glow -w0 --flow=64 - 2>&1)

) | vv; ok

# Test 8: Streaming frontmatter via stdin
it should handle frontmatter when streaming via stdin; (
  ((i==X || X<0)) && set -x; set -e

  # Create frontmatter content
  content='---
title: Stdin Test
---

# Streamed Content

This comes from stdin.'

  # Compare stdin handling
  # Compare streaming versions instead of with glow.orig
  diff -u <(echo "$content" | ./glow -w0 --flow=0 - 2>&1) <(echo "$content" | ./glow -w0 --flow=32 - 2>&1)

) | vv; ok

# Test 9: Frontmatter boundary exactly at flow size
it should handle frontmatter boundary exactly at flow boundary; (
  ((i==X || X<0)) && set -x; set -e

  # Create 64-byte frontmatter (including delimiters and newlines)
  cat > /tmp/fm_exact.md <<'EOF'
---
title: X
desc: YZ
---

# Content Here

Body text.
EOF

  # Calculate exact size and test
  fm_size=$(head -n 4 /tmp/fm_exact.md | wc -c | tr -d ' ')
  echo "Frontmatter size: $fm_size bytes" >&2

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_exact.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fm_exact.md | ./glow -w0 --flow=$fm_size - 2>&1)

) | vv; ok

# Test 10: Multiple --- markers (code blocks) after frontmatter
it should distinguish frontmatter from code block markers; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fm_code.md <<'EOF'
---
title: Code Example
---

# Example

```yaml
---
config: value
---
```

More content.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fm_code.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fm_code.md | ./glow -w0 --flow=32 - 2>&1)

) | vv; ok

echo "====================================" >&2
echo "Frontmatter test suite complete" >&2
echo "====================================" >&2