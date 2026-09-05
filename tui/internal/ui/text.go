package ui

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"aria2t/internal/rpc"
)

// safeText turns external text into inert, single-line terminal data.  SGR and
// other escape sequences must never be allowed to reach the renderer from an
// aria2 response, a path, configuration, or error message.
func safeText(s string) string {
	s = ansi.Strip(s)
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

func cellWidth(s string) int { return ansi.StringWidth(s) }

func asciiText(s string) string {
	return strings.NewReplacer(
		"←", "<-", "↑", "up", "↓", "down", "▲", "UL", "▼", "DL",
		"✓", "[OK]", "✗", "[X]", "∞", "unlimited", "…", "...",
		"▸", ">", "▾", "v", "◂", "<", "▪", "*", "⌕", "/",
		"│", "|", "─", "-", "━", "=", "╸", ">",
		"┌", "+", "┐", "+", "└", "+", "┘", "+",
		"╭", "+", "╮", "+", "╰", "+", "╯", "+",
		"█", "#", "▓", "+", "░", ".", "▁", ".", "▂", ":",
		"▃", ":", "▄", "=", "▅", "=", "▆", "#", "▇", "#",
	).Replace(s)
}

// safeKey keeps paste as a paste event while removing terminal controls from
// its payload. Newlines and tabs remain available to multiline text widgets.
func safeKey(msg tea.KeyMsg) tea.KeyMsg {
	if !msg.Paste {
		return msg
	}
	clean := ansi.Strip(string(msg.Runes))
	clean = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r == '\r' {
			return '\n'
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, clean)
	msg.Runes = []rune(clean)
	return msg
}

func safeStatus(s rpc.Status) rpc.Status {
	s.GID = safeText(s.GID)
	s.Status = safeText(s.Status)
	s.InfoHash = safeText(s.InfoHash)
	s.ErrorCode = safeText(s.ErrorCode)
	s.ErrorMessage = safeText(s.ErrorMessage)
	s.Following = safeText(s.Following)
	s.Dir = safeText(s.Dir)
	for i := range s.FollowedBy {
		s.FollowedBy[i] = safeText(s.FollowedBy[i])
	}
	for i := range s.Files {
		s.Files[i].Path = safeText(s.Files[i].Path)
		for j := range s.Files[i].URIs {
			s.Files[i].URIs[j].URI = safeText(s.Files[i].URIs[j].URI)
			s.Files[i].URIs[j].Status = safeText(s.Files[i].URIs[j].Status)
		}
	}
	if s.BitTorrent != nil {
		s.BitTorrent.Comment = safeText(s.BitTorrent.Comment)
		s.BitTorrent.Mode = safeText(s.BitTorrent.Mode)
		s.BitTorrent.Info.Name = safeText(s.BitTorrent.Info.Name)
		for i := range s.BitTorrent.AnnounceList {
			for j := range s.BitTorrent.AnnounceList[i] {
				s.BitTorrent.AnnounceList[i][j] = safeText(s.BitTorrent.AnnounceList[i][j])
			}
		}
	}
	return s
}

func safeSnapshot(s snapshot) snapshot {
	for _, list := range [][]rpc.Status{s.Active, s.Waiting, s.Stopped} {
		for i := range list {
			list[i] = safeStatus(list[i])
		}
	}
	return s
}

func safeDetailData(msg detailDataMsg) detailDataMsg {
	msg.status = safeStatus(msg.status)
	for i := range msg.peers {
		msg.peers[i].PeerID = safeText(msg.peers[i].PeerID)
		msg.peers[i].IP = safeText(msg.peers[i].IP)
		msg.peers[i].Port = safeText(msg.peers[i].Port)
	}
	for i := range msg.servers {
		for j := range msg.servers[i].Servers {
			msg.servers[i].Servers[j].URI = safeText(msg.servers[i].Servers[j].URI)
			msg.servers[i].Servers[j].CurrentURI = safeText(msg.servers[i].Servers[j].CurrentURI)
		}
	}
	return msg
}
