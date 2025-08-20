package server

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"nithronos/backend/nosd/internal/config"
)

var redactionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password\s*[:=]\s*)([^\s"']+)`),
	regexp.MustCompile(`(?i)(secret\s*[:=]\s*)([^\s"']+)`),
	regexp.MustCompile(`(?i)(token\s*[:=]\s*)([^\s"']+)`),
	regexp.MustCompile(`(?i)(key\s*=\s*)([^\s"']+)`),
}

func redactLine(line string) string {
	s := line
	for _, re := range redactionPatterns {
		s = re.ReplaceAllString(s, `$1REDACTED`)
	}
	return s
}

func writeTarFile(tw *tar.Writer, name string, r io.Reader) error {
	// Read all into buffer? For streams, we can read to memory to set size; but safer to stream without size by buffering.
	// We'll read into a temporary buffer since sizes are expected to be modest for text outputs.
	buf := bufio.NewScanner(r)
	var b strings.Builder
	for buf.Scan() {
		b.WriteString(redactLine(buf.Text()))
		b.WriteByte('\n')
	}
	data := []byte(b.String())
	hdr := &tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), ModTime: time.Now()}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func writeTarFileIfExists(tw *tar.Writer, hostPath, name string) {
	f, err := os.Open(hostPath)
	if err != nil {
		return
	}
	defer f.Close()
	_ = writeTarFile(tw, name, f)
}

func writeCmdOutput(tw *tar.Writer, name string, cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	out, err := c.CombinedOutput()
	if err != nil {
		// include stderr/output anyway for diagnostics
	}
	_ = writeTarFile(tw, name, strings.NewReader(string(out)))
}

func handleSupportBundle(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", "attachment; filename=nos-support-bundle.tar.gz")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		tw := tar.NewWriter(gz)
		defer tw.Close()

		// Journals and logs (last 2 days)
		writeCmdOutput(tw, "logs/journal_nosd.txt", "journalctl", "-u", "nosd", "--since", "2 days ago")
		writeTarFileIfExists(tw, "/var/log/caddy/access.log", "logs/caddy_access.log")
		writeTarFileIfExists(tw, "/var/log/caddy/error.log", "logs/caddy_error.log")

		// System info
		writeCmdOutput(tw, "system/uname.txt", "uname", "-a")
		writeTarFileIfExists(tw, "/etc/os-release", "system/os-release")

		// Firewall rules
		writeCmdOutput(tw, "network/nft_ruleset.txt", "nft", "list", "ruleset")

		// Storage
		writeCmdOutput(tw, "storage/lsblk.json", "lsblk", "-J", "-O")
		writeCmdOutput(tw, "storage/blkid.txt", "blkid")
		writeCmdOutput(tw, "storage/btrfs_show.txt", "btrfs", "filesystem", "show")
		// usage for common mount roots if present
		for _, m := range []string{"/", "/srv", "/mnt", "/pool", "/data"} {
			if fi, err := os.Stat(m); err == nil && fi.IsDir() {
				name := strings.TrimPrefix(m, "/")
				writeCmdOutput(tw, "storage/usage_"+strings.ReplaceAll(name, "/", "_")+".txt", "btrfs", "fi", "usage", m)
			}
		}

		// Config files with redaction
		nosConfDir := filepath.Join(cfg.EtcDir, "nos")
		if matches, _ := filepath.Glob(filepath.Join(nosConfDir, "*.conf")); len(matches) > 0 {
			for _, p := range matches {
				name := filepath.Base(p)
				writeTarFileIfExists(tw, p, filepath.Join("configs/nos", name))
			}
		}
		if matches, _ := filepath.Glob("/etc/samba/smb.conf.d/*.conf"); len(matches) > 0 {
			for _, p := range matches {
				name := filepath.Base(p)
				writeTarFileIfExists(tw, p, filepath.Join("configs/samba", name))
			}
		}
	}
}
