package flow

import (
	"testing"
)

// TestFence1SimpleTripleFence tests simple triple fence code blocks
func TestFence1SimpleTripleFence(t *testing.T) {
	input := `# Simple Code

` + "```python" + `
def hello():
    print("Hello")
` + "```" + `

Done.`

	// Compare patient vs aggressive streaming - output must be identical
	patientOutput := runFlow(t, input, 0)     // patient mode
	aggressiveOutput := runFlow(t, input, -1) // aggressive mode

	if patientOutput != aggressiveOutput {
		t.Errorf("Simple triple fence output differs between flow modes:\nPatient: %q\nAggressive: %q",
			patientOutput, aggressiveOutput)
	}
}

// TestFence2TripleInQuadruple tests triple fence nested inside quadruple fence
func TestFence2TripleInQuadruple(t *testing.T) {
	input := `# Nested Example

` + "````markdown" + `
Here's how to use code blocks:

` + "```python" + `
print("This is inside the markdown example")
` + "```" + `

The above is a Python example.
` + "````" + `

After the example.`

	// Compare patient vs buffered streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	bufferedOutput := runFlow(t, input, 256)

	if patientOutput != bufferedOutput {
		t.Errorf("Triple in quadruple fence output differs between flow modes:\nPatient: %q\nBuffered: %q",
			patientOutput, bufferedOutput)
	}
}

// TestFence3QuadrupleInQuintuple tests quadruple fence nested inside quintuple fence
func TestFence3QuadrupleInQuintuple(t *testing.T) {
	input := `# Deep Nesting

` + "`````markdown" + `
Documentation example:

` + "````python" + `
` + "```bash" + `
echo "Very nested"
` + "```" + `
Still in Python!
` + "````" + `

End of documentation.
` + "`````" + `

All done.`

	// Compare patient vs buffered streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	bufferedOutput := runFlow(t, input, 128)

	if patientOutput != bufferedOutput {
		t.Errorf("Quadruple in quintuple fence output differs between flow modes:\nPatient: %q\nBuffered: %q",
			patientOutput, bufferedOutput)
	}
}

// TestFence4MultipleSameLevel tests multiple same-level fences in sequence
func TestFence4MultipleSameLevel(t *testing.T) {
	input := `# Multiple Blocks

` + "```python" + `
first = 1
` + "```" + `

` + "```javascript" + `
const second = 2;
` + "```" + `

` + "```bash" + `
third=3
` + "```" + `

Done with examples.`

	// Compare patient vs aggressive streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	aggressiveOutput := runFlow(t, input, -1)

	if patientOutput != aggressiveOutput {
		t.Errorf("Multiple same-level fence output differs between flow modes:\nPatient: %q\nAggressive: %q",
			patientOutput, aggressiveOutput)
	}
}

// TestFence5MixedLevels tests mixed fence levels throughout document
func TestFence5MixedLevels(t *testing.T) {
	input := `# Mixed Levels

` + "```python" + `
simple = "code"
` + "```" + `

Text between.

` + "````markdown" + `
` + "```javascript" + `
nested = "example";
` + "```" + `
` + "````" + `

More text.

` + "`````complex" + `
` + "````nested" + `
` + "```deep" + `
content
` + "```" + `
` + "````" + `
` + "`````" + `

Final text.`

	// Compare patient vs buffered streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	bufferedOutput := runFlow(t, input, 512)

	if patientOutput != bufferedOutput {
		t.Errorf("Mixed levels fence output differs between flow modes:\nPatient: %q\nBuffered: %q",
			patientOutput, bufferedOutput)
	}
}

// TestFence6UnclosedOuter tests unclosed outer fence with closed inner fence (edge case)
func TestFence6UnclosedOuter(t *testing.T) {
	input := `# Unclosed Outer

` + "````markdown" + `
This starts a quad fence.

` + "```python" + `
print("Inner triple fence")
` + "```" + `

Inner is closed but outer is not...
This should all be treated as code since outer never closes.`

	// Both should treat everything after ```` as code block
	// Compare patient vs buffered streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	bufferedOutput := runFlow(t, input, 256)

	if patientOutput != bufferedOutput {
		t.Errorf("Unclosed outer fence output differs between flow modes:\nPatient: %q\nBuffered: %q",
			patientOutput, bufferedOutput)
	}
}

// TestFence7FlowBoundary tests fence boundaries at exact flow boundary
func TestFence7FlowBoundary(t *testing.T) {
	input := `# Flow Test

` + "```python" + `
# This is a longer code block
# with multiple lines
# to test flow boundaries
def example():
    return 42
` + "```" + `

After code.`

	// Test with small flow that might split the fence
	// Compare patient vs small buffer streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	smallBufferOutput := runFlow(t, input, 50)

	if patientOutput != smallBufferOutput {
		t.Errorf("Fence at flow boundary output differs between flow modes:\nPatient: %q\nSmall buffer: %q",
			patientOutput, smallBufferOutput)
	}
}

// TestFence8EmptyBlocks tests empty fence blocks
func TestFence8EmptyBlocks(t *testing.T) {
	input := `# Empty Blocks

` + "```" + `
` + "```" + `

` + "````" + `
` + "````" + `

Text after empty blocks.`

	// Compare patient vs aggressive streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	aggressiveOutput := runFlow(t, input, -1)

	if patientOutput != aggressiveOutput {
		t.Errorf("Empty blocks fence output differs between flow modes:\nPatient: %q\nAggressive: %q",
			patientOutput, aggressiveOutput)
	}
}

// TestFence9LanguageSpecifiers tests fences with inline language specifiers
func TestFence9LanguageSpecifiers(t *testing.T) {
	input := `# Language Specs

` + "```python" + `
code = "python"
` + "```" + `

` + "````markdown" + `
` + "```javascript" + `
nested = true;
` + "```" + `
` + "````" + `

Done.`

	// Compare patient vs buffered streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	bufferedOutput := runFlow(t, input, 128)

	if patientOutput != bufferedOutput {
		t.Errorf("Language specifiers fence output differs between flow modes:\nPatient: %q\nBuffered: %q",
			patientOutput, bufferedOutput)
	}
}

// TestFence10ComplexRealWorld tests complex real-world nested fence example
func TestFence10ComplexRealWorld(t *testing.T) {
	input := `# Documentation

Here's how to document code examples:

` + "`````markdown" + `
# API Documentation

To use our API, include code like this:

` + "````javascript" + `
// Initialize the client
const client = new APIClient({
  endpoint: 'https://api.example.com'
});

// Make a request with error handling
` + "```javascript" + `
try {
  const result = await client.get('/users');
  console.log(result);
} catch (error) {
  console.error('Failed:', error);
}
` + "```" + `

// The above shows error handling
` + "````" + `

You can also use Python:

` + "```python" + `
# Python example
client = APIClient()
result = client.get('/users')
` + "```" + `

End of examples.
` + "`````" + `

That's how you document APIs.`

	// Compare patient vs buffered streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	bufferedOutput := runFlow(t, input, 256)

	if patientOutput != bufferedOutput {
		t.Errorf("Complex real-world fence output differs between flow modes:\nPatient: %q\nBuffered: %q",
			patientOutput, bufferedOutput)
	}
}

// TestFence11AdjacentDifferentLevels tests adjacent fences with different levels
func TestFence11AdjacentDifferentLevels(t *testing.T) {
	input := `# Adjacent Fences

` + "```" + `
triple
` + "```" + `
` + "````" + `
quadruple
` + "````" + `
` + "`````" + `
quintuple
` + "`````" + `
` + "````" + `
quad again
` + "````" + `
` + "```" + `
triple again
` + "```" + `

Done.`

	// Compare patient vs aggressive streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	aggressiveOutput := runFlow(t, input, -1)

	if patientOutput != aggressiveOutput {
		t.Errorf("Adjacent different levels fence output differs between flow modes:\nPatient: %q\nAggressive: %q",
			patientOutput, aggressiveOutput)
	}
}

// TestFence12StreamingSplit tests fences split across streaming chunks
func TestFence12StreamingSplit(t *testing.T) {
	// Create content that will definitely split with small flow
	input := `# Split Test

` + "```python" + `
line1 = 1  # This is line 1 of the code
line2 = 2  # This is line 2 of the code
line3 = 3  # This is line 3 of the code
line4 = 4  # This is line 4 of the code
line5 = 5  # This is line 5 of the code
line6 = 6  # This is line 6 of the code
line7 = 7  # This is line 7 of the code
line8 = 8  # This is line 8 of the code
line9 = 9  # This is line 9 of the code
line10 = 10  # This is line 10 of the code
line11 = 11  # This is line 11 of the code
line12 = 12  # This is line 12 of the code
line13 = 13  # This is line 13 of the code
line14 = 14  # This is line 14 of the code
line15 = 15  # This is line 15 of the code
line16 = 16  # This is line 16 of the code
line17 = 17  # This is line 17 of the code
line18 = 18  # This is line 18 of the code
line19 = 19  # This is line 19 of the code
line20 = 20  # This is line 20 of the code
` + "```" + `

After the code block.`

	// Test with very small flow to force splitting
	// Compare patient vs very small buffer streaming - output must be identical
	patientOutput := runFlow(t, input, 0)
	verySmallBufferOutput := runFlow(t, input, 32)

	if patientOutput != verySmallBufferOutput {
		t.Errorf("Streaming split fence output differs between flow modes:\nPatient: %q\nVery small buffer: %q",
			patientOutput, verySmallBufferOutput)
	}
}
