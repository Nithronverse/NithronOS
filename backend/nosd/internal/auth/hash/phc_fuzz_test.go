package hash

import "testing"

// Fuzz the PHC parser with garbled inputs.
func FuzzParsePHC(f *testing.F) {
	seeds := []string{
		"",
		"notaphc",
		"$argon2id$v=19$m=65536,t=3,p=2$invalid$base64",
		"$argon2id$v=19$m=1,t=1,p=1$YQ$YQ",
		string(make([]byte, 256)),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		_, _, _, _ = parsePHC(in)
	})
}
