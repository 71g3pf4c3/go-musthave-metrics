package version

import "fmt"

var (
	BuildVersion string
	BuildDate    string
	BuildCommit  string
)

func valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

// Print outputs build info to stdout.
func Print() {
	fmt.Printf("Build version: %s\n", valueOrNA(BuildVersion))
	fmt.Printf("Build date: %s\n", valueOrNA(BuildDate))
	fmt.Printf("Build commit: %s\n", valueOrNA(BuildCommit))
}
