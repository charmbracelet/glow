# glow --flow

Read this document to better understand or further modify Glow's streaming `--flow` implementation.

## Package Overview

The Glow streaming markdown renderer transforms large documents incrementally while preserving content integrity and system resource boundaries. This architecture balances three critical concerns: **user experience** (transparent, fast rendering), **content fidelity** (zero corruption tolerance), and **system reliability** (bounded resources, graceful degradation).

### Design Principles

- **Separation of Concerns**: Each component has ONE primary responsibility
- **Interface Segregation**: Components depend on minimal, focused contracts
- **Dependency Inversion**: High-level logic doesn't depend on low-level details
- **Open-Closed Principle**: Open for extension, closed for modification
- **Failure Isolation**: Component failures don't cascade through the system
- **Observable Decisions**: Every architectural choice is measurable and auditable
- **Evolutionary Design**: Architecture enables change without wholesale rewrites

### Development Requirements

- **Incremental Processing**: Components support partial input/output for true streaming
- **Progressive Rendering**: Content flushed at appropriate `--flow`-mediated boundaries
- **Bounded Resources**: Hard limits on memory and CPU prevent system exhaustion
- **Content Integrity**: Structured content (YAML, code blocks) never corrupted
- **Observable Behavior**: Every decision point debuggable with environment variables
- **Progressive Enhancement**: Degrade quality before failing completely

### Global Invariants

These invariants MUST hold true regardless of implementation changes:

```console
∀ input, flow_mode → output(input, flow_mode) ≡ output(input, buffered)
∀ chunk_size → memory_usage ≤ O(min(chunk_size, document_size))
∀ fence_start, fence_end → never_split_between(fence_start, fence_end)
∀ error → error_visible ∧ error_actionable
∀ resource_limit → graceful_degradation_before_failure
```

### Design Overview

#### System Architecture

```console
Input Stream → Chunk Accumulator → Boundary Detector → Renderer → Output Stream
     ↑              ↑                    ↑                ↑           ↑
     └──────────────┴────────────────────┴────────────────┴───────────┘
                        Backpressure & Flow Control
```

#### Component Contracts

##### Current Implementation

- **Chunk Accumulator**: Manages windowed buffer with configurable size limits
- **Boundary Detector**: Ensures structural integrity of markdown elements
- **Renderer (Glamour)**: Stateless transformation of markdown to terminal output
- **Flow Controller**: Orchestrates pipeline stages and handles backpressure

### Known Trade-offs

#### Stateless Chunk Rendering

- **Pro**: Each chunk independently processable (parallelizable)
- **Pro**: No accumulation of state reduces memory footprint
- **Con**: Cannot resolve forward references across chunks
- **Con**: Glamour quirks from isolated content rendering
- **Mitigation**: Smart boundary detection minimizes isolated content

#### Fence-Aware Splitting

- **Pro**: Preserves code block integrity (critical for users)
- **Pro**: Maintains syntax highlighting correctness
- **Con**: May require larger chunks when fences span boundaries
- **Con**: Complex state tracking across boundaries
- **Mitigation**: Fence stack tracking with proper depth management

## Reference Links

**Key Insight**: When markdown reference links `[ref]: url` are rendered in isolation (split from surrounding content), glamour produces phantom blank lines that disappear when rendered together. This explains many streaming inconsistencies and led to the skip-isolated-content optimization.

**Implications**: Cannot be fixed without modifying glamour itself. Requires smart boundary detection to minimize content isolation. Demonstrates why streaming markdown is fundamentally different from batch rendering. Needs more investigation; should we pass ref links through instead of being dropping them post-glamour to avoid information loss?

### Allowable Differences

Some flow mode differences stem from:

- **Reference link resolution boundaries**: Small buffers can't resolve forward references
- **Structural formatting**: Glamour renders isolated content differently than connected content
- **Acceptable variation**: These differences are *visually* near-identical but byte-different
- **Test strategy**: Allow reference-link related differences in test validation
- **CRITICAL**: *Whitespace* differences are **not** allowable differences!

## User Experience Principles

### Core User Expectations

- **Performance First**: Users value responsiveness over pixel-perfect formatting
- **Content Fidelity**: Zero tolerance for corrupted YAML, code blocks, or structured content
- **Transparent Streaming**: Users should not notice the difference between streaming and buffered modes
- **Graceful Degradation**: Better to show content with minor formatting variations than fail entirely

### Primary Workflow Support

- **Live Preview**: Real-time editing feedback for content creators
- **Documentation Reading**: Large README files, API docs, technical documentation
- **Configuration Review**: YAML configs, CI/CD files, structured data in markdown
- **Terminal Integration**: Seamless work with pagers and TUI workflows

### Impact Assessment Framework

Every change must consider:

- **Content Trust**: Can users rely on the output being accurate?
- **Workflow Disruption**: Will this break how users currently work?
- **Performance Perception**: Does this feel fast and responsive?
- **Error Recovery**: How do users recover from edge cases?

### Content Integrity Standards

- **YAML/JSON Configuration**: Zero tolerance for splitting structured data mid-key/value
- **Frontmatter**: Handle arbitrary-length metadata without corruption
- **Code Blocks**: Never break syntax highlighting or corrupt indentation
- **Tables and Lists**: Preserve formatting hierarchy and visual structure
- **Reference Links**: Maintain link resolution consistency across all modes

### Performance Perception Guidelines

- **First Byte Latency**: Users notice delays over 100ms - optimize for immediate response
- **Progressive Rendering**: Show content as it arrives, don't wait for completion
- **Memory Efficiency**: Large documents should not consume unbounded memory
- **Responsive Feel**: Streaming should feel faster than traditional buffered rendering
- **Terminal Integration**: Must work seamlessly with pagers (less, more) and TUI tools

## Development Commands

### Build and Test

**ALWAYS** from the repository root:

```bash
GLOW_TEST_LOG=1 flow/t/test.sh go      # Build, run go tests; LOUD
GLOW_TEST_MAX_FAILED=0 flow/t/test.sh  # Build, run ALL tests; FAST
GLOW_TEST_LOG=1 flow/t/test.sh         # Build, run ALL tests; LOUD
flow/t/test.sh                         # Build, run ALL tests; QUIET
flow/t/test.sh flow/t/test_arch.sh     # Build, run arch tests; QUIET
flow/t/test.sh go flow/t/test_arch.sh  # Build, run go tests, run arch tests; QUIET
```

### Debug Mode

```bash
GLOW_FLOW_DEBUG=1 go test ./flow/ -v   # Enable debug output
GLOW_TEST_LOG=1 flow/t/test.sh         # Verbose test logging

# Debug specific issues
GLOW_FLOW_DEBUG=1 echo "test" | ./glow --flow=-1 - 2>&1 | grep BOUNDARY
GLOW_FLOW_DEBUG=1 cat README.md | ./glow --flow=16 - 2>&1 | grep FINAL

# Compare flow modes for consistency
diff <(cat test.md | ./glow --flow=-1 -) <(cat test.md | ./glow --flow=0 -)

# Find where content gets corrupted
GLOW_FLOW_DEBUG=1 cat problematic.md | ./glow --flow=16 - 2>&1 | grep -E "BOUNDARY|SPLIT|FINAL"
```

### Performance Analysis

```bash
# Memory profiling
go test -memprofile=mem.prof -bench=. ./flow/...
go tool pprof mem.prof

# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./flow/...
go tool pprof cpu.prof

# Benchmark different buffer sizes
for size in -1 16 100 1024; do
    time cat large.md | ./glow --flow=$size - > /dev/null
done

# Check memory usage
time -l ./glow --flow=1024 huge.md

# Profile hot paths
go test -bench=. -cpuprofile=cpu.prof ./flow/...
go tool pprof -http=:8080 cpu.prof
```

## Streaming Architecture Principles

### Flow Control Modes

**Critical requirement**: All modes MUST produce identical WHITESPACE output!

- **Unbuffered (-1)**: Immediate output at safe boundaries for minimal latency
- **Buffered (0)**: Complete processing before output for optimal throughput
- **Windowed (>0)**: Bounded memory usage with configurable limits (bytes)

### Markdown Renderer Integration

- **Glamour blank sequences**: Renderer produces `\n  \n` patterns that need normalization
- **Trailing newline consistency**: Renderer requires consistent input termination; unconditionally passing a trailing newlines for normalization has proven successful
- **Newline normalization strategy**: Adding trailing newlines unconditionally prevents glamour inconsistencies and ensures predictable rendering behavior
- **Isolated reference links**: Standalone `[ref]: url` lines create phantom whitespace when document is split
- **Output normalization**: Convert renderer-specific patterns to consistent format

### Safe Boundary Detection

- **NEVER split inside code blocks (for now)**: Preserve syntax highlighting integrity
- **NEVER split inside tables (for now)**: Preserve structural integrity of its content
- **Boundary hierarchy**: Prefer paragraph breaks, then line breaks, then forced splits
- **Fence state tracking**: Monitor opening/closing of code fences throughout document
- **Content preservation**: Avoid splitting YAML frontmatter or structured content
- **Small buffer protection**: Prevent splitting very small chunks that cause renderer inconsistencies

### Content Splitting Strategy

- **Implicit closing of blocks**: *Whitespace aside*, only the last block accumulates can rarely switch type
- **Minimal peek to guess**: Every line classified based on its prefix and potentially deferred
- **Minimal wait to know**: Every line that cannot be fully classified should only need one more line
- **Content in isolation**: Skip rendering standalone reference links when document was split
- **Preservation of state**: Maintain fence tracking depth across chunk boundaries
- **Validation**: Test every potential split point for content integrity
- **Fallback behavior**: Return larger chunks if no safe boundary exists
- **Safety behavior**: Force chunking when no safe boundary emerges
- **Self correct**: Limit everything and passthrough unsound input

## Testing Strategy

### Test Categories

- **Unit tests**: Core streaming logic and boundary detection
- **Integration tests**: Full pipeline with real markdown renderer
- **Edge case tests**: Empty input, large files, malformed content
- **Consistency tests**: Verify identical output across all flow modes
- **Pathological tests**: Memory bombs, infinite streams, malformed fences
- **Regression tests**: Previously fixed issues stay fixed

### Debugging Approach

- **Golden file comparison**: Compare expected vs actual output (todo)
- **Environment variables**: Enable debug output for streaming analysis
- **Whitespace normalization**: Handle renderer-specific formatting quirks
- **Progressive validation**: Test each boundary decision independently
- **Differential debugging**: Compare flow=-1 vs flow=0 for inconsistencies

### Known Test Challenges

- **Reference link resolution**: Small buffers can't resolve ref links
- **Code block boundaries**: Space-filled lines after blocks might vary
- **Trailing spaces are already confusing** Force zero-width (i.e. `-w0`)

## Performance Characteristics

### Memory Usage

- **Bounded memory**: Streaming prevents loading entire documents into memory
- **Configurable limits**: Window size controls maximum memory footprint
- **Resource protection**: Hard limits prevent runaway memory consumption

### Latency vs Throughput

- **Unbuffered mode**: Prioritizes immediate output over efficiency
- **Buffered mode**: Optimizes for overall throughput
- **Windowed mode**: Balances memory usage with performance

### Performance Trade-offs

- **Test complexity**: Multiple modes require extensive validation
- **Implementation complexity**: Streaming adds boundary detection overhead
- **Renderer integration**: External dependencies may have quirks requiring workarounds

## Development Workflow

### Development Cycle

- **Understand the failure**: Use `GLOW_FLOW_DEBUG=1` to trace execution
- **Reproduce minimally**: Create smallest test case that shows issue
- **Fix incrementally**: Small changes with immediate validation
- **Document gotchas**: Update this file with learnings

### Code Review Checklist

- [ ] Test coverage includes edge cases
- [ ] Glamour quirks documented if found
- [ ] Context cancellation handled correctly
- [ ] EPIPE errors handled for pipe scenarios
- [ ] Fence state tracking maintained throughout
- [ ] Performance impact measured (memory & CPU)
- [ ] Memory bounds enforced (no unbounded accumulation)

### Common Pitfalls to Avoid

- **Unbounded accumulation**: Always have a forced flush threshold
- **Assuming whitespace**: Glamour varies based on context and document
- **Ignoring glamour quirks**: Test isolated content rendering separately
- **Forgetting EPIPE handling**: Piped commands will crash unexpectedly
- **Boolean fence tracking**: Use stack/depth for proper nesting of quad fences
- **Re-processing entire buffer**: Always track offset for incremental processing

### Engineering Principles

- **Fail loudly in development and gracefully in production**
- **Streaming performance over formatting perfection**
- **User workflow preservation over code elegance**
- **Test what users see and not internal state**
- **Memory safety over perfect output**

### If You Remember Nothing Else

- **DEBUG with `GLOW_FLOW_DEBUG=1`**
- **NEVER split inside fences or frontmatter**
- **REFERENCE link differences are acceptable**
- **SMALL buffers (under 100) are not special**
- **CONSIDER prefix/suffix normalization for Glamour**
- **READ and NOTE other Glamour quirks documented HERE**

---

*This document is the authorial constitution for the streaming markdown system; it's the how, the why, and the where to.*
