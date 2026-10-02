package mermaid

import (
	"strings"
	"testing"
	"time"
)

func TestStressNoPanic(t *testing.T) {
	inputs := []string{
		"flowchart TD\n    A --> B[",
		"flowchart TD\n    A[\"unterminated\n    B --> C",
		"flowchart TD\n    A --> A\n    A --> A\n    A --> A",
		"flowchart TD\n    " + strings.Repeat("A1 --> B1; ", 2000),
		"flowchart TD\n    A --> B\n    B --> A",
		"flowchart RL\n    X --> Y\n    Y --> Z\n    Z --> X\n    Z --> W",
		"flowchart TD\n    A --> B --> C --> D --> E --> F --> G --> H --> I --> J --> K --> L --> M --> N --> O --> P",
		"flowchart TD\n    a[" + strings.Repeat("x", 500) + "] --> b",
		"flowchart TD\n    A -->|piece| B\n    B -->|piece| C\n    C -->|piece| A",
		"flowchart TD\n    A[-] --> B[--] --> C[--x--]",
		"flowchart TD\n    A-- --> B",
		"flowchart TD\n    A ---|x| B",
		"flowchart TD\n    A -->|| B",
		"flowchart TD\n    A((()) --> B",
		"flowchart TD\n    subgraph\n    A --> B",
		"flowchart TD\n    end\n    end\n    end",
		"graph TD\n    A --> B[<br/><br/><br/>]",
		"graph TD\n    A[\"\\\"\"] --> B",
		"flowchart TD\n    0 --> 1\n    1 --> 0",
	}
	for i, src := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("input %d panicked: %v", i, r)
				}
			}()
			done := make(chan struct{})
			go func() {
				defer close(done)
				out, err := Render(src)
				_ = out
				_ = err
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("input %d timed out", i)
			}
		}()
	}
}
