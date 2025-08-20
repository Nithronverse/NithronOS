package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nithronos/backend/nosd/internal/shares"
)

func writeSmbShare(etcDir string, sh shares.Share) error {
	dir := filepath.Join(etcDir, "samba", "smb.conf.d")
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, fmt.Sprintf("nos-%s.conf", sh.Name))
	ro := "no"
	if sh.RO {
		ro = "yes"
	}
	content := fmt.Sprintf("[%s]\npath = %s\nread only = %s\nvalid users = %s\n", sh.Name, sh.Path, ro, usersCSV(sh.Users))
	return os.WriteFile(path, []byte(content), 0o644)
}

func usersCSV(u []string) string {
	if len(u) == 0 {
		return ""
	}
	out := u[0]
	for i := 1; i < len(u); i++ {
		out += "," + u[i]
	}
	return out
}

func appendNfsExport(etcDir string, sh shares.Share) error {
	dir := filepath.Join(etcDir, "exports.d")
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "nos.exports")
	line := fmt.Sprintf("%s *(ro,sync,no_subtree_check)\n", sh.Path)
	if !sh.RO {
		line = fmt.Sprintf("%s *(rw,sync,no_subtree_check)\n", sh.Path)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func removeSmbShare(etcDir, name string) error {
	path := filepath.Join(etcDir, "samba", "smb.conf.d", fmt.Sprintf("nos-%s.conf", name))
	_ = os.Remove(path)
	return nil
}

func removeNfsExport(etcDir, pathToRemove string) error {
	path := filepath.Join(etcDir, "exports.d", "nos.exports")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if !strings.HasPrefix(ln, pathToRemove+" ") {
			out = append(out, ln)
		}
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}
