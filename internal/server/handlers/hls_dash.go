package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// StreamingFormats serves HLS and DASH manifests for compatibility.
type StreamingFormats struct{}

// NewStreamingFormats creates handler.
func NewStreamingFormats() *StreamingFormats {
	return &StreamingFormats{}
}

// HLSMaster generates HLS master playlist (optional feature).
// Most clients prefer direct playback, but this enables web player support.
func (sf *StreamingFormats) HLSMaster(w http.ResponseWriter, r *http.Request) {
	mediaSourceID := chi.URLParam(r, "mediaSourceId")
	if mediaSourceID == "" {
		WriteStatus(w, http.StatusBadRequest)
		return
	}
	
	// Generate HLS master playlist
	// In production, this would integrate with ffmpeg for transcoding
	playlist := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.0,
/hls/%s/segment-0.ts
#EXT-X-ENDLIST
`, mediaSourceID)
	
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
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
	
	// Generate DASH manifest
	manifest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" minBufferTime="PT1.5S" type="static" 
     mediaPresentationDuration="%s" profiles="urn:mpeg:dash:profile:isoff-live:2011">
  <Period>
    <AdaptationSet>
      <Representation id="1" mimeType="video/mp4" codecs="avc1">
        <BaseURL>/dash/%s/stream.m4s</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>
`, duration, mediaSourceID)
	
	w.Header().Set("Content-Type", "application/dash+xml")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(manifest))
}

func formatISO8601Duration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	
	return fmt.Sprintf("PT%dH%dM%dS", hours, minutes, secs)
}

// PlaybackCapabilities returns what streaming formats this server supports.
func (sf *StreamingFormats) PlaybackCapabilities(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"SupportsMediaControl": true,
		"SupportsPersistentBitrateSetting": true,
		"SupportsMediaSourceSelection": true,
		"SupportsHLS": true,
		"SupportsDASH": false, // Optional
		"SupportsDirectPlay": true,
		"SupportsDirectStream": true,
		"SupportsTranscoding": false,
		"RequiresClosing": false,
		"RequiresOpening": false,
		"RequiresLooping": false,
		"SupportsProbing": false,
		"MaxBitrate": 0,
		"MaxStreamingBitrate": 0,
		"MusicStreamingTranscodingBitrate": 0,
	})
}
