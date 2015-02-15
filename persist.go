package pelican

import (
	"os"
)

func CleanupOldKnownHosts(fn string) {
	os.Remove(fn + defaultFileFormat())
}
