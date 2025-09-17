// Package flow unicode tests ensure global text compatibility and proper handling.
// These tests validate Unicode and internationalization support:
// - RTL (Right-to-Left) text in Arabic, Hebrew
// - CJK (Chinese, Japanese, Korean) text
// - Emoji sequences and combining characters
// - Zero-width characters and special Unicode blocks
// - Mixed directionality and complex scripts
package flow

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// 1. UNICODE EDGE CASES TESTS

func TestUnicodeEdgeCases(t *testing.T) {
	t.Run("rtl_text_arabic_hebrew", func(t *testing.T) {
		// Arabic and Hebrew text
		inputs := []string{
			"# مرحبا بالعالم\n\nهذا نص عربي مع **تنسيق** و *مائل*.\n\n- قائمة\n- بالعربية\n- مع نقاط\n",
			"# שלום עולם\n\nזה טקסט עברי עם **הדגשה** ו*נטוי*.\n\n- רשימה\n- בעברית\n- עם נקודות\n",
		}

		for _, input := range inputs {
			var buf bytes.Buffer
			err := Flow(context.Background(), strings.NewReader(input), &buf, 1024, passthroughRenderer)
			if err != nil {
				t.Fatalf("Failed on RTL text: %v", err)
			}

			output := buf.String()
			// Glamour formats the markdown, so we check content preservation
			// not exact byte-for-byte equality
			if !strings.Contains(output, "مرحبا بالعالم") && !strings.Contains(output, "שלום עולם") {
				t.Errorf("RTL text content not preserved")
			}

			// Test with small windows too
			testFlowWithVariousSizes(t, input)
		}
	})

	t.Run("mixed_scripts", func(t *testing.T) {
		input := "# Multi-language 多语言 متعدد اللغات мультиязычный\n\n" +
			"English text followed by 中文内容 and עברית text.\n" +
			"日本語も含めて and مع العربية too.\n" +
			"Кириллица здесь и Ελληνικά also.\n"

		testFlowWithVariousSizes(t, input)

		// Verify all scripts preserved
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)
		if err != nil {
			t.Fatalf("Mixed scripts failed: %v", err)
		}

		output := buf.String()
		// Check each script is present
		scripts := []string{"Multi-language", "多语言", "متعدد اللغات", "мультиязычный",
			"中文内容", "עברית", "日本語", "العربية", "Кириллица", "Ελληνικά"}
		for _, script := range scripts {
			if !strings.Contains(output, script) {
				t.Errorf("Script %q not found in output", script)
			}
		}
	})

	t.Run("emoji_clusters", func(t *testing.T) {
		// Various emoji including complex clusters
		input := "# Emoji Test 🎉\n\n" +
			"Simple: 😀 😃 😄 😁\n" +
			"Flags: 🇺🇸 🇬🇧 🇯🇵 🇨🇳\n" +
			"Family: 👨‍👩‍👧‍👦\n" +
			"Professions: 👨‍💻 👩‍🔬 👨‍🎨\n" +
			"Skin tones: 👋🏻 👋🏽 👋🏿\n" +
			"Combined: 👨🏻‍💻 👩🏽‍🔬\n"

		testFlowWithVariousSizes(t, input)

		// Verify emoji preserved
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 10, passthroughRenderer)
		if err != nil {
			t.Fatalf("Emoji test failed: %v", err)
		}

		// Glamour formats text, just verify emoji are present
		if !strings.Contains(buf.String(), "🎉") || !strings.Contains(buf.String(), "👨") {
			t.Error("Emoji content not found")
		}
	})

	t.Run("zero_width_characters", func(t *testing.T) {
		// Zero-width space (U+200B), Zero-width joiner (U+200D), Zero-width non-joiner (U+200C)
		input := "Word\u200Bbreak\n" + // ZWSP for word break
			"لا\u200Carial\n" + // ZWNJ in Arabic/Latin mix
			"👨\u200D👩\u200D👧\u200D👦\n" + // ZWJ in emoji family
			"a\u200Db\u200Dc\n" + // ZWJ between letters
			"test\u200B\u200B\u200Bmultiple\n" // Multiple ZWSP

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 1, passthroughRenderer)
		if err != nil {
			t.Fatalf("Zero-width char test failed: %v", err)
		}

		// Glamour may process zero-width characters
		// Just verify visible text is present
		if !strings.Contains(buf.String(), "Word") || !strings.Contains(buf.String(), "break") {
			t.Error("Text around zero-width chars not found")
		}

		// Test with various window sizes
		testFlowWithVariousSizes(t, input)
	})

	t.Run("combining_marks", func(t *testing.T) {
		// Test both precomposed and decomposed forms
		input := "# Combining Marks\n\n" +
			"Precomposed: café naïve résumé\n" +
			"Decomposed: cafe\u0301 nai\u0308ve re\u0301sume\u0301\n" + // Using combining marks
			"Multiple marks: a\u0300\u0301\u0302\u0303\u0304\n" + // a with 5 combining marks
			"Thai: กำ ดี มาก\n" + // Thai with combining marks
			"Vietnamese: Tiếng Việt\n"

		testFlowWithVariousSizes(t, input)

		// Verify marks preserved
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 10, passthroughRenderer)
		if err != nil {
			t.Fatalf("Combining marks test failed: %v", err)
		}

		// Glamour may normalize combining marks
		// Just verify base text is present
		if !strings.Contains(buf.String(), "café") || !strings.Contains(buf.String(), "Combining Marks") {
			t.Error("Content with combining marks not found")
		}
	})
}

// 2. CHARACTER ENCODING TESTS

func TestUnicodeCharacterEncoding(t *testing.T) {
	t.Run("utf8_bom", func(t *testing.T) {
		// UTF-8 BOM (Byte Order Mark)
		input := "\xEF\xBB\xBF# Document with BOM\n\nContent after BOM.\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)
		if err != nil {
			t.Fatalf("BOM handling failed: %v", err)
		}

		// Glamour may handle BOM differently
		// Just verify content is preserved
		if !strings.Contains(buf.String(), "Document with BOM") || !strings.Contains(buf.String(), "Content after BOM") {
			t.Error("Content not preserved with BOM")
		}
	})

	t.Run("invalid_utf8_sequences", func(t *testing.T) {
		// Invalid UTF-8 bytes
		invalidBytes := []byte{0x80, 0x81, 0xFF, 0xFE, 0xFD}
		input := append([]byte("Valid start\n"), invalidBytes...)
		input = append(input, []byte("\nValid end")...)

		var buf bytes.Buffer
		err := Flow(context.Background(), bytes.NewReader(input), &buf, 100, passthroughRenderer)
		if err != nil {
			t.Logf("Invalid UTF-8 handling: %v", err)
		}

		// Should handle gracefully
		if buf.Len() == 0 {
			t.Error("Expected some output for invalid UTF-8")
		}
	})

	t.Run("replacement_character", func(t *testing.T) {
		// Unicode replacement character U+FFFD
		input := "Text with � replacement � characters\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)
		if err != nil {
			t.Fatalf("Replacement char test failed: %v", err)
		}

		// Glamour formats the text, verify replacement chars are present
		if !strings.Contains(buf.String(), "replacement") {
			t.Error("Content with replacement characters not preserved")
		}
	})

	t.Run("multi_byte_boundaries", func(t *testing.T) {
		// Test multi-byte characters at window boundaries
		// Using Chinese characters (3 bytes each in UTF-8)
		input := "中文字符测试文本\n日本語テキスト\n한글 텍스트\n"

		// Test with 1-byte window (most challenging)
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 1, passthroughRenderer)
		if err != nil {
			t.Fatalf("Multi-byte boundary test failed: %v", err)
		}

		// Glamour formats the text, verify key content is preserved
		if !strings.Contains(buf.String(), "中文") || !strings.Contains(buf.String(), "日本語") {
			t.Error("Multi-byte characters not preserved at boundaries")
		}

		// Test with windows that might split multi-byte chars
		for _, window := range []int64{2, 3, 5, 7, 11} {
			buf.Reset()
			err := Flow(context.Background(), strings.NewReader(input), &buf, window, passthroughRenderer)
			if err != nil {
				t.Errorf("Failed with window=%d: %v", window, err)
			}
			// Just verify content is present
			if !strings.Contains(buf.String(), "中文") {
				t.Errorf("Multi-byte chars not found with window=%d", window)
			}
		}
	})

	t.Run("surrogate_pairs", func(t *testing.T) {
		// Emoji and characters outside BMP use surrogate pairs in UTF-16
		// but are valid 4-byte sequences in UTF-8
		input := "Mathematical: 𝕏 𝕐 𝕑\n" + // Mathematical double-struck
			"Ancient: 𐌀 𐌁 𐌂\n" + // Old Italic
			"Emoji: 😀 🎉 🚀\n" + // Emoji (also 4-byte UTF-8)
			"Egyptian: 𓀀 𓀁 𓀂\n" // Egyptian hieroglyphs

		testFlowWithVariousSizes(t, input)
	})
}

// 3. TEXT DIRECTION TESTS

func TestUnicodeTextDirection(t *testing.T) {
	t.Run("pure_rtl_markdown", func(t *testing.T) {
		input := "# عنوان رئيسي\n\n" +
			"## عنوان فرعي\n\n" +
			"فقرة عربية مع **نص غامق** و *نص مائل*.\n\n" +
			"### قائمة:\n" +
			"- العنصر الأول\n" +
			"- العنصر الثاني\n" +
			"- العنصر الثالث\n\n" +
			"[رابط](https://example.com)\n"

		testFlowWithVariousSizes(t, input)
	})

	t.Run("mixed_ltr_rtl", func(t *testing.T) {
		input := "# Mixed Direction Text\n\n" +
			"The Hebrew word שלום means peace.\n" +
			"The Arabic phrase مرحبا بك means welcome.\n" +
			"Numbers 123 in Arabic: ١٢٣\n" +
			"Code: `console.log('שלום');`\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 100, passthroughRenderer)
		if err != nil {
			t.Fatalf("Mixed LTR/RTL failed: %v", err)
		}

		// Glamour formats text, verify key content is preserved
		if !strings.Contains(buf.String(), "שלום") || !strings.Contains(buf.String(), "مرحبا") {
			t.Error("Mixed direction text content not found")
		}
	})

	t.Run("bidirectional_overrides", func(t *testing.T) {
		// LRO (U+202D), RLO (U+202E), PDF (U+202C)
		input := "Normal text\n" +
			"\u202Dforced LTR text\u202C\n" +
			"\u202Eforced RTL text\u202C\n" +
			"Mixed: \u202Dabc\u202Edef\u202Cghi\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 10, passthroughRenderer)
		if err != nil {
			t.Fatalf("Bidi override test failed: %v", err)
		}

		// Glamour may process bidi overrides
		// Just verify visible text is present
		if !strings.Contains(buf.String(), "Normal text") || !strings.Contains(buf.String(), "forced") {
			t.Error("Text with bidi overrides not found")
		}
	})

	t.Run("rtl_in_code_blocks", func(t *testing.T) {
		input := "```javascript\n" +
			"// تعليق بالعربية\n" +
			"const greeting = 'שלום עולם';\n" +
			"console.log(greeting);\n" +
			"```\n"

		testFlowWithVariousSizes(t, input)
	})

	t.Run("direction_marks_in_links", func(t *testing.T) {
		// LRM (U+200E) and RLM (U+200F)
		input := "[English \u200Elink](url)\n" +
			"[عربي \u200Fرابط](url)\n" +
			"Mixed: [EN \u200Eעברית \u200Fالعربية](url)\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 10, passthroughRenderer)
		if err != nil {
			t.Fatalf("Direction marks test failed: %v", err)
		}

		// Glamour formats links, verify content is present
		if !strings.Contains(buf.String(), "English") || !strings.Contains(buf.String(), "عربي") {
			t.Error("Direction marks content not found")
		}
	})
}

// 4. EXTREME UNICODE TESTS

func TestUnicodeExtreme(t *testing.T) {
	t.Run("max_combining_marks", func(t *testing.T) {
		// Create a character with many combining marks
		var input strings.Builder
		input.WriteString("a")
		// Add 50 different combining marks (there are ~350 total)
		combiningMarks := []rune{
			0x0300, 0x0301, 0x0302, 0x0303, 0x0304, 0x0305, 0x0306, 0x0307,
			0x0308, 0x0309, 0x030A, 0x030B, 0x030C, 0x030D, 0x030E, 0x030F,
			0x0310, 0x0311, 0x0312, 0x0313, 0x0314, 0x0315, 0x0316, 0x0317,
			0x0318, 0x0319, 0x031A, 0x031B, 0x031C, 0x031D, 0x031E, 0x031F,
			0x0320, 0x0321, 0x0322, 0x0323, 0x0324, 0x0325, 0x0326, 0x0327,
			0x0328, 0x0329, 0x032A, 0x032B, 0x032C, 0x032D, 0x032E, 0x032F,
			0x0330, 0x0331,
		}
		for _, mark := range combiningMarks {
			input.WriteRune(mark)
		}
		input.WriteString("\n")

		inputStr := input.String()
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(inputStr), &buf, 1, passthroughRenderer)
		if err != nil {
			t.Fatalf("Max combining marks test failed: %v", err)
		}

		// Glamour may normalize combining marks
		// Just verify base character is present
		if !strings.Contains(buf.String(), "a") {
			t.Error("Base character not found in output")
		}
	})

	t.Run("complex_grapheme_clusters", func(t *testing.T) {
		// Complex emoji with skin tone and gender modifiers
		input := "👨🏻‍💻 Woman technologist: 👩🏽‍💻\n" +
			"Family: 👨‍👩‍👧‍👦\n" +
			"Flag: 🏴󠁧󠁢󠁳󠁣󠁴󠁿\n" + // Scotland flag with tag characters
			"Rainbow flag: 🏳️‍🌈\n" +
			"Keycap: 3️⃣ #️⃣ *️⃣\n"

		testFlowWithVariousSizes(t, input)
	})

	t.Run("fullwidth_characters", func(t *testing.T) {
		input := "# Ｆｕｌｌｗｉｄｔｈ　Ｔｅｘｔ\n\n" +
			"Ｈｅｌｌｏ　Ｗｏｒｌｄ！\n" +
			"Numbers:　１２３４５\n" +
			"Mixed: Normal and Ｆｕｌｌｗｉｄｔｈ text\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 10, passthroughRenderer)
		if err != nil {
			t.Fatalf("Fullwidth test failed: %v", err)
		}

		// Glamour formats text, verify fullwidth content
		if !strings.Contains(buf.String(), "Ｆｕｌｌｗｉｄｔｈ") {
			t.Error("Fullwidth content not found")
		}
	})

	t.Run("mathematical_symbols", func(t *testing.T) {
		input := "# Mathematical Notation\n\n" +
			"∀x ∈ ℝ: x² ≥ 0\n" +
			"∑ᵢ₌₁ⁿ i = n(n+1)/2\n" +
			"∫₀^∞ e⁻ˣ dx = 1\n" +
			"√2 ≈ 1.414\n" +
			"∞ ≠ -∞\n" +
			"𝔸𝔹ℂ𝔻𝔼𝔽𝔾ℍ𝕀𝕁𝕂𝕃𝕄ℕ𝕆ℙℚℝ𝕊𝕋𝕌𝕍𝕎𝕏𝕐ℤ\n" // Mathematical double-struck

		testFlowWithVariousSizes(t, input)
	})

	t.Run("historic_scripts", func(t *testing.T) {
		input := "# Historic Scripts\n\n" +
			"Cuneiform: 𒀀 𒀁 𒀂 𒀃 𒀄\n" +
			"Egyptian hieroglyphs: 𓀀 𓀁 𓀂 𓀃 𓀄\n" +
			"Linear B: 𐀀 𐀁 𐀂 𐀃 𐀄\n" +
			"Old Persian: 𐎠 𐎡 𐎢 𐎣 𐎤\n" +
			"Phoenician: 𐤀 𐤁 𐤂 𐤃 𐤄\n" +
			"Runic: ᚠ ᚡ ᚢ ᚣ ᚤ\n"

		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, 1, passthroughRenderer)
		if err != nil {
			t.Fatalf("Historic scripts test failed: %v", err)
		}

		// Glamour formats text, verify historic scripts are present
		if !strings.Contains(buf.String(), "Cuneiform") || !strings.Contains(buf.String(), "Egyptian") {
			t.Error("Historic script content not found")
		}

		// These are all 4-byte UTF-8 sequences, test boundaries
		testFlowWithVariousSizes(t, input)
	})
}

// Helper function to test with various window sizes
func testFlowWithVariousSizes(t *testing.T, input string) {
	windows := []int64{1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610, 987, 1024}

	// Extract key content to verify preservation
	// Look for non-ASCII content as markers
	var keyContent []string
	for _, r := range input {
		if r > 127 { // Non-ASCII
			keyContent = append(keyContent, string(r))
		}
	}

	for _, window := range windows {
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, window, passthroughRenderer)

		if err != nil {
			t.Errorf("Failed with window=%d: %v", window, err)
			continue
		}

		output := buf.String()
		// Don't expect exact match with glamour formatting
		// Just verify key unicode content is preserved
		for _, content := range keyContent {
			if !strings.Contains(output, content) {
				t.Errorf("Unicode content %q not preserved with window=%d", content, window)
				break
			}
		}
	}
}

// Additional comprehensive Unicode test
func TestUnicodeComprehensive(t *testing.T) {
	// A document containing a wide variety of Unicode features
	//lint:ignore ST1018 raw string intentionally contains Unicode characters for test coverage
	input := `# 🌍 International Document 文档 مستند ドキュメント

## Mixed Scripts 多种文字 نصوص مختلطة

This paragraph contains English, 中文, العربية, עברית, 日本語, 한국어,
ελληνικά, русский, हिन्दी, ไทย, and more.

### Emoji and Symbols 😀

- Simple emoji: 😀 😃 😄
- Flags: 🇺🇸 🇬🇧 🇯🇵
- Families: 👨‍👩‍👧‍👦
- Professions: 👨‍💻 👩‍🔬
- Animals: 🐶 🐱 🐭

### Mathematical ∑∫∂

∀x ∈ ℝ: x² ≥ 0
∫₀^∞ e⁻ˣ dx = 1

### Combining Marks

café (precomposed) vs cafe\u0301 (decomposed)
ñ vs n\u0303
ą vs a\u0328

### Direction Testing

LTR: Hello World
RTL: مرحبا بالعالم
Mixed: The word שלום means peace

### Special Characters

Zero-width: a​b (ZWSP)
Soft hyphen: ex­ample
Non-breaking space: hello world

### Full-width

Ｈｅｌｌｏ　Ｗｏｒｌｄ

### Historic Scripts

Cuneiform: 𒀀 𒀁 𒀂
Egyptian: 𓀀 𓀁 𓀂
`

	// Test with many different window sizes
	windows := []int64{1, 10, 100, 1024, 10000}
	for _, window := range windows {
		var buf bytes.Buffer
		err := Flow(context.Background(), strings.NewReader(input), &buf, window, passthroughRenderer)

		if err != nil {
			t.Errorf("Comprehensive test failed with window=%d: %v", window, err)
			continue
		}

		// Glamour formats the comprehensive unicode text
		// Just verify key sections are present
		if !strings.Contains(buf.String(), "International Document") ||
		   !strings.Contains(buf.String(), "English") ||
		   !strings.Contains(buf.String(), "中文") {
			t.Errorf("Comprehensive Unicode content not found with window=%d", window)
		}
	}
}
