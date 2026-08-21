package control

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// fakeInfo is the little of fs.FileInfo that Delete looks at.
type fakeInfo struct{ mode fs.FileMode }

func (f fakeInfo) Name() string       { return "x" }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeInfo) Sys() any           { return nil }

// stubLstat swaps lstatFile with a table of path → mode. A path that is absent
// reads as gone, which is how a download whose files a user already deleted by
// hand behaves.
func stubLstat(t *testing.T, modes map[string]fs.FileMode) {
	t.Helper()
	orig := lstatFile
	lstatFile = func(p string) (os.FileInfo, error) {
		m, ok := modes[p]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return fakeInfo{mode: m}, nil
	}
	t.Cleanup(func() { lstatFile = orig })
}

func regular(paths ...string) map[string]fs.FileMode {
	m := make(map[string]fs.FileMode, len(paths))
	for _, p := range paths {
		m[p] = 0
	}
	return m
}

func TestTargetsAcceptsFilesUnderTheDir(t *testing.T) {
	got, err := Targets("/dl", "/dl/show", []string{"/dl/show/a.mkv", "/dl/show/subs/a.srt"})
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	want := []string{"/dl/show/a.mkv", "/dl/show/subs/a.srt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
}

func TestTargetsSkipsAria2Placeholders(t *testing.T) {
	got, err := Targets("/dl", "/dl", []string{"[METADATA]abc", "", "/dl/a.iso", "[MEMORY]x"})
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"/dl/a.iso"}) {
		t.Fatalf("targets = %v", got)
	}
}

// The whole point of the package: nothing outside the configured root may be
// named, however it is spelled.
func TestTargetsRefusesEscapes(t *testing.T) {
	cases := []struct {
		name, root, dir string
		files           []string
	}{
		{"dir outside root", "/dl", "/etc", []string{"/etc/passwd"}},
		{"dir climbs out", "/dl", "/dl/../etc", []string{"/etc/passwd"}},
		{"file outside dir", "/dl", "/dl", []string{"/etc/passwd"}},
		{"file climbs out", "/dl", "/dl", []string{"/dl/../etc/passwd"}},
		{"file is the dir", "/dl", "/dl", []string{"/dl"}},
		{"file is relative", "/dl", "/dl", []string{"a.iso"}},
		{"root is relative", "dl", "/dl", []string{"/dl/a.iso"}},
		{"root is empty", "", "/dl", []string{"/dl/a.iso"}},
		{"dir is relative", "/dl", "dl", []string{"/dl/a.iso"}},
		{"dir is empty", "/dl", "", []string{"/dl/a.iso"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Targets(c.root, c.dir, c.files); err == nil {
				t.Fatal("expected a refusal")
			}
		})
	}
}

func TestTargetsAllowsTheRootItselfAsTheDir(t *testing.T) {
	if _, err := Targets("/dl", "/dl", []string{"/dl/a.iso"}); err != nil {
		t.Fatalf("Targets: %v", err)
	}
}

func TestDeleteRemovesFilesAndRecordsTheGid(t *testing.T) {
	dataDir := t.TempDir()
	seen := stubRemove(t, nil)
	stubLstat(t, regular("/dl/a.iso"))

	if err := Delete(dataDir, "g1", "/dl", []string{"/dl/a.iso"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !reflect.DeepEqual(*seen, []string{"/dl/a.iso"}) {
		t.Fatalf("removed = %v", *seen)
	}
	// Recorded, or the next daemon start would reload the entry and download it
	// all over again.
	if got := readCleaned(dataDir); !reflect.DeepEqual(got, []string{"g1"}) {
		t.Fatalf("cleaned = %v, want [g1]", got)
	}
}

func TestDeleteWalksOutOfTheDirectoriesItEmptied(t *testing.T) {
	seen := stubRemove(t, nil)
	stubLstat(t, regular("/dl/show/season 1/a.mkv"))

	if err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/show/season 1/a.mkv"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// The file, then up to the root — but never the root itself.
	want := []string{"/dl/show/season 1/a.mkv", "/dl/show/season 1", "/dl/show"}
	if !reflect.DeepEqual(*seen, want) {
		t.Fatalf("removed = %v, want %v", *seen, want)
	}
}

func TestDeleteStopsPruningAtADirectoryThatStillHoldsSomething(t *testing.T) {
	var seen []string
	orig := removeFile
	removeFile = func(p string) error {
		seen = append(seen, p)
		// The file goes; its directory is shared with something else.
		if p == "/dl/show/season 1" {
			return errors.New("directory not empty")
		}
		return nil
	}
	t.Cleanup(func() { removeFile = orig })
	stubLstat(t, regular("/dl/show/season 1/a.mkv"))

	if err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/show/season 1/a.mkv"}); err != nil {
		t.Fatalf("a non-empty directory is not an error: %v", err)
	}
	// Stopped there: /dl/show was never attempted.
	want := []string{"/dl/show/season 1/a.mkv", "/dl/show/season 1"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("removed = %v, want %v", seen, want)
	}
}

func TestDeleteRefusesAnythingButARegularFile(t *testing.T) {
	seen := stubRemove(t, nil)
	// A symlink inside the download dir could point anywhere; a directory is
	// never one of a download's files.
	stubLstat(t, map[string]fs.FileMode{
		"/dl/link":   fs.ModeSymlink,
		"/dl/nested": fs.ModeDir,
	})

	err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/link", "/dl/nested"})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("err = %v, want a refusal", err)
	}
	if len(*seen) != 0 {
		t.Fatalf("nothing should have been removed, got %v", *seen)
	}
}

func TestDeleteTreatsAnAlreadyGoneFileAsDone(t *testing.T) {
	seen := stubRemove(t, nil)
	stubLstat(t, nil) // every path reads as missing

	if err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/a.iso"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(*seen) != 0 {
		t.Fatalf("removed = %v, want nothing", *seen)
	}
}

func TestDeleteReportsTheFirstFailureButAttemptsEveryFile(t *testing.T) {
	var seen []string
	orig := removeFile
	removeFile = func(p string) error {
		seen = append(seen, p)
		if p == "/dl/a.iso" {
			return errors.New("busy")
		}
		return nil
	}
	t.Cleanup(func() { removeFile = orig })
	stubLstat(t, regular("/dl/a.iso", "/dl/b.iso"))

	err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/a.iso", "/dl/b.iso"})
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("err = %v, want the first failure", err)
	}
	if !reflect.DeepEqual(seen, []string{"/dl/a.iso", "/dl/b.iso"}) {
		t.Fatalf("removed = %v, want both attempted", seen)
	}
}

func TestDeleteSurfacesAnLstatFailure(t *testing.T) {
	stubRemove(t, nil)
	orig := lstatFile
	lstatFile = func(string) (os.FileInfo, error) { return nil, errors.New("permission denied") }
	t.Cleanup(func() { lstatFile = orig })

	err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/a.iso"})
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("err = %v", err)
	}
}

func TestDeleteIgnoresAFileThatVanishedBetweenLstatAndRemove(t *testing.T) {
	stubRemove(t, fs.ErrNotExist)
	stubLstat(t, regular("/dl/a.iso"))

	if err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/a.iso"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDeleteSkipsTheSessionRecordWhenNothingWasRemoved(t *testing.T) {
	dataDir := t.TempDir()
	stubRemove(t, nil)
	stubLstat(t, nil)

	if err := Delete(dataDir, "g1", "/dl", []string{"/dl/a.iso"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "cleaned.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("nothing was deleted, so nothing should be dropped from the session")
	}
}

func TestDeleteWithoutAGidStillRemovesFiles(t *testing.T) {
	dataDir := t.TempDir()
	seen := stubRemove(t, nil)
	stubLstat(t, regular("/dl/a.iso"))

	if err := Delete(dataDir, "", "/dl", []string{"/dl/a.iso"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(*seen) == 0 {
		t.Fatal("the file should still go")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "cleaned.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("no gid, nothing to record")
	}
}

func TestDeleteReportsAFailedSessionRecord(t *testing.T) {
	stubRemove(t, nil)
	stubLstat(t, regular("/dl/a.iso"))
	stubWriteErr(t, errors.New("disk full"))

	err := Delete(t.TempDir(), "g1", "/dl", []string{"/dl/a.iso"})
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("err = %v", err)
	}
}

// within is tested directly for this one: Targets rejects a relative root or dir
// before it ever gets here, so filepath.Rel cannot fail by that route — but the
// guard is what keeps a Rel failure from reading as "inside", which is the
// dangerous way to be wrong, so it stays and is pinned here.
func TestWithinRefusesWhatItCannotCompare(t *testing.T) {
	if within("/dl", "relative/path") {
		t.Fatal("a path Rel cannot resolve must not count as contained")
	}
	if !within("/dl", "/dl/a.iso") {
		t.Fatal("a plain child must be contained")
	}
	if within("/dl", "/dl2") {
		t.Fatal("a sibling whose name merely starts the same must not be")
	}
}
