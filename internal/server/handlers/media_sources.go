package handlers

import (
	"fmt"
	"strings"
)

// CodecInfo holds codec detection info.
type CodecInfo struct {
	Codec     string
	Profile   string
	Level     string
	BitDepth  int
	SupportHW bool
}

// DetectCodec extracts codec information from stream metadata.
func DetectCodec(stream map[string]any) *CodecInfo {
	codec := stringVal(stream["codec_name"])
	if codec == "" {
		codec = stringVal(stream["codec_type"])
	}
	
	return &CodecInfo{
		Codec:    normalizeCodec(codec),
		Profile:  stringVal(stream["profile"]),
		Level:    stringVal(stream["level"]),
		BitDepth: int(anyInt64(stream["bits_per_raw_sample"])),
	}
}

// normalizeCodec maps ffprobe codec names to standard names.
func normalizeCodec(codec string) string {
	codec = strings.ToLower(strings.TrimSpace(codec))
	switch codec {
	case "h264", "avc1":
		return "h264"
	case "h265", "hevc":
		return "hevc"
	case "vp8":
		return "vp8"
	case "vp9":
		return "vp9"
	case "av1":
		return "av1"
	case "mpeg2video":
		return "mpeg2"
	case "mpeg4":
		return "mpeg4"
	case "aac":
		return "aac"
	case "ac3":
		return "ac3"
	case "eac3":
		return "eac3"
	case "dca", "dts":
		return "dts"
	case "mp3", "libmp3lame":
		return "mp3"
	case "libopus", "opus":
		return "opus"
	case "libvorbis", "vorbis":
		return "vorbis"
	case "flac":
		return "flac"
	case "pcm_s16le", "pcm":
		return "pcm"
	case "subrip", "ass", "ssa":
		return codec // Return as-is for subtitles
	default:
		return codec
	}
}

// EnhanceMediaStreamInfo adds detailed codec information.
func EnhanceMediaStreamInfo(stream map[string]any) map[string]any {
	if stream == nil {
		stream = make(map[string]any)
	}
	
	codecType := stringVal(stream["codec_type"])
	if codecType == "" {
		return stream
	}
	
	switch codecType {
	case "video":
		enableVideoHWDecoding(stream)
		addVideoColorInfo(stream)
		
	case "audio":
		enableAudioHWDecoding(stream)
		addAudioChannelInfo(stream)
		
	case "subtitle":
		formatSubtitleInfo(stream)
	}
	
	return stream
}

func enableVideoHWDecoding(stream map[string]any) {
	codec := normalizeCodec(stringVal(stream["codec_name"]))
	switch codec {
	case "h264", "hevc", "vp9", "av1":
		stream["SupportsHwDecode"] = true
	default:
		stream["SupportsHwDecode"] = false
	}
}

func addVideoColorInfo(stream map[string]any) {
	if colorSpace := stringVal(stream["color_space"]); colorSpace != "" {
		stream["ColorSpace"] = colorSpace
	}
	if colorTransfer := stringVal(stream["color_transfer"]); colorTransfer != "" {
		stream["ColorTransfer"] = colorTransfer
	}
	if colorPrimaries := stringVal(stream["color_primaries"]); colorPrimaries != "" {
		stream["ColorPrimaries"] = colorPrimaries
	}
}

func enableAudioHWDecoding(stream map[string]any) {
	codec := normalizeCodec(stringVal(stream["codec_name"]))
	switch codec {
	case "aac", "mp3", "flac", "dts", "ac3", "eac3":
		stream["SupportsHwDecode"] = true
	default:
		stream["SupportsHwDecode"] = false
	}
}

func addAudioChannelInfo(stream map[string]any) {
	if channels := anyInt64(stream["channels"]); channels > 0 {
		stream["Channels"] = channels
		stream["ChannelLayout"] = getChannelLayout(int(channels))
	}
	if sampleRate := anyInt64(stream["sample_rate"]); sampleRate > 0 {
		stream["SampleRate"] = sampleRate
	}
}

func getChannelLayout(channels int) string {
	switch channels {
	case 1:
		return "mono"
	case 2:
		return "stereo"
	case 6:
		return "5.1"
	case 8:
		return "7.1"
	default:
		return fmt.Sprintf("%d-channel", channels)
	}
}

func formatSubtitleInfo(stream map[string]any) {
	codec := normalizeCodec(stringVal(stream["codec_name"]))
	stream["SubtitleFormat"] = codec
	stream["IsExternal"] = false
	stream["IsTextSubtitleStream"] = isTextCodec(codec)
}

func isTextCodec(codec string) bool {
	switch codec {
	case "subrip", "ass", "ssa", "webvtt", "srt", "vtt":
		return true
	default:
		return false
	}
}

func stringVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
