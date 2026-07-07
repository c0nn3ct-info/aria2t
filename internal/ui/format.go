package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FmtBytes renders n like the design mocks: one decimal below 10 units,
// integers above ("628 MiB", "5.8 GiB", "48 MiB").
func FmtBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	v := float64(n)
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	if v < 10 {
		return fmt.Sprintf("%.1f %s", v, units[i])
	}
	return fmt.Sprintf("%.0f %s", v, units[i])
}

// FmtSpeed renders a transfer rate, em-dash when idle. Unlike sizes, the
// design keeps one decimal for MiB-and-up speeds ("12.4 MiB/s") and
// integers for KiB ("720 KiB/s").
func FmtSpeed(n int64) string {
	if n <= 0 {
		return "-" // ASCII, not "—": the em-dash is 2 cells on some terminals
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	v := float64(n)
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i >= 2 {
		return fmt.Sprintf("%.1f %s/s", v, units[i])
	}
	return fmt.Sprintf("%.0f %s/s", v, units[i])
}

// FmtETA estimates remaining time from bytes left and current speed.
func FmtETA(remaining, speed int64) string {
	if remaining <= 0 || speed <= 0 {
		return "-" // ASCII, not "—" (2 cells on some terminals → column drift)
	}
	return FmtDuration(time.Duration(remaining/speed) * time.Second)
}

// FmtDuration renders like "19s", "7m 12s", "1h 4m".
func FmtDuration(d time.Duration) string {
	s := int64(d.Seconds())
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm %ds", s/60, s%60)
	default:
		return fmt.Sprintf("%dh %dm", s/3600, (s%3600)/60)
	}
}

// FmtAgo renders a past timestamp like "2m ago".
func FmtAgo(t time.Time, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// Bar renders the design's progress bar as separately-stylable halves:
// "━━━━╸" (done, with a half-cell cap) and "────" (rest).
func Bar(frac float64, width int) (filled, empty string) {
	if width <= 0 {
		return "", ""
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	cells := int(frac*float64(width) + 0.5)
	switch {
	case cells <= 0:
		return "", strings.Repeat("─", width)
	case cells >= width:
		return strings.Repeat("━", width), ""
	default:
		return strings.Repeat("━", cells-1) + "╸", strings.Repeat("─", width-cells)
	}
}

// Pieces buckets a hex bitfield (MSB-first, 4 pieces per hex digit) into
// width cells: █ complete, ▓ partial, ░ empty.
func Pieces(bitfieldHex string, numPieces, width int) string {
	if numPieces <= 0 || width <= 0 {
		return strings.Repeat("░", max(width, 0))
	}
	have := make([]bool, numPieces)
	for i, c := range bitfieldHex {
		v, err := strconv.ParseUint(string(c), 16, 8)
		if err != nil {
			continue
		}
		for bit := 0; bit < 4; bit++ {
			idx := i*4 + bit
			if idx < numPieces && v&(1<<(3-bit)) != 0 {
				have[idx] = true
			}
		}
	}
	var b strings.Builder
	for cell := 0; cell < width; cell++ {
		lo := cell * numPieces / width
		hi := (cell + 1) * numPieces / width
		if hi <= lo {
			hi = lo + 1
		}
		done := 0
		for i := lo; i < hi && i < numPieces; i++ {
			if have[i] {
				done++
			}
		}
		switch {
		case done == hi-lo:
			b.WriteRune('█')
		case done > 0:
			b.WriteRune('▓')
		default:
			b.WriteRune('░')
		}
	}
	return b.String()
}

// plural returns "s" unless n is 1.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// trunc shortens s to w cells with an ellipsis, without padding.
func trunc(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}

// bitfieldProgress returns the fraction of pieces present in a hex bitfield
// (MSB-first, 4 pieces per hex digit), clamped to [0,1]. Used for per-peer
// completion in the detail view.
func bitfieldProgress(hex string, numPieces int) float64 {
	if numPieces <= 0 {
		return 0
	}
	have := 0
	for i, c := range hex {
		v, err := strconv.ParseUint(string(c), 16, 8)
		if err != nil {
			continue
		}
		for bit := 0; bit < 4; bit++ {
			if idx := i*4 + bit; idx < numPieces && v&(1<<(3-bit)) != 0 {
				have++
			}
		}
	}
	return float64(have) / float64(numPieces)
}

// friendlyError renders an aria2 failure in plain language. code is aria2's
// numeric errorCode; msg is its raw message, used as the fallback for codes
// this table does not cover. See aria2's EXIT STATUS documentation.
func friendlyError(code, msg string) string {
	switch code {
	case "2":
		return "connection timed out"
	case "3":
		return "file not found on server (404)"
	case "4":
		return "file not found on server (too many 404s)"
	case "5":
		return "download too slow — aborted"
	case "6":
		return "network problem"
	case "8":
		return "server does not support resuming"
	case "9":
		return "not enough disk space"
	case "11":
		return "already downloading the same file"
	case "12":
		return "already downloading the same torrent"
	case "13":
		return "file already exists"
	case "15":
		return "could not open the destination file"
	case "16":
		return "could not create or truncate the file"
	case "17":
		return "file read/write error"
	case "18":
		return "could not create the download directory"
	case "19":
		return "couldn't resolve host (name lookup failed)"
	case "20":
		return "bad or unreadable metalink"
	case "22":
		return "server rejected the request (bad response header)"
	case "23":
		return "too many redirects"
	case "24":
		return "authentication (login) failed"
	case "25":
		return "could not parse the .torrent file"
	case "26":
		return ".torrent file was corrupt or incomplete"
	case "27":
		return "bad magnet link"
	}
	if msg != "" {
		return msg
	}
	if code != "" && code != "0" {
		return "error " + code
	}
	return "error"
}

var sparkLevels = []rune("▁▂▃▄▅▆▇█")

// Spark renders the trailing samples as a block sparkline scaled to max.
func Spark(samples []int64, max int64, width int) string {
	if width <= 0 {
		return ""
	}
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}
	var b strings.Builder
	for i := 0; i < width-len(samples); i++ {
		b.WriteRune(' ')
	}
	for _, s := range samples {
		lvl := 0
		if max > 0 && s > 0 {
			lvl = int(float64(s) / float64(max) * float64(len(sparkLevels)-1))
			if lvl >= len(sparkLevels) {
				lvl = len(sparkLevels) - 1
			}
			if lvl == 0 {
				lvl = 1 // visible floor for non-zero traffic
			}
		}
		b.WriteRune(sparkLevels[lvl])
	}
	return b.String()
}

// ParseLimit normalizes a user-entered rate limit to a value aria2
// accepts: an integer, optionally suffixed K or M (nothing else). "∞", ""
// and "0" mean unlimited; G and fractional inputs are converted, since
// aria2 itself rejects them.
func ParseLimit(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "∞" || s == "0" {
		return "0", nil
	}
	up := strings.ToUpper(s)
	num, mult := up, int64(1)
	switch up[len(up)-1] {
	case 'K':
		num, mult = up[:len(up)-1], 1<<10
	case 'M':
		num, mult = up[:len(up)-1], 1<<20
	case 'G':
		num, mult = up[:len(up)-1], 1<<30
	}
	f, err := strconv.ParseFloat(num, 64)
	if err != nil || num == "" || f < 0 {
		return "", fmt.Errorf("bad limit %q (use 0, 5M, 256K…)", s)
	}
	bytes := int64(f * float64(mult))
	switch {
	case bytes == 0:
		return "0", nil
	case bytes%(1<<20) == 0:
		return strconv.FormatInt(bytes>>20, 10) + "M", nil
	case bytes%(1<<10) == 0:
		return strconv.FormatInt(bytes>>10, 10) + "K", nil
	default:
		return strconv.FormatInt(bytes, 10), nil
	}
}

// NormalizeLimit renders an aria2-reported limit (plain bytes, e.g.
// "5242880") back to the compact form the UI uses ("5M"); values it cannot
// parse pass through unchanged.
func NormalizeLimit(s string) string {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return s
	}
	v, _ := ParseLimit(strconv.FormatInt(n, 10))
	return v
}

// FmtLimit renders an aria2 limit value for display ("0" → "∞").
func FmtLimit(s string) string {
	if s == "" || s == "0" {
		return "∞"
	}
	return s
}
