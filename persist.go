package pelican

import (
	"os"
)

// CleanupOldKnownHosts removes fn + defaultFileFormat().
func CleanupOldKnownHosts(fn string) {
	os.Remove(fn + defaultFileFormat())
}
