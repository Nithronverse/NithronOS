package validate

import (
	"path/filepath"
	"strings"
	"testing"
)

func fromSlash(p string) string { return filepath.FromSlash(p) }

func TestShareName_Valid(t *testing.T) {
	valid := []string{
		"media",
		"A_1-2",
		strings.Repeat("a", 32),
		"Z_y-9",
	}
	for _, v := range valid {
		if err := ShareName(v); err != nil {
			t.Fatalf("expected valid name %q, got error: %v", v, err)
		}
	}
}

func TestShareName_Invalid(t *testing.T) {
	invalid := []string{
		"",
		strings.Repeat("a", 33),
		"bad name",
		"bad$",
		".dot",
		"slash/inside",
	}
	for _, v := range invalid {
		if err := ShareName(v); err == nil {
			t.Fatalf("expected error for invalid name %q", v)
		}
	}
}

func TestPathUnder_Valid(t *testing.T) {
	roots := []string{fromSlash("/srv/pool"), fromSlash("/mnt/data")}
	cases := []string{
		"/srv/pool",
		"/srv/pool/photos",
		"/srv/pool/photos/2024",
		"/mnt/data",
		"/mnt/data/sub",
		"/srv/pool/../pool/ok", // cleans to /srv/pool/ok
	}
	for _, c := range cases {
		if err := PathUnder(roots, fromSlash(c)); err != nil {
			t.Fatalf("expected valid path %q under roots, got %v", c, err)
		}
	}
}

func TestPathUnder_Invalid(t *testing.T) {
	roots := []string{fromSlash("/srv/pool"), fromSlash("/mnt/data")}
	cases := []string{
		"",
		"/srv",         // parent, not allowed
		"/srv/pool2",   // edge: different root
		"/srv/poolish", // prefix but not a subdir
		"/etc",
		"/mnt/dat", // typo
	}
	for _, c := range cases {
		if err := PathUnder(roots, fromSlash(c)); err == nil {
			t.Fatalf("expected error for invalid path %q", c)
		}
	}
}
