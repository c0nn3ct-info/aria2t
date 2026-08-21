// Package control removes aria2's .aria2 control files once a download has
// finished.
//
// aria2 normally deletes a control file itself on completion, but aria2t runs
// its daemon with --force-save=true so completed downloads survive a restart —
// and force-save deliberately keeps the control file too. The leftovers sit
// next to the user's files forever.
//
// Deleting one is therefore only safe together with dropping that download
// from the saved session: on the next launch aria2 reloads session.txt, and an
// entry whose control file is gone is re-downloaded (auto-file-renaming would
// even write a "file.1" beside the finished one). So Clean records the gid it
// cleaned, and FilterSession — called before aria2c is spawned, when aria2t is
// the only writer of the file — removes those entries.
package control

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Injection seams for tests (the repo's standard pattern over direct OS calls).
var (
	removeFile = os.Remove
	readFile   = os.ReadFile
	writeFile  = os.WriteFile
)

// cleanedPath is the record of gids whose control file we deleted, waiting to
// be dropped from the session on the next daemon start.
func cleanedPath(dataDir string) string { return filepath.Join(dataDir, "cleaned.json") }

// Paths lists the control files a finished download may have left behind.
// aria2 names it "<file>.aria2" for a plain download, but "<dir>/<name>.aria2"
// for a multi-file torrent, so both candidates are returned; a candidate that
// does not exist is simply skipped by Clean. Placeholder paths aria2 uses for
// downloads that never hit the disk ("[METADATA]…", "[MEMORY]…") are ignored.
func Paths(dir, torrentName string, files []string) []string {
	out := make([]string, 0, len(files)+1)
	seen := make(map[string]bool, len(files)+1)
	add := func(p string) {
		c := p + ".aria2"
		if seen[c] {
			return
		}
		seen[c] = true
		out = append(out, c)
	}
	for _, f := range files {
		if f == "" || f[0] == '[' {
			continue
		}
		add(f)
	}
	if torrentName != "" && dir != "" {
		add(filepath.Join(dir, torrentName))
	}
	return out
}

// isInfoHash reports whether s is aria2's infoHash: 40 lower- or upper-case hex
// digits. Checked rather than trusted — it arrives from the browser and is about
// to be turned into a path to unlink.
func isInfoHash(s string) bool {
	if len(s) != 40 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// Saved names the metadata torrent aria2 writes beside a download when
// --bt-save-metadata is on: "<dir>/<infohash>.torrent" (aria2 lower-cases the
// hash). aria2 never lists it among the download's files, so nothing that acts
// on those touches it and it outlives the download as litter in the download
// folder. Returns nothing for a non-torrent, an unusable hash or no dir.
func Saved(dir, infoHash string) []string {
	if dir == "" || !isInfoHash(infoHash) {
		return nil
	}
	return []string{filepath.Join(dir, strings.ToLower(infoHash)+".torrent")}
}

// Leftovers is everything aria2 keeps beside a download that is not part of it:
// the control files above and the saved metadata torrent. This is what has to go
// when a download is finished with, whether it completed or was removed.
func Leftovers(dir, torrentName, infoHash string, files []string) []string {
	return append(Paths(dir, torrentName, files), Saved(dir, infoHash)...)
}

// Clean deletes the given control files and, if any existed, records gid so
// FilterSession can drop the download from the saved session. A control file
// that is already gone is not an error. The first real failure is returned,
// but every candidate is still attempted.
func Clean(dataDir, gid string, controlFiles []string) error {
	var firstErr error
	removed := 0
	for _, p := range controlFiles {
		err := removeFile(p)
		switch {
		case err == nil:
			removed++
		case errors.Is(err, fs.ErrNotExist):
		default:
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if removed == 0 || gid == "" {
		return firstErr
	}
	if err := record(dataDir, gid); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// record appends gid to the cleaned set.
func record(dataDir, gid string) error {
	gids := readCleaned(dataDir)
	for _, g := range gids {
		if g == gid {
			return nil
		}
	}
	raw, _ := json.Marshal(append(gids, gid)) // []string cannot fail
	return writeFile(cleanedPath(dataDir), raw, 0o600)
}

// readCleaned returns the recorded gids; a missing or malformed file is an
// empty set (the worst case is a leftover session entry, not a crash).
func readCleaned(dataDir string) []string {
	raw, err := readFile(cleanedPath(dataDir))
	if err != nil {
		return nil
	}
	var gids []string
	if json.Unmarshal(raw, &gids) != nil {
		return nil
	}
	return gids
}

// FilterSession rewrites sessionPath without the downloads whose control file
// Clean deleted, then forgets them. It must run while aria2c is not running —
// Start calls it just before spawning, when aria2t owns the file.
//
// The session is aria2's --input-file format: a URI line at column 0 followed
// by indented "  key=value" option lines, one of which is the gid.
func FilterSession(sessionPath, dataDir string) error {
	gids := readCleaned(dataDir)
	if len(gids) == 0 {
		return nil
	}
	drop := make(map[string]bool, len(gids))
	for _, g := range gids {
		drop[g] = true
	}

	raw, err := readFile(sessionPath)
	if err != nil {
		// No session yet: nothing to filter, and nothing left to remember.
		if errors.Is(err, fs.ErrNotExist) {
			return forget(dataDir)
		}
		return err
	}

	var kept []string
	var block []string
	blockGid := ""
	flush := func() {
		if len(block) > 0 && !drop[blockGid] {
			kept = append(kept, block...)
		}
		block = nil
		blockGid = ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" {
			continue
		}
		indented := line[0] == ' ' || line[0] == '\t'
		if !indented {
			flush()
		} else if g, ok := strings.CutPrefix(strings.TrimSpace(line), "gid="); ok {
			blockGid = g
		}
		block = append(block, line)
	}
	flush()

	out := ""
	if len(kept) > 0 {
		out = strings.Join(kept, "\n") + "\n"
	}
	if err := writeFile(sessionPath, []byte(out), 0o600); err != nil {
		return err
	}
	return forget(dataDir)
}

// forget drops the cleaned record; the session no longer mentions those gids.
func forget(dataDir string) error {
	err := removeFile(cleanedPath(dataDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
