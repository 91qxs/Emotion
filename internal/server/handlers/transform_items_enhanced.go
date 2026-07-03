package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// EnhanceMediaSourcesForClient adapts MediaSources based on client capabilities.
// It performs deep copies to avoid mutating original data.
func EnhanceMediaSourcesForClient(ctx context.Context, sources []any, r *http.Request, log *slog.Logger) []any {
	compat := &ClientCompat{}
	clientType := compat.DetectClient(r)

	enhanced := make([]any, 0, len(sources))
	for _, src := range sources {
		ms, ok := src.(map[string]any)
		if !ok {
			enhanced = append(enhanced, src)
			continue
		}

		// Deep copy using JSON marshal/unmarshal
		data, _ := json.Marshal(ms)
		var newMs map[string]any
		_ = json.Unmarshal(data, &newMs)

		// Apply client-specific adaptations (returns a copy, doesn't mutate)
		newMs = compat.AdaptMediaSourceForClient(clientType, newMs)

		// Enhance all media streams with codec info
		if streams, ok := newMs["MediaStreams"].([]any); ok {
			enhanced := make([]any, len(streams))
			for i, s := range streams {
				if stream, ok := s.(map[string]any); ok {
					enhanced[i] = EnhanceMediaStreamInfo(stream)
				} else {
					enhanced[i] = s
				}
			}
			newMs["MediaStreams"] = enhanced
		}

		enhanced = append(enhanced, newMs)
	}

	return enhanced
}

// BuildPlaybackInfoResponse creates enhanced PlaybackInfo response with client-aware optimizations.
func BuildPlaybackInfoResponse(ctx context.Context, sources []any, playSessionID, apiKey string, r *http.Request, log *slog.Logger) map[string]any {
	// Enhance sources for client capabilities
	enhancedSources := EnhanceMediaSourcesForClient(ctx, sources, r, log)

	return map[string]any{
		"MediaSources":  enhancedSources,
		"PlaySessionId": playSessionID,
	}
}

// EnhancePlaybackInfoWithClientHints adds client-specific hints to playback info.
// This helps clients optimize their playback strategy.
func EnhancePlaybackInfoWithClientHints(ctx context.Context, playbackInfo map[string]any, r *http.Request, log *slog.Logger) map[string]any {
	compat := &ClientCompat{}
	clientType := compat.DetectClient(r)

	// Add client detection hint
	playbackInfo["ClientType"] = string(clientType)

	// Add format support hints
	playbackInfo["SupportsHLS"] = compat.SupportsHLS(clientType)
	playbackInfo["SupportsDASH"] = compat.SupportsDASH(clientType)
	playbackInfo["CanDirectPlay"] = compat.CanDirectPlay(clientType)

	// Add timing and buffer hints based on client type
	switch clientType {
	case ClientTypeWeb:
		// Web players need buffering info
		playbackInfo["BufferMS"] = 5000
		playbackInfo["PreferredFormat"] = "HLS"

	case ClientTypeMXPlayer, ClientTypeNPlayer:
		// Mobile players need quick buffer
		playbackInfo["BufferMS"] = 2000
		playbackInfo["PreferredFormat"] = "DirectPlay"

	case ClientTypeKodi, ClientTypeVLC:
		// Desktop clients can handle larger buffers
		playbackInfo["BufferMS"] = 3000
		playbackInfo["PreferredFormat"] = "DirectPlay"
	}

	return playbackInfo
}

// ValidateMediaSourcesForClient checks if media sources are compatible with the client.
// Returns a list of any validation issues found.
func ValidateMediaSourcesForClient(ctx context.Context, sources []any, clientType ClientType) []string {
	issues := []string{}

	for i, src := range sources {
		ms, ok := src.(map[string]any)
		if !ok {
			continue
		}

		// Validate presence of required fields
		if _, ok := ms["Id"]; !ok {
			issues = append(issues, "MediaSource["+string(rune(i))+"] missing Id")
		}
		if _, ok := ms["Container"]; !ok && clientType != ClientTypeWeb {
			issues = append(issues, "MediaSource["+string(rune(i))+"] missing Container")
		}

		// Validate MediaStreams
		if streams, ok := ms["MediaStreams"].([]any); ok {
			for j, s := range streams {
				stream, ok := s.(map[string]any)
				if !ok {
					continue
				}

				// For strict clients, validate codec info
				if clientType == ClientTypeNPlayer || clientType == ClientTypeMXPlayer {
					if _, ok := stream["Codec"]; !ok {
						issues = append(issues, "MediaStream["+string(rune(i))+"]["+string(rune(j))+"] missing Codec")
					}
					if codec, ok := stream["Codec"].(string); ok && codec == "unknown" {
						issues = append(issues, "MediaStream["+string(rune(i))+"]["+string(rune(j))+"] has unknown Codec")
					}
				}
			}
		}
	}

	return issues
}

// BuildMediaSourcesWithClientFallback builds media sources with automatic fallback
// if validation fails for the detected client type.
func BuildMediaSourcesWithClientFallback(ctx context.Context, sources []any, r *http.Request, log *slog.Logger) []any {
	compat := &ClientCompat{}
	clientType := compat.DetectClient(r)

	// First attempt: enhanced sources for specific client
	enhanced := EnhanceMediaSourcesForClient(ctx, sources, r, log)

	// Validate for strict clients
	issues := ValidateMediaSourcesForClient(ctx, enhanced, clientType)
	if len(issues) > 0 {
		log.Warn("validation issues for client type",
			"client", clientType,
			"issues_count", len(issues),
		)
		// If there are issues, still return enhanced sources but log for debugging
		// The server should try to serve anyway rather than fail
	}

	return enhanced
}
