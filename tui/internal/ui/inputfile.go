package ui

import "strings"

// inputEntry is one download from an aria2 input file: its URIs (mirrors of a
// single file) and the per-download options that followed it.
type inputEntry struct {
	uris []string
	opts map[string]string
}

// parseInputFile parses aria2's --input-file format: each non-indented line
// starts a download (tab-separated URIs are mirrors of one file); indented
// "key=value" lines set options for the download above them; blank lines and
// lines beginning with '#' are ignored.
func parseInputFile(content string) []inputEntry {
	var out []inputEntry
	ci := -1
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			// Option line for the current download.
			if ci >= 0 {
				if k, v, ok := strings.Cut(trimmed, "="); ok {
					out[ci].opts[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}
			continue
		}
		out = append(out, inputEntry{uris: splitURIs(trimmed), opts: map[string]string{}})
		ci = len(out) - 1
	}
	return out
}

// splitURIs splits a download line into its tab-separated mirror URIs.
func splitURIs(line string) []string {
	var out []string
	for _, u := range strings.Split(line, "\t") {
		if u = strings.TrimSpace(u); u != "" {
			out = append(out, u)
		}
	}
	return out
}

// mergeOpts returns base overlaid with extra (extra wins).
func mergeOpts(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
