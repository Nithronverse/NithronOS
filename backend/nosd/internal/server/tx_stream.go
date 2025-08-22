package server

import (
	"bufio"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

func handleTxStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// start by sending current log from cursor 0
	f, err := os.Open(txLogPath(id))
	if err == nil {
		scan := bufio.NewScanner(f)
		for scan.Scan() {
			ln := scan.Text()
			_, _ = w.Write([]byte("event: log\n"))
			_, _ = w.Write([]byte("data: " + ln + "\n\n"))
		}
		_ = f.Close()
		flusher.Flush()
	}
	// keepalive and tail new lines for up to ~5m
	deadline := time.Now().Add(5 * time.Minute)
	lastSize := int64(0)
	_ = func() error {
		if st, err := os.Stat(txLogPath(id)); err == nil {
			lastSize = st.Size()
		}
		return nil
	}()
	for time.Now().Before(deadline) {
		// keepalive comment
		_, _ = w.Write([]byte(": keepalive\n\n"))
		flusher.Flush()
		time.Sleep(1 * time.Second)
		// check for appended data
		st, err := os.Stat(txLogPath(id))
		if err == nil && st.Size() > lastSize {
			f, err := os.Open(txLogPath(id))
			if err == nil {
				if _, err := f.Seek(lastSize, 0); err == nil {
					scan := bufio.NewScanner(f)
					for scan.Scan() {
						ln := scan.Text()
						_, _ = w.Write([]byte("event: log\n"))
						_, _ = w.Write([]byte("data: " + ln + "\n\n"))
					}
				}
				lastSize = st.Size()
				_ = f.Close()
				flusher.Flush()
			}
		}
	}
}
