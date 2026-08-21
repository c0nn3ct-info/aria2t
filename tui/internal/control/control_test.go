package control

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// stubRemove swaps removeFile, recording every path it was asked to delete
// and returning err for each.
func stubRemove(t *testing.T, err error) *[]string {
	t.Helper()
	var seen []string
	orig := removeFile
	removeFile = func(p string) error {
		seen = append(seen, p)
		return err
	}
	t.Cleanup(func() { removeFile = orig })
	return &seen
}

// stubWriteErr makes every write fail.
func stubWriteErr(t *testing.T, err error) {
	t.Helper()
	orig := writeFile
	writeFile = func(string, []byte, fs.FileMode) error { return err }
	t.Cleanup(func() { writeFile = orig })
}

// stubReadErr makes every read fail with err.
func stubReadErr(t *testing.T, err error) {
	t.Helper()
	orig := readFile
	readFile = func(string) ([]byte, error) { return nil, err }
	t.Cleanup(func() { readFile = orig })
}

// writeCleaned seeds the cleaned-gid record in dataDir.
func writeCleaned(t *testing.T, dataDir string, gids []string) {
	t.Helper()
	raw, _ := json.Marshal(gids)
	if err := os.WriteFile(cleanedPath(dataDir), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPaths(t *testing.T) {
	// A multi-file torrent's control file is <dir>/<name>.aria2, beside the
	// folder aria2 created for it — not next to any single file.
	got := Paths("/dl", "Movie", []string{"/dl/Movie/a.mkv", "/dl/Movie/b.srt"})
	want := []string{"/dl/Movie/a.mkv.aria2", "/dl/Movie/b.srt.aria2", "/dl/Movie.aria2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Paths = %v, want %v", got, want)
	}
}

// The daemon runs with --bt-save-metadata, so every torrent leaves an
// <infohash>.torrent in the download folder that aria2 never lists among the
// download's files — it outlived removal until it was named here.
func TestSaved(t *testing.T) {
	got := Saved("/dl", "1E838BF16C32054F7D7A02197A20DEB0545E74B7")
	want := []string{"/dl/1e838bf16c32054f7d7a02197a20deb0545e74b7.torrent"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Saved = %v, want %v", got, want)
	}
}

// The hash reaches this from the browser and is turned into a path to unlink,
// so anything that is not plainly a hash yields nothing rather than a filename.
func TestSavedRefusesAnythingThatIsNotAHash(t *testing.T) {
	for _, hash := range []string{
		"",
		"short",
		"../../../etc/passwd",
		"1e838bf16c32054f7d7a02197a20deb0545e74b",   // 39
		"1e838bf16c32054f7d7a02197a20deb0545e74b77", // 41
		"1e838bf16c32054f7d7a02197a20deb0545e74bZ",  // not hex
		"1e838bf16c32054f7d7a02197a20deb0545e74b/",
	} {
		if got := Saved("/dl", hash); got != nil {
			t.Fatalf("Saved(%q) = %v, want nil", hash, got)
		}
	}
	if got := Saved("", "1e838bf16c32054f7d7a02197a20deb0545e74b7"); got != nil {
		t.Fatalf("Saved with no dir = %v, want nil", got)
	}
}

func TestLeftovers(t *testing.T) {
	got := Leftovers("/dl", "Movie", "1e838bf16c32054f7d7a02197a20deb0545e74b7", []string{"/dl/Movie/a.mkv"})
	want := []string{
		"/dl/Movie/a.mkv.aria2",
		"/dl/Movie.aria2",
		"/dl/1e838bf16c32054f7d7a02197a20deb0545e74b7.torrent",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Leftovers = %v, want %v", got, want)
	}
	// A plain HTTP download has no hash and no torrent name: control file only.
	if got := Leftovers("/dl", "", "", []string{"/dl/a.iso"}); !reflect.DeepEqual(got, []string{"/dl/a.iso.aria2"}) {
		t.Fatalf("Leftovers without a torrent = %v", got)
	}
}

func TestPathsSkipsPlaceholdersAndDuplicates(t *testing.T) {
	got := Paths("/dl", "", []string{"[METADATA]abc", "", "/dl/f.iso", "/dl/f.iso"})
	want := []string{"/dl/f.iso.aria2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Paths = %v, want %v", got, want)
	}
}

func TestPathsWithoutDirSkipsTorrentName(t *testing.T) {
	if got := Paths("", "Movie", nil); len(got) != 0 {
		t.Fatalf("Paths = %v, want none", got)
	}
}

func TestCleanDeletesAndRecords(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.iso.aria2")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Clean(dir, "gid1", []string{file}); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if _, err := os.Stat(file); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("control file still present: %v", err)
	}
	if got := readCleaned(dir); !reflect.DeepEqual(got, []string{"gid1"}) {
		t.Fatalf("cleaned = %v, want [gid1]", got)
	}
}

func TestCleanMissingFileRecordsNothing(t *testing.T) {
	dir := t.TempDir()
	if err := Clean(dir, "gid1", []string{filepath.Join(dir, "gone.aria2")}); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if got := readCleaned(dir); got != nil {
		t.Fatalf("cleaned = %v, want none", got)
	}
}

func TestCleanReportsFirstDeleteError(t *testing.T) {
	seen := stubRemove(t, errors.New("boom"))
	err := Clean(t.TempDir(), "gid1", []string{"/a.aria2", "/b.aria2"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v, want boom", err)
	}
	if len(*seen) != 2 {
		t.Fatalf("attempted %d deletions, want both", len(*seen))
	}
}

func TestCleanWithoutGidSkipsTheRecord(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.aria2")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Clean(dir, "", []string{file}); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if got := readCleaned(dir); got != nil {
		t.Fatalf("cleaned = %v, want none", got)
	}
}

func TestCleanSurfacesRecordFailure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.aria2")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	stubWriteErr(t, errors.New("no space"))
	if err := Clean(dir, "gid1", []string{file}); err == nil || err.Error() != "no space" {
		t.Fatalf("err = %v, want no space", err)
	}
}

func TestRecordIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	writeCleaned(t, dir, []string{"gid1"})
	if err := record(dir, "gid1"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if got := readCleaned(dir); !reflect.DeepEqual(got, []string{"gid1"}) {
		t.Fatalf("cleaned = %v, want one entry", got)
	}
}

func TestReadCleanedIgnoresMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(cleanedPath(dir), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readCleaned(dir); got != nil {
		t.Fatalf("cleaned = %v, want none", got)
	}
}

const session = `http://example.com/a.iso
 gid=aaaa
 dir=/dl
http://example.com/b.iso
 gid=bbbb
 dir=/dl
`

func TestFilterSessionDropsCleanedEntries(t *testing.T) {
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "session.txt")
	if err := os.WriteFile(sessionFile, []byte(session), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCleaned(t, dir, []string{"aaaa"})

	if err := FilterSession(sessionFile, dir); err != nil {
		t.Fatalf("FilterSession: %v", err)
	}
	raw, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if strings.Contains(got, "a.iso") || strings.Contains(got, "aaaa") {
		t.Fatalf("cleaned entry survived:\n%s", got)
	}
	if !strings.Contains(got, "b.iso") || !strings.Contains(got, "gid=bbbb") {
		t.Fatalf("kept entry lost:\n%s", got)
	}
	if _, err := os.Stat(cleanedPath(dir)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("cleaned record survived: %v", err)
	}
}

func TestFilterSessionDroppingEverythingEmptiesTheFile(t *testing.T) {
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "session.txt")
	if err := os.WriteFile(sessionFile, []byte(session), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCleaned(t, dir, []string{"aaaa", "bbbb"})

	if err := FilterSession(sessionFile, dir); err != nil {
		t.Fatalf("FilterSession: %v", err)
	}
	raw, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 0 {
		t.Fatalf("session = %q, want empty", raw)
	}
}

func TestFilterSessionWithoutRecordIsANoop(t *testing.T) {
	dir := t.TempDir()
	if err := FilterSession(filepath.Join(dir, "session.txt"), dir); err != nil {
		t.Fatalf("FilterSession: %v", err)
	}
}

func TestFilterSessionWithoutSessionForgetsTheRecord(t *testing.T) {
	dir := t.TempDir()
	writeCleaned(t, dir, []string{"aaaa"})
	if err := FilterSession(filepath.Join(dir, "session.txt"), dir); err != nil {
		t.Fatalf("FilterSession: %v", err)
	}
	if got := readCleaned(dir); got != nil {
		t.Fatalf("cleaned = %v, want forgotten", got)
	}
}

func TestFilterSessionReportsReadError(t *testing.T) {
	dir := t.TempDir()
	writeCleaned(t, dir, []string{"aaaa"})
	// The record read must still succeed, so only fail the session read.
	orig := readFile
	readFile = func(p string) ([]byte, error) {
		if strings.HasSuffix(p, "session.txt") {
			return nil, errors.New("io fail")
		}
		return orig(p)
	}
	t.Cleanup(func() { readFile = orig })

	if err := FilterSession(filepath.Join(dir, "session.txt"), dir); err == nil ||
		err.Error() != "io fail" {
		t.Fatalf("err = %v, want io fail", err)
	}
}

func TestFilterSessionReportsWriteError(t *testing.T) {
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "session.txt")
	if err := os.WriteFile(sessionFile, []byte(session), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCleaned(t, dir, []string{"aaaa"})
	stubWriteErr(t, errors.New("read-only fs"))

	if err := FilterSession(sessionFile, dir); err == nil || err.Error() != "read-only fs" {
		t.Fatalf("err = %v, want read-only fs", err)
	}
}

func TestForgetWithoutRecordIsANoop(t *testing.T) {
	if err := forget(t.TempDir()); err != nil {
		t.Fatalf("forget: %v", err)
	}
}

func TestForgetReportsRealErrors(t *testing.T) {
	stubRemove(t, errors.New("perm"))
	if err := forget(t.TempDir()); err == nil || err.Error() != "perm" {
		t.Fatalf("err = %v, want perm", err)
	}
}

func TestReadCleanedIgnoresUnreadableFile(t *testing.T) {
	stubReadErr(t, errors.New("io"))
	if got := readCleaned(t.TempDir()); got != nil {
		t.Fatalf("cleaned = %v, want none", got)
	}
}
