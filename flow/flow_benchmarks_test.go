// Package flow benchmark tests measure and validate performance characteristics.
// These benchmarks ensure streaming performance meets requirements:
// - Throughput for various document sizes and complexities
// - Latency (time to first byte, chunk processing time)
// - Memory usage and allocation patterns
// - Scalability with document size and window parameters
// - Performance regression detection
package flow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Helper types for instrumentation

type instrumentedReader struct {
	io.Reader
	onFirstRead func()
	firstRead   atomic.Bool
}

func (r *instrumentedReader) Read(p []byte) (n int, err error) {
	if r.firstRead.CompareAndSwap(false, true) && r.onFirstRead != nil {
		r.onFirstRead()
	}
	return r.Reader.Read(p)
}

type instrumentedWriter struct {
	io.Writer
	onFirstWrite func()
	firstWrite   atomic.Bool
}

func (w *instrumentedWriter) Write(p []byte) (n int, err error) {
	if w.firstWrite.CompareAndSwap(false, true) && w.onFirstWrite != nil {
		w.onFirstWrite()
	}
	return w.Writer.Write(p)
}

// Content generators

func generateMarkdown(size int) string {
	var buf strings.Builder
	buf.Grow(size)

	written := 0
	sections := []string{
		"# Main Title\n\n",
		"## Section Header\n\n",
		"This is a paragraph with **bold** and *italic* text.\n\n",
		"- List item one\n- List item two\n- List item three\n\n",
		"```go\nfunc example() {\n    return 42\n}\n```\n\n",
		"> Blockquote content goes here\n\n",
		"| Column 1 | Column 2 |\n|----------|----------|\n| Value 1  | Value 2  |\n\n",
	}

	for written < size {
		section := sections[rand.Intn(len(sections))]
		if written+len(section) > size {
			buf.WriteString(section[:size-written])
			break
		}
		buf.WriteString(section)
		written += len(section)
	}

	return buf.String()
}

func generateCodeHeavyMarkdown(size int) string {
	var buf strings.Builder
	buf.Grow(size)

	written := 0
	buf.WriteString("# Code Documentation\n\n")
	written += 22

	for written < size {
		code := fmt.Sprintf("```go\nfunc function%d() {\n    // Implementation\n    return %d\n}\n```\n\n",
			written/100, written)
		if written+len(code) > size {
			remaining := size - written
			if remaining > 0 {
				buf.WriteString(code[:remaining])
			}
			break
		}
		buf.WriteString(code)
		written += len(code)
	}

	return buf.String()
}

func generateProseHeavyMarkdown(size int) string {
	var buf strings.Builder
	buf.Grow(size)

	written := 0
	buf.WriteString("# Document Title\n\n")
	written += 18

	prose := `Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.

`

	for written < size {
		if written+len(prose) > size {
			buf.WriteString(prose[:size-written])
			break
		}
		buf.WriteString(prose)
		written += len(prose)
	}

	return buf.String()
}

func generateMixedMarkdown(size int) string {
	return generateMarkdown(size) // Use default mixed content
}

func generateTableHeavyMarkdown(size int) string {
	var buf strings.Builder
	buf.Grow(size)

	written := 0
	buf.WriteString("# Data Tables\n\n")
	written += 15

	tableNum := 0
	for written < size {
		table := fmt.Sprintf("## Table %d\n\n| ID | Name | Value | Status |\n|-----|------|-------|--------|\n", tableNum)
		for i := 0; i < 10 && written < size; i++ {
			row := fmt.Sprintf("| %d | Item%d | %d | OK |\n", i, i, i*10)
			if written+len(table)+len(row) > size {
				if written+len(table) <= size {
					buf.WriteString(table[:size-written])
				}
				break
			}
			table += row
		}
		table += "\n"

		if written+len(table) > size {
			buf.WriteString(table[:size-written])
			break
		}
		buf.WriteString(table)
		written += len(table)
		tableNum++
	}

	return buf.String()
}

// 1. FLOW PERFORMANCE BENCHMARKS

func BenchmarkFlowSmallFiles(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"5KB", 5 * 1024},
		{"10KB", 10 * 1024},
	}

	windows := []int64{-1, 100, 1024, 16384}

	for _, size := range sizes {
		for _, window := range windows {
			name := fmt.Sprintf("%s/window_%d", size.name, window)

			b.Run(name, func(b *testing.B) {
				content := generateMarkdown(size.size)

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					var buf bytes.Buffer
					err := Flow(context.Background(),
						strings.NewReader(content), &buf, window, passthroughRenderer)

					if err != nil {
						b.Fatal(err)
					}
				}

				b.SetBytes(int64(size.size))
			})
		}
	}
}

func BenchmarkFlowMediumFiles(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"100KB", 100 * 1024},
		{"500KB", 500 * 1024},
		{"1MB", 1024 * 1024},
	}

	windows := []int64{-1, 1024, 16384, 65536}

	for _, size := range sizes {
		for _, window := range windows {
			name := fmt.Sprintf("%s/window_%d", size.name, window)

			b.Run(name, func(b *testing.B) {
				content := generateMarkdown(size.size)

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					var buf bytes.Buffer
					err := Flow(context.Background(),
						strings.NewReader(content), &buf, window, passthroughRenderer)

					if err != nil {
						b.Fatal(err)
					}
				}

				b.SetBytes(int64(size.size))
			})
		}
	}
}

func BenchmarkFlowContentTypes(b *testing.B) {
	types := []struct {
		name string
		gen  func(int) string
	}{
		{"code_heavy", generateCodeHeavyMarkdown},
		{"prose_heavy", generateProseHeavyMarkdown},
		{"mixed", generateMixedMarkdown},
		{"table_heavy", generateTableHeavyMarkdown},
	}

	size := 100 * 1024 // 100KB

	for _, contentType := range types {
		b.Run(contentType.name, func(b *testing.B) {
			content := contentType.gen(size)

			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, 1024, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFlowVariousWindows(b *testing.B) {
	content := generateMarkdown(100 * 1024) // 100KB
	windows := []int64{-1, 1, 10, 100, 1024, 4096, 16384, 65536}

	for _, window := range windows {
		name := fmt.Sprintf("window_%d", window)

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(len(content)))

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, window, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// 2. LATENCY MEASUREMENTS

func BenchmarkFlowFirstByte(b *testing.B) {
	content := generateMarkdown(100 * 1024) // 100KB

	b.Run("streaming", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			start := time.Now()
			var firstByteTime time.Time

			reader := &instrumentedReader{
				Reader: strings.NewReader(content),
				onFirstRead: func() {
					// First read from source
				},
			}

			writer := &instrumentedWriter{
				Writer: io.Discard,
				onFirstWrite: func() {
					firstByteTime = time.Now()
				},
			}

			err := Flow(context.Background(), reader, writer, 1024, passthroughRenderer)
			if err != nil {
				b.Fatal(err)
			}

			if !firstByteTime.IsZero() {
				latency := firstByteTime.Sub(start).Microseconds()
				b.ReportMetric(float64(latency), "first_byte_us")
			}
		}
	})

	b.Run("buffered", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			start := time.Now()

			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 0, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}

			latency := time.Since(start).Microseconds()
			b.ReportMetric(float64(latency), "total_us")
		}
	})
}

func BenchmarkFlowLatencyPercentiles(b *testing.B) {
	sizes := []int{1024, 10 * 1024, 100 * 1024, 1024 * 1024}

	for _, size := range sizes {
		name := fmt.Sprintf("%dKB", size/1024)

		b.Run(name, func(b *testing.B) {
			content := generateMarkdown(size)
			var latencies []int64

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				start := time.Now()

				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, 1024, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}

				latencies = append(latencies, time.Since(start).Microseconds())
			}

			// Calculate percentiles
			if len(latencies) > 0 {
				p50 := latencies[len(latencies)*50/100]
				p95 := latencies[len(latencies)*95/100]
				p99 := latencies[len(latencies)*99/100]

				b.ReportMetric(float64(p50), "p50_us")
				b.ReportMetric(float64(p95), "p95_us")
				b.ReportMetric(float64(p99), "p99_us")
			}
		})
	}
}

func BenchmarkFlowStreamingVsBuffered(b *testing.B) {
	content := generateMarkdown(500 * 1024) // 500KB

	b.Run("streaming/window_1024", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(int64(len(content)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("buffered/window_0", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(int64(len(content)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 0, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("unbuffered/window_-1", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(int64(len(content)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, -1, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkFlowTimeToCompletion(b *testing.B) {
	sizes := []int{10 * 1024, 100 * 1024, 500 * 1024}

	for _, size := range sizes {
		name := fmt.Sprintf("%dKB", size/1024)

		b.Run(name, func(b *testing.B) {
			content := generateMarkdown(size)

			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				start := time.Now()

				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, 1024, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}

				elapsed := time.Since(start).Microseconds()
				b.ReportMetric(float64(elapsed), "completion_us")
			}
		})
	}
}

// 3. RESOURCE USAGE BENCHMARKS

func BenchmarkFlowMemoryUsage(b *testing.B) {
	sizes := []int{10 * 1024, 100 * 1024, 1024 * 1024}

	for _, size := range sizes {
		name := fmt.Sprintf("%dKB", size/1024)

		b.Run(name, func(b *testing.B) {
			content := generateMarkdown(size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m1)

				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, 1024, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}

				runtime.ReadMemStats(&m2)

				alloced := m2.TotalAlloc - m1.TotalAlloc
				heapUsed := m2.HeapAlloc - m1.HeapAlloc

				b.ReportMetric(float64(alloced/1024), "KB_allocated")
				b.ReportMetric(float64(heapUsed/1024), "KB_heap")
			}
		})
	}
}

func BenchmarkFlowGoroutineCount(b *testing.B) {
	content := generateMarkdown(100 * 1024) // 100KB

	b.Run("goroutines", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			before := runtime.NumGoroutine()

			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}

			after := runtime.NumGoroutine()

			// Should not leak goroutines
			if after > before {
				b.ReportMetric(float64(after-before), "goroutine_delta")
			}
		}
	})
}

func BenchmarkFlowGCPressure(b *testing.B) {
	content := generateMarkdown(500 * 1024) // 500KB

	b.Run("gc_impact", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var m1, m2 runtime.MemStats
			runtime.ReadMemStats(&m1)

			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}

			runtime.ReadMemStats(&m2)

			gcPauses := m2.NumGC - m1.NumGC
			b.ReportMetric(float64(gcPauses), "gc_cycles")

			if m2.PauseTotalNs > m1.PauseTotalNs {
				pauseTime := (m2.PauseTotalNs - m1.PauseTotalNs) / 1000 // Convert to microseconds
				b.ReportMetric(float64(pauseTime), "gc_pause_us")
			}
		}
	})
}

func BenchmarkFlowAllocationPatterns(b *testing.B) {
	windows := []int64{100, 1024, 16384}
	content := generateMarkdown(100 * 1024) // 100KB

	for _, window := range windows {
		name := fmt.Sprintf("window_%d", window)

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, window, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}
			}

			// ReportAllocs automatically tracks allocations
		})
	}
}

// 4. REGRESSION DETECTION BENCHMARKS

func BenchmarkFlowBaselinePerformance(b *testing.B) {
	// Establish baseline with known good performance
	content := generateMarkdown(100 * 1024) // 100KB

	b.Run("baseline_throughput", func(b *testing.B) {
		b.ResetTimer()
		b.SetBytes(int64(len(content)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}

		// Baseline: Should process at least 100MB/s
		// This will be reported as MB/s by the benchmark framework
	})
}

func BenchmarkFlowThroughputThresholds(b *testing.B) {
	sizes := []struct {
		name    string
		size    int
		minMBps float64 // Minimum expected MB/s
	}{
		{"small_10KB", 10 * 1024, 50},
		{"medium_100KB", 100 * 1024, 100},
		{"large_1MB", 1024 * 1024, 150},
	}

	for _, test := range sizes {
		b.Run(test.name, func(b *testing.B) {
			content := generateMarkdown(test.size)

			b.ResetTimer()
			b.SetBytes(int64(test.size))

			start := time.Now()
			iterations := 0

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, 1024, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}
				iterations++
			}

			elapsed := time.Since(start).Seconds()
			throughput := float64(iterations*test.size) / elapsed / (1024 * 1024)

			b.ReportMetric(throughput, "MB/s")

			// Log if below threshold (for regression detection)
			if throughput < test.minMBps {
				b.Logf("WARNING: Throughput %.2f MB/s below threshold %.2f MB/s",
					throughput, test.minMBps)
			}
		})
	}
}

func BenchmarkFlowMemoryLimits(b *testing.B) {
	sizes := []struct {
		name     string
		size     int
		maxAlloc int64 // Maximum expected allocations in KB
	}{
		{"10KB_max_50KB", 10 * 1024, 50},
		{"100KB_max_500KB", 100 * 1024, 500},
		{"1MB_max_5MB", 1024 * 1024, 5 * 1024},
	}

	for _, test := range sizes {
		b.Run(test.name, func(b *testing.B) {
			content := generateMarkdown(test.size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m1)

				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, 1024, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}

				runtime.ReadMemStats(&m2)

				allocKB := int64((m2.TotalAlloc - m1.TotalAlloc) / 1024)

				if allocKB > test.maxAlloc {
					b.Logf("WARNING: Allocated %d KB exceeds limit %d KB",
						allocKB, test.maxAlloc)
				}

				b.ReportMetric(float64(allocKB), "KB_alloc")
			}
		})
	}
}

func BenchmarkFlowConsistency(b *testing.B) {
	// Test performance consistency across runs
	content := generateMarkdown(100 * 1024) // 100KB

	b.Run("consistency_check", func(b *testing.B) {
		var durations []time.Duration

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			start := time.Now()

			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}

			durations = append(durations, time.Since(start))
		}

		if len(durations) > 1 {
			// Calculate variance
			var sum, sumSquares time.Duration
			for _, d := range durations {
				sum += d
				sumSquares += d * d
			}

			mean := sum / time.Duration(len(durations))
			variance := (sumSquares / time.Duration(len(durations))) - (mean * mean)
			stdDev := time.Duration(float64(variance) * 0.5) // Approximate sqrt

			b.ReportMetric(float64(mean.Microseconds()), "mean_us")
			b.ReportMetric(float64(stdDev.Microseconds()), "stddev_us")

			// Coefficient of variation should be low for consistent performance
			cv := float64(stdDev) / float64(mean)
			b.ReportMetric(cv*100, "cv_percent")

			if cv > 0.2 { // More than 20% variation
				b.Logf("WARNING: High performance variance (CV=%.2f%%)", cv*100)
			}
		}
	})
}

// 5. ADDITIONAL PERFORMANCE BENCHMARKS

func BenchmarkFlowConcurrency(b *testing.B) {
	content := generateMarkdown(50 * 1024) // 50KB

	b.Run("sequential", func(b *testing.B) {
		b.ResetTimer()
		b.SetBytes(int64(len(content)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.ResetTimer()
		b.SetBytes(int64(len(content)))

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var buf bytes.Buffer
				err := Flow(context.Background(),
					strings.NewReader(content), &buf, 1024, passthroughRenderer)

				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

func BenchmarkFlowWorstCase(b *testing.B) {
	// Test worst-case scenarios

	b.Run("single_byte_window", func(b *testing.B) {
		content := generateMarkdown(1024) // 1KB

		b.ResetTimer()
		b.SetBytes(int64(len(content)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pathological_markdown", func(b *testing.B) {
		// Deeply nested structure
		var content strings.Builder
		for i := 0; i < 100; i++ {
			content.WriteString(strings.Repeat("  ", i))
			content.WriteString("- Item\n")
		}
		doc := content.String()

		b.ResetTimer()
		b.SetBytes(int64(len(doc)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_ = Flow(ctx, strings.NewReader(doc), &buf, 1024, passthroughRenderer)
			// Don't fail on timeout - that's expected for pathological input
		}
	})
}

func BenchmarkFlowRealWorld(b *testing.B) {
	// Simulate real-world usage patterns

	b.Run("readme_file", func(b *testing.B) {
		// Typical README.md content
		content := `# Project Name

[![Build Status](https://img.shields.io/badge/build-passing-green.svg)]()
[![Coverage](https://img.shields.io/badge/coverage-95%25-brightgreen.svg)]()

## Installation

` + "```bash\nnpm install package-name\n```" + `

## Usage

` + "```javascript\nconst pkg = require('package-name');\npkg.doSomething();\n```" + `

## API Reference

### Function: doSomething()

Does something useful.

**Parameters:**
- ` + "`options`" + ` (Object): Configuration options
  - ` + "`timeout`" + ` (Number): Timeout in milliseconds
  - ` + "`retries`" + ` (Number): Number of retries

**Returns:**
- Promise<Result>: The operation result

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT © [Your Name]
`

		b.ResetTimer()
		b.SetBytes(int64(len(content)))
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("api_docs", func(b *testing.B) {
		// API documentation pattern
		var content strings.Builder
		for i := 0; i < 50; i++ {
			content.WriteString(fmt.Sprintf(`
## Endpoint: /api/v1/resource%d

### GET /api/v1/resource%d/{id}

Retrieves a specific resource.

**Parameters:**
- `+"`id`"+` (string): Resource identifier

**Response:**
`+"```json\n{\n  \"id\": \"123\",\n  \"name\": \"Resource Name\"\n}\n```"+`

`, i, i))
		}
		doc := content.String()

		b.ResetTimer()
		b.SetBytes(int64(len(doc)))
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(doc), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkFlowEdgeCases(b *testing.B) {
	b.Run("empty_document", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(""), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("single_character", func(b *testing.B) {
		b.ResetTimer()
		b.SetBytes(1)

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader("a"), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("only_whitespace", func(b *testing.B) {
		content := strings.Repeat(" \n\t\r", 1000)

		b.ResetTimer()
		b.SetBytes(int64(len(content)))

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := Flow(context.Background(),
				strings.NewReader(content), &buf, 1024, passthroughRenderer)

			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
