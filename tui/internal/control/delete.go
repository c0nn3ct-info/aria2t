package control

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// lstatFile is a seam (see removeFile). Lstat, not Stat: a symlink must be seen
// as a symlink, or a link inside the download dir could be followed to unlink
// something outside it.
var lstatFile = os.Lstat

// within reports whether p is inside root. Both must already be cleaned.
// filepath.Rel is what does the work: a path that escapes produces a result
// starting with "..", and one on another volume produces an error.
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	// Rel never returns an absolute path for cleaned inputs, but a relative root
	// would make one meaningless, so this is checked rather than assumed.
	return !filepath.IsAbs(rel)
}

// Targets vets the files of one download before anything is unlinked, and is
// the only thing standing between a bad path and the user's disk.
//
// The paths come from the daemon and reach us over the extension's stdio pipe,
// so they are not attacker-controlled in any ordinary sense — but this deletes
// data and cannot be undone, so it refuses anything it cannot prove is a file of
// this download: dir must sit inside root (the configured download directory),
// and every file inside dir. A path that escapes is an error rather than a skip,
// so a surprise is reported instead of half-deleting the rest.
//
// aria2's placeholders for downloads that never hit the disk ("[METADATA]…",
// "[MEMORY]…") are skipped — those are expected, not surprises.
func Targets(root, dir string, files []string) ([]string, error) {
	if root == "" || !filepath.IsAbs(root) {
		return nil, fmt.Errorf("download root %q is not an absolute path", root)
	}
	root = filepath.Clean(root)
	if dir == "" || !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("download dir %q is not an absolute path", dir)
	}
	dir = filepath.Clean(dir)
	if dir != root && !within(root, dir) {
		return nil, fmt.Errorf("download dir %q is outside %q", dir, root)
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		if f == "" || f[0] == '[' {
			continue
		}
		p := filepath.Clean(f)
		if !filepath.IsAbs(p) || p == dir || !within(dir, p) {
			return nil, fmt.Errorf("file %q is outside %q", f, dir)
		}
		out = append(out, p)
	}
	return out, nil
}

// Delete unlinks a finished download's files and tidies up after them, then
// records gid so FilterSession drops the download from the saved session — the
// same rule Clean documents, and for the same reason: an entry left in
// session.txt whose files are gone is simply downloaded again.
//
// targets must come from Targets. Only regular files are removed, one at a time
// and never recursively; a file that is already gone is not an error, and every
// target is attempted even after a failure, of which the first is returned.
func Delete(dataDir, gid, root string, targets []string) error {
	var firstErr error
	keep := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}
	removed := 0
	emptied := make([]string, 0, len(targets))
	for _, p := range targets {
		fi, err := lstatFile(p)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				keep(err)
			}
			continue
		}
		if !fi.Mode().IsRegular() {
			keep(fmt.Errorf("refusing to delete %q: not a regular file", p))
			continue
		}
		if err := removeFile(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				keep(err)
			}
			continue
		}
		removed++
		emptied = append(emptied, filepath.Dir(p))
	}
	prune(filepath.Clean(root), emptied)
	if removed == 0 || gid == "" {
		return firstErr
	}
	if err := record(dataDir, gid); err != nil {
		keep(err)
	}
	return firstErr
}

// prune removes the directories a torrent's files left behind, walking up from
// each until it reaches the download root. os.Remove refuses a non-empty
// directory, which is exactly the wanted rule — so a failure stops the walk
// instead of being reported: a directory that still holds something is not a
// problem, it just is not ours to remove.
func prune(root string, dirs []string) {
	for _, d := range dirs {
		for d != root && within(root, d) {
			if err := removeFile(d); err != nil {
				break
			}
			d = filepath.Dir(d)
		}
	}
}
