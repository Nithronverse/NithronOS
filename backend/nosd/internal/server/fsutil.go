package server

import "strings"

func uniqueStringsSafe(in []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func isUnderAllowed(path string, roots []string) bool {
	for _, r := range roots {
		if strings.HasPrefix(path+"/", strings.TrimRight(r, "/")+"/") {
			return true
		}
	}
	return false
}
