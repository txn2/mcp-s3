package extensions

import "testing"

func FuzzParseSize(f *testing.F) {
	// Seed corpus with valid inputs
	f.Add("1024")
	f.Add("10KB")
	f.Add("10kb")
	f.Add("10K")
	f.Add("10MB")
	f.Add("10M")
	f.Add("1GB")
	f.Add("1G")
	f.Add("1TB")
	f.Add("1T")
	f.Add("")
	f.Add("invalid")
	f.Add("0")
	f.Add("-1")
	f.Add("-100MB")
	f.Add("999999999999999999999")

	f.Fuzz(func(t *testing.T, input string) {
		defaultVal := int64(42)
		result := parseSize(input, defaultVal)

		// parseSize should never panic and should return non-negative values
		if result < 0 {
			t.Errorf("parseSize(%q) returned negative value: %d", input, result)
		}
	})
}
