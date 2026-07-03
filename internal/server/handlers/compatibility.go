package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ClientCompat holds client detection and compatibility rules.
type ClientCompat struct{}

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

// AdaptMediaSourceForClient creates a modified copy of MediaSource based on client capabilities.
// It performs a deep copy to avoid mutating the original map.
func (cc *ClientCompat) AdaptMediaSourceForClient(clientType ClientType, ms map[string]any) map[string]any {
	// Deep copy to avoid side effects on original map
	adapted := cc.deepCopyMediaSource(ms)

	switch clientType {
	case ClientTypeInfuse:
		// Infuse prefers direct streaming info and requires codec validation
		adapted["SupportsDirectStream"] = true
		adapted["SupportsDirectPlay"] = true
		adapted["RequiredHttpHeaders"] = map[string]any{}

	case ClientTypeKodi:
		// Kodi handles Range requests and bitrate switching well
		adapted["SupportsDirectPlay"] = true
		adapted["SupportsDirectStream"] = true
		adapted["MaxBitrate"] = 0 // Unlimited

	case ClientTypeMXPlayer:
		// MXPlayer needs explicit subtitle and codec info
		if _, ok := adapted["MediaStreams"]; !ok {
			adapted["MediaStreams"] = []any{}
		}
		// Ensure all streams have proper info
		cc.validateMediaStreams(adapted)

	case ClientTypeNPlayer:
		// nPlayer is strict about codec info - validate all streams
		adapted["SupportsDirectPlay"] = true
		cc.validateMediaStreams(adapted)

	case ClientTypePlex:
		// Plex-like clients may expect HTTP protocol explicitly
		adapted["Protocol"] = "Http"
		cc.validateMediaStreams(adapted)

	case ClientTypeWeb:
		// Web browsers need HLS/DASH support info
		adapted["Protocol"] = "Http"
		adapted["SupportsHLS"] = true

	default:
		// Unknown client: provide basic compatibility flags
		adapted["SupportsDirectPlay"] = true
		adapted["SupportsDirectStream"] = true
	}

	return adapted
}

// deepCopyMediaSource performs a deep copy of a MediaSource map to avoid mutations.
func (cc *ClientCompat) deepCopyMediaSource(ms map[string]any) map[string]any {
	data, err := json.Marshal(ms)
	if err != nil {
		// Fallback: return original if marshaling fails
		return ms
	}
	var copy map[string]any
	if err := json.Unmarshal(data, &copy); err != nil {
		// Fallback: return original if unmarshaling fails
		return ms
	}
	return copy
}

// validateMediaStreams ensures all MediaStreams have required fields for strict clients.
func (cc *ClientCompat) validateMediaStreams(ms map[string]any) {
	streams, ok := ms["MediaStreams"].([]any)
	if !ok || len(streams) == 0 {
		return
	}

	validated := make([]any, 0, len(streams))
	for _, s := range streams {
		stream, ok := s.(map[string]any)
		if !ok {
			validated = append(validated, s)
			continue
		}

		// Ensure critical fields exist
		if _, ok := stream["Codec"]; !ok {
			stream["Codec"] = "unknown"
		}
		if _, ok := stream["CodecTag"]; !ok {
			stream["CodecTag"] = ""
		}
		if _, ok := stream["DisplayTitle"]; !ok {
			stream["DisplayTitle"] = ""
		}
		if _, ok := stream["IsInterlaced"]; !ok {
			stream["IsInterlaced"] = false
		}
		if _, ok := stream["BitRate"]; !ok {
			stream["BitRate"] = 0
		}

		validated = append(validated, stream)
	}
	ms["MediaStreams"] = validated
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

// CanDirectPlay checks if client supports direct playback without transcoding.
func (cc *ClientCompat) CanDirectPlay(clientType ClientType) bool {
	switch clientType {
	case ClientTypeVLC, ClientTypeKodi, ClientTypeNPlayer, ClientTypeInfuse:
		return true
	default:
		return false
	}
}
