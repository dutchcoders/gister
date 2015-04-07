package main

import (
	"fmt"
	"regexp"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func DebugPrintln(format string, args ...interface{}) {
	if !*debug {
		return
	}

	fmt.Printf(fmt.Sprintf("%s\n", format), args...)
}

// "user/NNNNNNN" -> "NNNNNNN"
func clearGistId(id string) string {
	return regexp.MustCompile(`^[^/]+/`).ReplaceAllLiteralString(id, "")
}
