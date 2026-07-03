package handlers

import (
	"net/http"
	"strings"
)

// ClientCompat holds client detection and compatibility rules.
type ClientCompat struct {}

// ClientType identifies the media player/client.
type ClientType string

const (
	ClientTypeInfuse       ClientType = "infuse"
	ClientTypeKodi         ClientType = "kodi"
	ClientTypeVLC          ClientType = "vlc"
	ClientTypeMXPlayer     ClientType = "mxplayer"
	ClientTypeNPlayer      ClientType = "nplayer"
	ClientTypePlex         ClientType = "plex"
	ClientTypeEmby         ClientType = "emby"
	ClientTypeSubsonicAnd  ClientType = "subsonic-android"
	ClientTypeWeb          ClientType = "web"
	ClientTypeUnknown      ClientType = "unknown"
)

// DetectClient identifies the media player from User-Agent header.
func (cc *ClientCompat) DetectClient(r *http.Request) ClientType {
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	
	switch {
	case strings.Contains(ua, "infuse"):
		return ClientTypeInfuse
	case strings.Contains(ua, "kodi") || strings.Contains(ua, "xbmc"):
		return ClientTypeKodi
	case strings.Contains(ua, "vlc"):
		return ClientTypeVLC
	case strings.Contains(ua, "mxplayer"):
		return ClientTypeMXPlayer
	case strings.Contains(ua, "nplayer"):
		return ClientTypeNPlayer
	case strings.Contains(ua, "plex"):
		return ClientTypePlex
	case strings.Contains(ua, "emby"):
		return ClientTypeEmby
	case strings.Contains(ua, "subsonic") && strings.Contains(ua, "android"):
		return ClientTypeSubsonicAnd
	case strings.Contains(ua, "mozilla") || strings.Contains(ua, "webkit"):
		return ClientTypeWeb
	default:
		return ClientTypeUnknown
	}
}

// AdaptMediaSourceForClient modifies MediaSource based on client capabilities.
func (cc *ClientCompat) AdaptMediaSourceForClient(clientType ClientType, ms map[string]any) map[string]any {
	switch clientType {
	case ClientTypeInfuse:
		// Infuse prefers transcoding support info
		ms["SupportsDirectStream"] = true
		ms["SupportsDirectPlay"] = true
		ms["RequiredHttpHeaders"] = map[string]any{}
		
	case ClientTypeKodi:
		// Kodi handles Range requests well
		ms["SupportsDirectPlay"] = true
		ms["SupportsDirectStream"] = true
		
	case ClientTypeMXPlayer:
		// MXPlayer needs clear subtitle info
		if _, ok := ms["MediaStreams"]; !ok {
			ms["MediaStreams"] = []any{}
		}
		
	case ClientTypeNPlayer:
		// nPlayer is very strict about codec info
		ms["SupportsDirectPlay"] = true
		
	case ClientTypePlex:
		// Plex-like clients may expect different response format
		ms["Protocol"] = "Http"
		
	case ClientTypeWeb:
		// Web browsers need HLS/DASH
		ms["Protocol"] = "Http"
	}
	
	return ms
}

// SupportsHLS checks if client supports HLS streaming.
func (cc *ClientCompat) SupportsHLS(clientType ClientType) bool {
	switch clientType {
	case ClientTypeWeb, ClientTypeInfuse, ClientTypePlex:
		return true
	default:
		return false
	}
}

// SupportsDASH checks if client supports DASH streaming.
func (cc *ClientCompat) SupportsDASH(clientType ClientType) bool {
	switch clientType {
	case ClientTypeWeb, ClientTypeInfuse:
		return true
	default:
		return false
	}
}
