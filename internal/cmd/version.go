package cmd

var (
	commit  = "<none>"
	version = "snapshot"
)

func init() {
	if 8 < len(commit) {
		commit = commit[:8]
	}
}
