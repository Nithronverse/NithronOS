//go:build windows

package server

func mustBeRoot() error { return nil }

func runtimeChownSupported() bool                 { return false }
func chownByName(path, owner, group string) error { return nil }
