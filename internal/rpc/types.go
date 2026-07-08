// Package rpc implements a typed client for the aria2 JSON-RPC interface.
// aria2 encodes all numbers as strings on the wire; getters parse them and
// degrade to zero values on malformed input.
package rpc

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
)

// Error is a JSON-RPC error object returned by aria2.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return "aria2: " + e.Message + " (" + strconv.Itoa(e.Code) + ")" }

// URI is one source of a file.
type URI struct {
	URI    string `json:"uri"`
	Status string `json:"status"` // used | waiting
}

// File is one file within a download.
type File struct {
	Index           string `json:"index"`
	Path            string `json:"path"`
	Length          string `json:"length"`
	CompletedLength string `json:"completedLength"`
	Selected        string `json:"selected"` // "true" | "false"
	URIs            []URI  `json:"uris"`
}

func (f File) Len() int64       { return atoi64(f.Length) }
func (f File) Completed() int64 { return atoi64(f.CompletedLength) }
func (f File) IsSelected() bool { return f.Selected == "true" }

// BTInfo is the bittorrent section of a status response.
type BTInfo struct {
	AnnounceList [][]string `json:"announceList"`
	Comment      string     `json:"comment"`
	Mode         string     `json:"mode"`
	Info         struct {
		Name string `json:"name"`
	} `json:"info"`
}

// Status is aria2.tellStatus / tellActive / tellWaiting / tellStopped item.
type Status struct {
	GID             string   `json:"gid"`
	Status          string   `json:"status"` // active|waiting|paused|error|complete|removed
	TotalLength     string   `json:"totalLength"`
	CompletedLength string   `json:"completedLength"`
	UploadLength    string   `json:"uploadLength"`
	Bitfield        string   `json:"bitfield"`
	DownloadSpeed   string   `json:"downloadSpeed"`
	UploadSpeed     string   `json:"uploadSpeed"`
	InfoHash        string   `json:"infoHash"`
	NumSeeders      string   `json:"numSeeders"`
	Seeder          string   `json:"seeder"`
	PieceLength     string   `json:"pieceLength"`
	NumPieces       string   `json:"numPieces"`
	Connections     string   `json:"connections"`
	ErrorCode       string   `json:"errorCode"`
	ErrorMessage    string   `json:"errorMessage"`
	FollowedBy      []string `json:"followedBy"`
	Following       string   `json:"following"`
	Dir             string   `json:"dir"`
	Files           []File   `json:"files"`
	BitTorrent      *BTInfo  `json:"bittorrent"`
}

func (s Status) Total() int64     { return atoi64(s.TotalLength) }
func (s Status) Completed() int64 { return atoi64(s.CompletedLength) }
func (s Status) Uploaded() int64  { return atoi64(s.UploadLength) }
func (s Status) DownSpeed() int64 { return atoi64(s.DownloadSpeed) }
func (s Status) UpSpeed() int64   { return atoi64(s.UploadSpeed) }
func (s Status) Pieces() int      { return int(atoi64(s.NumPieces)) }
func (s Status) PieceLen() int64  { return atoi64(s.PieceLength) }
func (s Status) Seeds() int       { return int(atoi64(s.NumSeeders)) }
func (s Status) Conns() int       { return int(atoi64(s.Connections)) }
func (s Status) IsTorrent() bool  { return s.InfoHash != "" }

// IsSeeding reports whether the download has finished and the torrent is now
// only uploading. aria2 keeps such a torrent in the active list with
// Status=="active", so callers must check this to show a distinct state.
func (s Status) IsSeeding() bool { return s.Status == "active" && s.Seeder == "true" }

// IsMetadata reports whether this is aria2's transient magnet-metadata
// download — the placeholder entry that resolves into the real torrent (its
// only "file" is the [METADATA] marker).
func (s Status) IsMetadata() bool {
	return len(s.Files) > 0 && strings.HasPrefix(s.Files[0].Path, "[METADATA]")
}

// Progress returns completion in [0,1].
func (s Status) Progress() float64 {
	t := s.Total()
	if t <= 0 {
		return 0
	}
	return float64(s.Completed()) / float64(t)
}

// Ratio returns upload/completed share ratio.
func (s Status) Ratio() float64 {
	c := s.Completed()
	if c <= 0 {
		return 0
	}
	return float64(s.Uploaded()) / float64(c)
}

// Name derives a display name: torrent name, else first file base name,
// else first URI base name, else the gid.
func (s Status) Name() string {
	if s.IsMetadata() {
		h := s.InfoHash
		if len(h) > 8 {
			h = h[:8]
		}
		if h == "" {
			return "fetching metadata"
		}
		return "metadata · " + h
	}
	if s.BitTorrent != nil && s.BitTorrent.Info.Name != "" {
		return s.BitTorrent.Info.Name
	}
	if len(s.Files) > 0 {
		if p := s.Files[0].Path; p != "" && p[0] != '[' { // aria2 uses "[MEMORY]..." placeholders
			return filepath.Base(p)
		}
		if len(s.Files[0].URIs) > 0 {
			return filepath.Base(s.Files[0].URIs[0].URI)
		}
	}
	return s.GID
}

// Peer is one bittorrent peer.
type Peer struct {
	PeerID        string `json:"peerId"`
	IP            string `json:"ip"`
	Port          string `json:"port"`
	Bitfield      string `json:"bitfield"`
	AmChoking     string `json:"amChoking"`
	PeerChoking   string `json:"peerChoking"`
	DownloadSpeed string `json:"downloadSpeed"`
	UploadSpeed   string `json:"uploadSpeed"`
	Seeder        string `json:"seeder"`
}

func (p Peer) DownSpeed() int64 { return atoi64(p.DownloadSpeed) }
func (p Peer) UpSpeed() int64   { return atoi64(p.UploadSpeed) }

// ServerStat is one file's connected servers, from aria2.getServers (the
// HTTP/FTP analog of peers).
type ServerStat struct {
	Index   string       `json:"index"`
	Servers []ServerInfo `json:"servers"`
}

// ServerInfo is one mirror aria2 is currently downloading from.
type ServerInfo struct {
	URI           string `json:"uri"`
	CurrentURI    string `json:"currentUri"`
	DownloadSpeed string `json:"downloadSpeed"`
}

func (s ServerInfo) DownSpeed() int64 { return atoi64(s.DownloadSpeed) }

// GlobalStat is aria2.getGlobalStat.
type GlobalStat struct {
	DownloadSpeed   string `json:"downloadSpeed"`
	UploadSpeed     string `json:"uploadSpeed"`
	NumActive       string `json:"numActive"`
	NumWaiting      string `json:"numWaiting"`
	NumStopped      string `json:"numStopped"`
	NumStoppedTotal string `json:"numStoppedTotal"`
}

func (g GlobalStat) DownSpeed() int64 { return atoi64(g.DownloadSpeed) }
func (g GlobalStat) UpSpeed() int64   { return atoi64(g.UploadSpeed) }
func (g GlobalStat) Active() int      { return int(atoi64(g.NumActive)) }
func (g GlobalStat) Waiting() int     { return int(atoi64(g.NumWaiting)) }
func (g GlobalStat) Stopped() int     { return int(atoi64(g.NumStopped)) }

// Notification is an aria2 push event received over websocket.
type Notification struct {
	Method string // e.g. aria2.onDownloadComplete
	GIDs   []string
}

func atoi64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// rawParams decodes notification params of the form [{"gid":"..."}].
func gidsFromParams(raw json.RawMessage) []string {
	var items []struct {
		GID string `json:"gid"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it.GID != "" {
			out = append(out, it.GID)
		}
	}
	return out
}
