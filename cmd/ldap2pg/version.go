package main

var commit = "<none>"

func init() {
	if 8 < len(commit) {
		commit = commit[:8]
	}
}
