package homedir

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func Expand(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	// Current home
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, _ := os.UserHomeDir()
		if home == "" {
			return path
		}
		return strings.Replace(path, "~", home, 1)
	}

	// Other home
	username, relpath, _ := strings.Cut(path, "/")
	if username == "" {
		return path
	}
	username = strings.TrimPrefix(username, "~")
	u, err := user.Lookup(username)
	if err != nil {
		return path
	}
	return filepath.Join(u.HomeDir, relpath)
}
