package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// EnhanceMediaSourcesForClient adapts MediaSources based on client capabilities.
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
		
		// Deep copy
		data, _ := json.Marshal(ms)
		var newMs map[string]any
		_ = json.Unmarshal(data, &newMs)
		
		// Apply client-specific adaptations
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

// BuildPlaybackInfoResponse creates enhanced PlaybackInfo response.
func BuildPlaybackInfoResponse(ctx context.Context, sources []any, playSessionID, apiKey string, r *http.Request, log *slog.Logger) map[string]any {
	// Enhance sources for client
	ensources = EnhanceMediaSourcesForClient(ctx, sources, r, log)
	
	return map[string]any{
		"MediaSources":  ensources,
		"PlaySessionId": playSessionID,
	}
}
