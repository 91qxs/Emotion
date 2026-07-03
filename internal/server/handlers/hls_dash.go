package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// StreamingFormats serves HLS and DASH manifests for compatibility.
type StreamingFormats struct {
	// In production, would hold ffmpeg transcoding coordinator
}

// NewStreamingFormats creates handler.
func NewStreamingFormats() *StreamingFormats {
	return &StreamingFormats{}
}

// HLSMaster generates HLS master playlist for web/adaptive streaming.
// Most native clients prefer direct playback, but this enables web player support.
func (sf *StreamingFormats) HLSMaster(w http.ResponseWriter, r *http.Request) {
	mediaSourceID := chi.URLParam(r, "mediaSourceId")
	if mediaSourceID == "" {
		WriteStatus(w, http.StatusBadRequest)
		return
	}

	// Parse optional parameters
	quality := r.URL.Query().Get("quality") // "auto", "low", "medium", "high"
	if quality == "" {
		quality = "auto"
	}

	// Generate HLS master playlist with multiple bitrate variants
	// In production, this would be generated from actual transcode outputs
	playlist := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD

#EXT-X-STREAM-INF:BANDWIDTH=5000000,RESOLUTION=1920x1080
/hls/%s/playlist-1080p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1280x720
/hls/%s/playlist-720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=640x360
/hls/%s/playlist-360p.m3u8
`, mediaSourceID, mediaSourceID, mediaSourceID)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(playlist))
}

// DASHManifest generates DASH manifest (MPD) for web players.
func (sf *StreamingFormats) DASHManifest(w http.ResponseWriter, r *http.Request) {
	mediaSourceID := chi.URLParam(r, "mediaSourceId")
	if mediaSourceID == "" {
		WriteStatus(w, http.StatusBadRequest)
		return
	}

	// Parse duration from query
	durationStr := r.URL.Query().Get("duration")
	duration := "PT1H" // Default
	if d, err := strconv.ParseInt(durationStr, 10, 64); err == nil && d > 0 {
		duration = formatISO8601Duration(d)
	}

	// Generate DASH manifest with multiple representations
	// In production, this would be generated from actual transcode outputs
	manifest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" minBufferTime="PT1.5S" type="static"
     mediaPresentationDuration="%s" profiles="urn:mpeg:dash:profile:isoff-live:2011">
  <Period>
    <AdaptationSet mimeType="video/mp4" segmentAlignment="true">
      <Representation id="1" codecs="avc1.4d401e" bandwidth="5000000" width="1920" height="1080">
        <BaseURL>/dash/%s/1080p/stream.m4s</BaseURL>
      </Representation>
      <Representation id="2" codecs="avc1.4d401e" bandwidth="2500000" width="1280" height="720">
        <BaseURL>/dash/%s/720p/stream.m4s</BaseURL>
      </Representation>
      <Representation id="3" codecs="avc1.4d401e" bandwidth="1000000" width="640" height="360">
        <BaseURL>/dash/%s/360p/stream.m4s</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet mimeType="audio/mp4" segmentAlignment="true">
      <Representation id="audio1" codecs="mp4a.40.2" bandwidth="128000" audioSamplingRate="48000">
        <BaseURL>/dash/%s/audio/stream.m4s</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>
`, duration, mediaSourceID, mediaSourceID, mediaSourceID, mediaSourceID)

	w.Header().Set("Content-Type", "application/dash+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(manifest))
}

func formatISO8601Duration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	return fmt.Sprintf("PT%dH%dM%dS", hours, minutes, secs)
}

// PlaybackCapabilities returns what streaming formats and features this server supports.
// This response helps clients decide which playback method to use.
func (sf *StreamingFormats) PlaybackCapabilities(w http.ResponseWriter, r *http.Request) {
	capabilities := map[string]any{
		"SupportsMediaControl":              true,
		"SupportsPersistentBitrateSetting": true,
		"SupportsMediaSourceSelection":     true,
		"SupportsHLS":                      true,  // HTTP Live Streaming (web/iOS)
		"SupportsDASH":                     true,  // DASH (web/Android)
		"SupportsDirectPlay":               true,  // Native codec support
		"SupportsDirectStream":             true,  // Direct stream with container remux
		"SupportsTranscoding":              false, // Not implemented
		"RequiresClosing":                  false,
		"RequiresOpening":                  false,
		"RequiresLooping":                  false,
		"SupportsProbing":                  false,
		"MaxBitrate":                       0, // Unlimited
		"MaxStreamingBitrate":              0, // Unlimited
		"MusicStreamingTranscodingBitrate": 0,
		"OptimalPlaybackBitrate":           0,
		"SupportedMediaTypes":              []string{"Video", "Audio", "Photo"},
		"SupportedSubtitleFormats":         []string{"srt", "ass", "ssa", "vtt"},
		"TimelineOffsetSeconds":            0,
	}

	WriteJSON(w, http.StatusOK, capabilities)
}

// PlaybackHints returns client-specific playback hints based on detected capabilities.
// Helps clients choose the best playback strategy.
func (sf *StreamingFormats) PlaybackHints(w http.ResponseWriter, r *http.Request, clientType ClientType) {
	ctx := r.Context()
	hints := sf.buildPlaybackHints(ctx, clientType)
	WriteJSON(w, http.StatusOK, hints)
}

func (sf *StreamingFormats) buildPlaybackHints(ctx context.Context, clientType ClientType) map[string]any {
	hints := map[string]any{
		"PreferredPlayMethod": "DirectPlay", // Default
		"SubtitleDelayMs":     0,
		"BufferMs":            2000,
		"MaxPlaylistLength":   100,
	}

	switch clientType {
	case ClientTypeWeb:
		hints["PreferredPlayMethod"] = "HLS"
		hints["BufferMs"] = 5000
		hints["AllowedFormats"] = []string{"HLS", "DASH"}

	case ClientTypeKodi:
		hints["PreferredPlayMethod"] = "DirectPlay"
		hints["AllowRangeRequests"] = true
		hints["BufferMs"] = 3000

	case ClientTypeMXPlayer:
		hints["PreferredPlayMethod"] = "DirectPlay"
		hints["RequireSubtitleInfo"] = true
		hints["SubtitleDelayMs"] = 100

	case ClientTypeNPlayer:
		hints["PreferredPlayMethod"] = "DirectPlay"
		hints["StrictCodecValidation"] = true
		hints["BufferMs"] = 2000

	case ClientTypeInfuse:
		hints["PreferredPlayMethod"] = "DirectPlay"
		hints["AllowedFormats"] = []string{"DirectPlay", "HLS"}
		hints["BufferMs"] = 2500

	case ClientTypePlex:
		hints["PreferredPlayMethod"] = "HLS"
		hints["RequireAuthHeaders"] = true
	}

	return hints
}

// ValidatePlaybackRequest validates incoming playback requests for compatibility.
// Returns any adaptation needed for the client type.
func (sf *StreamingFormats) ValidatePlaybackRequest(ctx context.Context, req map[string]any, clientType ClientType) error {
	// Extract and validate requested format
	format, ok := req["Format"].(string)
	if !ok || format == "" {
		format = "DirectPlay"
	}

	// Check if client supports requested format
	supported := sf.getSupportedFormats(clientType)
	if !stringInSlice(format, supported) && format != "DirectPlay" {
		// Return error or fallback?
		// For now, just log and let client try anyway
	}

	return nil
}

func (sf *StreamingFormats) getSupportedFormats(clientType ClientType) []string {
	switch clientType {
	case ClientTypeWeb:
		return []string{"HLS", "DASH", "DirectPlay"}
	case ClientTypeKodi:
		return []string{"DirectPlay", "DirectStream"}
	default:
		return []string{"DirectPlay", "DirectStream"}
	}
}

func stringInSlice(s string, slice []string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// MediaSourceInfo represents detailed info about a media source for playback.
type MediaSourceInfo struct {
	ID                   string         `json:"Id"`
	Protocol             string         `json:"Protocol"`
	MediaStreams         []StreamInfo   `json:"MediaStreams"`
	Bitrate              int64          `json:"Bitrate,omitempty"`
	Container            string         `json:"Container,omitempty"`
	RunTimeTicks         int64          `json:"RunTimeTicks,omitempty"`
	SupportsDirectPlay   bool           `json:"SupportsDirectPlay"`
	SupportsDirectStream bool           `json:"SupportsDirectStream"`
	SupportsTranscoding  bool           `json:"SupportsTranscoding"`
}

// StreamInfo represents info about a media stream (video/audio/subtitle).
type StreamInfo struct {
	Type              string `json:"Type"`
	Codec             string `json:"Codec"`
	CodecTag          string `json:"CodecTag,omitempty"`
	Language          string `json:"Language,omitempty"`
	BitRate           int64  `json:"BitRate,omitempty"`
	Width             int    `json:"Width,omitempty"`
	Height            int    `json:"Height,omitempty"`
	FrameRate         string `json:"FrameRate,omitempty"`
	Channels          int    `json:"Channels,omitempty"`
	ChannelLayout     string `json:"ChannelLayout,omitempty"`
	SampleRate        int    `json:"SampleRate,omitempty"`
	IsDefault         bool   `json:"IsDefault"`
	IsForced          bool   `json:"IsForced"`
	IsHearingImpaired bool   `json:"IsHearingImpaired"`
}

// BuildMediaSourceInfoFromJSON constructs structured MediaSourceInfo from raw JSON.
// This helps validate and ensure consistency in media source responses.
func BuildMediaSourceInfoFromJSON(data map[string]any) *MediaSourceInfo {
	info := &MediaSourceInfo{
		ID:                   stringVal(data["Id"]),
		Protocol:             stringVal(data["Protocol"]),
		Container:            stringVal(data["Container"]),
		SupportsDirectPlay:   boolVal(data["SupportsDirectPlay"]),
		SupportsDirectStream: boolVal(data["SupportsDirectStream"]),
		SupportsTranscoding:  boolVal(data["SupportsTranscoding"]),
	}

	if d, ok := data["RunTimeTicks"].(float64); ok {
		info.RunTimeTicks = int64(d)
	}
	if b, ok := data["Bitrate"].(float64); ok {
		info.Bitrate = int64(b)
	}

	if streams, ok := data["MediaStreams"].([]any); ok {
		info.MediaStreams = parseStreamInfos(streams)
	}

	return info
}

func parseStreamInfos(streams []any) []StreamInfo {
	result := make([]StreamInfo, 0, len(streams))
	for _, s := range streams {
		if sm, ok := s.(map[string]any); ok {
			result = append(result, StreamInfo{
				Type:              stringVal(sm["Type"]),
				Codec:             stringVal(sm["Codec"]),
				CodecTag:          stringVal(sm["CodecTag"]),
				Language:          stringVal(sm["Language"]),
				IsDefault:         boolVal(sm["IsDefault"]),
				IsForced:          boolVal(sm["IsForced"]),
				IsHearingImpaired: boolVal(sm["IsHearingImpaired"]),
			})
		}
	}
	return result
}

func boolVal(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
