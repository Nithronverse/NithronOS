//go:build !windows

package server

import (
	"log/syslog"
)

func logAuthPriv(message string) {
	w, err := syslog.New(syslog.LOG_AUTHPRIV|syslog.LOG_INFO, "nos-agent")
	if err != nil {
		return
	}
	_ = w.Info(message)
	_ = w.Close()
}
