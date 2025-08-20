package validate

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reShare    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	ErrBadName = errors.New("invalid share name")
	ErrBadPath = errors.New("path must be under an allowed root")
)

func ShareName(s string) error {
	if !reShare.MatchString(s) {
		return ErrBadName
	}
	return nil
}

func PathUnder(roots []string, p string) error {
	if p == "" {
		return ErrBadPath
	}
	ap := filepath.Clean(p)
	for _, r := range roots {
		rr := filepath.Clean(r)
		if ap == rr || strings.HasPrefix(ap, rr+string(filepath.Separator)) {
			return nil
		}
	}
	return ErrBadPath
}
