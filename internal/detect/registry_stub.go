//go:build !windows

package detect

func nodeVersionFromRegistry(string) string { return "" }
