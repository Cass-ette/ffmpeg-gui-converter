package main

import (
	"strings"
	"testing"
)

func TestAudioConversionAndExtractionPresets(t *testing.T) {
	byKey := make(map[string]Preset, len(presets))
	for _, preset := range presets {
		byKey[preset.Key] = preset
	}

	if !containsString(categoryLabels(), categoryAudio) {
		t.Fatalf("category labels do not include %q", categoryAudio)
	}
	if !containsString(categoryLabels(), categoryAudioExtract) {
		t.Fatalf("category labels do not include %q", categoryAudioExtract)
	}

	for _, key := range []string{"audio_mp3", "extract_mp3", "extract_wav"} {
		preset, ok := byKey[key]
		if !ok {
			t.Fatalf("missing preset %q", key)
		}
		if key == "audio_mp3" && preset.Category != categoryAudio {
			t.Errorf("%s category = %q, want %q", key, preset.Category, categoryAudio)
		}
		if strings.HasPrefix(key, "extract_") && preset.Category != categoryAudioExtract {
			t.Errorf("%s category = %q, want %q", key, preset.Category, categoryAudioExtract)
		}

		args := (&ConverterUI{}).buildFFmpegArgs(ConversionRequest{
			InputPath:  "input.mp4",
			OutputPath: "output." + preset.Extension,
			Preset:     preset,
			Resolution: resolutionOptions[1],
			FrameMode:  "crop",
			FPS:        30,
		})
		command := strings.Join(args, " ")
		if !strings.Contains(command, "-vn") {
			t.Errorf("%s args do not disable video output: %v", key, args)
		}
		if strings.Contains(command, "-vf") {
			t.Errorf("%s args unexpectedly include video filters: %v", key, args)
		}
	}
}
