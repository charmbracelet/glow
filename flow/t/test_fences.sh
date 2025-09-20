#!/bin/bash

set -uo pipefail

# Test framework setup
i=0 x=-1 a=-1 y=-1 n=0 all=-1 yes=-1 no=0 on=-1 off=0
[[ ${V-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || V=y
[[ ${X-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || X=n

it () { echo -e "$(printf '%.0s=' {1..79})\nFENCE TEST $((++i)) $*\n$(printf '%.0s-' {1..79})"; } 1>&2
ok () { echo FENCE TEST $i $((( $? )) && echo ❌ || echo ✅; echo " V=$V X=$X"); } 1>&2
vv () { ((i==V || V<0)) && cat -u || cat -u > /dev/null; }

# Build glow
go build .
cp -avn ./glow ./glow.orig >&2 || true
timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3


# Test 1: Simple triple fence code block
it should handle simple triple fence code blocks; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_simple.md <<'EOF'
# Simple Code

```python
def hello():
    print("Hello")
```

Done.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_simple.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_simple.md | ./glow -w0 --flow=-1 - 2>&1)

) | vv; ok

# Test 2: Triple fence inside quadruple fence (nested)
it should handle triple fence nested inside quadruple fence; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_nested_3in4.md <<'EOF'
# Nested Example

````markdown
Here's how to use code blocks:

```python
print("This is inside the markdown example")
```

The above is a Python example.
````

After the example.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_nested_3in4.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_nested_3in4.md | ./glow -w0 --flow=256 - 2>&1)

) | vv; ok

# Test 3: Quadruple fence inside quintuple fence
it should handle quadruple fence nested inside quintuple fence; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_nested_4in5.md <<'EOF'
# Deep Nesting

`````markdown
Documentation example:

````python
```bash
echo "Very nested"
```
Still in Python!
````

End of documentation.
`````

All done.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_nested_4in5.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_nested_4in5.md | ./glow -w0 --flow=128 - 2>&1)

) | vv; ok

# Test 4: Multiple same-level fences in sequence
it should handle multiple same-level fences correctly; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_multiple_same.md <<'EOF'
# Multiple Blocks

```python
first = 1
```

```javascript
const second = 2;
```

```bash
third=3
```

Done with examples.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_multiple_same.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_multiple_same.md | ./glow -w0 --flow=-1 - 2>&1)

) | vv; ok

# Test 5: Mixed fence levels in document
it should handle mixed fence levels throughout document; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_mixed_levels.md <<'EOF'
# Mixed Levels

```python
simple = "code"
```

Text between.

````markdown
```javascript
nested = "example";
```
````

More text.

`````complex
````nested
```deep
content
```
````
`````

Final text.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_mixed_levels.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_mixed_levels.md | ./glow -w0 --flow=512 - 2>&1)

) | vv; ok

# Test 6: Unclosed outer fence with closed inner fence (edge case)
it should handle unclosed outer fence with closed inner correctly; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_unclosed_outer.md <<'EOF'
# Unclosed Outer

````markdown
This starts a quad fence.

```python
print("Inner triple fence")
```

Inner is closed but outer is not...
This should all be treated as code since outer never closes.
EOF

  # Both should treat everything after ```` as code block
  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_unclosed_outer.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_unclosed_outer.md | ./glow -w0 --flow=256 - 2>&1)

) | vv; ok

# Test 7: Fence boundary at exact flow boundary
it should handle fence boundaries at flow boundaries correctly; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_flow_boundary.md <<'EOF'
# Flow Test

```python
# This is a longer code block
# with multiple lines
# to test flow boundaries
def example():
    return 42
```

After code.
EOF

  # Test with small flow that might split the fence
  # Compare streaming with different buffer sizes - they should be consistent
  diff -u <(cat /tmp/fence_flow_boundary.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_flow_boundary.md | ./glow -w0 --flow=50 - 2>&1)

) | vv; ok

# Test 8: Empty fence blocks
it should handle empty fence blocks correctly; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_empty.md <<'EOF'
# Empty Blocks

```
```

````
````

Text after empty blocks.
EOF

  diff -u <(./glow.orig -w0 /tmp/fence_empty.md 2>&1) <(./glow -w0 --flow=-1 /tmp/fence_empty.md 2>&1)

) | vv; ok

# Test 9: Fence with language specifier on same line
it should handle fences with inline language specifiers; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_lang.md <<'EOF'
# Language Specs

```python
code = "python"
```

````markdown
```javascript
nested = true;
```
````

Done.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_lang.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_lang.md | ./glow -w0 --flow=128 - 2>&1)

) | vv; ok

# Test 10: Complex real-world example with multiple nesting levels
it should handle complex real-world nested fence example; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_complex.md <<'EOF'
# Documentation

Here's how to document code examples:

`````markdown
# API Documentation

To use our API, include code like this:

````javascript
// Initialize the client
const client = new APIClient({
  endpoint: 'https://api.example.com'
});

// Make a request with error handling
```javascript
try {
  const result = await client.get('/users');
  console.log(result);
} catch (error) {
  console.error('Failed:', error);
}
```

// The above shows error handling
````

You can also use Python:

```python
# Python example
client = APIClient()
result = client.get('/users')
```

End of examples.
`````

That's how you document APIs.
EOF

  # Compare streaming with different buffer sizes - they should be consistent
  diff -u <(cat /tmp/fence_complex.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_complex.md | ./glow -w0 --flow=256 - 2>&1)

) | vv; ok

# Test 11: Adjacent fences with different levels
it should handle adjacent fences with different levels; (
  ((i==X || X<0)) && set -x; set -e

  cat > /tmp/fence_adjacent.md <<'EOF'
# Adjacent Fences

```
triple
```
````
quadruple
````
`````
quintuple
`````
````
quad again
````
```
triple again
```

Done.
EOF

  # Compare streaming versions instead of with glow.orig
  diff -u <(cat /tmp/fence_adjacent.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_adjacent.md | ./glow -w0 --flow=-1 - 2>&1)

) | vv; ok

# Test 12: Streaming with fence split across chunks
it should handle fences split across streaming chunks; (
  ((i==X || X<0)) && set -x; set -e

  # Create content that will definitely split with small flow
  {
    echo "# Split Test"
    echo ""
    echo '```python'
    for i in {1..20}; do
      echo "line$i = $i  # This is line $i of the code"
    done
    echo '```'
    echo ""
    echo "After the code block."
  } > /tmp/fence_split.md

  # Test with very small flow to force splitting
  # Compare streaming with different buffer sizes - they should be consistent
  diff -u <(cat /tmp/fence_split.md | ./glow -w0 --flow=0 - 2>&1) <(cat /tmp/fence_split.md | ./glow -w0 --flow=32 - 2>&1)

) | vv; ok

echo "====================================" >&2
echo "Fence test suite complete" >&2
echo "====================================" >&2