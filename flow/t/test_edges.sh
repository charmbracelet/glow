#!/bin/bash

set -uo pipefail

i=0 x=-1 a=-1 y=-1 n=0 all=-1 yes=-1 no=0 on=-1 off=0
[[ ${V-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || V=y
[[ ${X-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || X=n

it () { echo -e "$(printf '%.0s=' {1..79})\nEDGE_TEST $((++i)) $*\n$(printf '%.0s-' {1..79})"; } 1>&2
ok () { echo EDGE_TEST $i $((( $? )) && echo ❌ || echo ✅; echo " V=$V X=$X"); } 1>&2
vv () { ((i==V || V<0)) && cat -u || cat -u > /dev/null; }

go build .
cp -avn ./glow ./glow.orig >&2 || true
timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3


# === FENCE COUNTING EDGE CASES ===

it should handle unclosed quadruple fence with embedded triple fences; (
  ((i==X || X<0)) && set -x; set -e

  # This tests if implementation incorrectly counts ALL backticks together
  # 4 opening + 3 + 3 = 10 (even) would be wrong - quadruple fence is still open!
  cat > /tmp/fence_bug.md << 'EOF'
# Documentation

````markdown
Example with complete code block:
```python
print("hello")
```
But quadruple fence never closes
EOF

  # Should handle unclosed fence gracefully
  timeout 1s cat /tmp/fence_bug.md | ./glow -w0 --flow=50 -w0 - > /tmp/fence_out.md 2>&1 || true

  # Output should exist and contain content
  test -s /tmp/fence_out.md && grep -q "hello" /tmp/fence_out.md

) | vv; ok

it should track fence levels independently not total count; (
  ((i==X || X<0)) && set -x; set -e

  # 4 + 3 + 3 + 4 = 14 (even) but fences are properly nested and closed
  cat > /tmp/fence_levels.md << 'EOF'
````markdown
```python
code1
```
```javascript
code2
```
````
After all fences
EOF

  # Different flush sizes should produce identical output
  cat /tmp/fence_levels.md | ./glow -w0 --flow=30 - > /tmp/level1.md 2>&1
  cat /tmp/fence_levels.md | ./glow -w0 --flow=-1 - > /tmp/level2.md 2>&1

  diff -u /tmp/level1.md /tmp/level2.md

) | vv; ok

# === TABLE EDGE CASES ===

it should handle table alignment markers correctly; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/aligned.md << 'EOF'
| Left | Center | Right |
|:-----|:------:|------:|
| L1   | C1     | R1    |
| L2   | C2     | R2    |
EOF

  # Small buffer might split table
  cat /tmp/aligned.md | ./glow -w0 --flow=20 - > /tmp/align1.md 2>&1
  cat /tmp/aligned.md | ./glow -w0 --flow=-1 - > /tmp/align2.md 2>&1

  # Both should produce valid output
  test -s /tmp/align1.md && test -s /tmp/align2.md

) | vv; ok

it should handle escaped pipes in table cells; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/pipes.md << 'EOF'
| Command | Description |
|---------|-------------|
| `ls \| grep` | Uses pipe |
| `cat \| sed` | Also pipe |
EOF

  cat /tmp/pipes.md | ./glow -w0 --flow=30 - > /tmp/pipes.md 2>&1

  # Should produce output without errors
  test -s /tmp/pipes.md

) | vv; ok

it should not render tables inside code blocks as tables; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/code_table.md << 'EOF'
```markdown
| Col1 | Col2 |
|------|------|
| A    | B    |
```
EOF

  cat /tmp/code_table.md | ./glow -w0 --flow=15 - > /tmp/ct1.md 2>&1
  cat /tmp/code_table.md | ./glow -w0 --flow=-1 - > /tmp/ct2.md 2>&1

  # Both should treat table as code, producing identical output
  diff -u /tmp/ct1.md /tmp/ct2.md

) | vv; ok

it should handle table split at row boundary; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/split_table.md << 'EOF'
| A | B | C |
|---|---|---|
| 1 | 2 | 3 |
| 4 | 5 | 6 |
EOF

  # Force split mid-table
  cat /tmp/split_table.md | ./glow -w0 --flow=25 - > /tmp/split1.md 2>&1
  cat /tmp/split_table.md | ./glow -w0 --flow=-1 - > /tmp/split2.md 2>&1

  # Should handle gracefully even if output differs
  test -s /tmp/split1.md && test -s /tmp/split2.md

) | vv; ok

# === MIXED COMPLEXITY CASES ===

it should handle code block containing fence examples; (
  ((i==X || X<0)) && set -x; set -e

  # Code block showing how to use code blocks - meta!
  cat > /tmp/meta.md << 'EOF'
Here's how to write code blocks:

````markdown
Use three backticks:
```
code here
```
````

Regular text after.
EOF

  cat /tmp/meta.md | ./glow -w0 --flow=40 - > /tmp/meta1.md 2>&1
  cat /tmp/meta.md | ./glow -w0 --flow=-1 - > /tmp/meta2.md 2>&1

  # Should properly close all fences
  diff -u /tmp/meta1.md /tmp/meta2.md

) | vv; ok

it should handle deeply nested fence structures; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/deep.md << 'EOF'
`````markdown
````markdown
```javascript
console.log("deep");
```
````
`````
EOF

  cat /tmp/deep.md | ./glow -w0 --flow=20 - > /tmp/deep.md 2>&1

  # Should complete without hanging
  test -s /tmp/deep.md

) | vv; ok

# === CLEANUP ===
rm -f /tmp/fence_*.* /tmp/level*.* /tmp/align*.* /tmp/pipes.* /tmp/ct*.* /tmp/split*.* /tmp/meta*.* /tmp/deep.*