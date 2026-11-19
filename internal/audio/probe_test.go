package audio

import "testing"

func TestParseProbeOutputUsesAudiobookTagAliases(t *testing.T) {
	metadata, err := parseProbeOutput([]byte(`{
		"format":{"duration":"90.5","bit_rate":"256000","tags":{"title":"A Book","author":"Writer","albumartist":"Publisher","grouping":"A Series","series-part":"Book 3.5","read_by":"Reader","date":"2021-06-04"}},
		"streams":[{"codec_type":"audio","codec_name":"flac","sample_rate":"48000","bits_per_raw_sample":"24"},{"codec_type":"video","codec_name":"mjpeg","disposition":{"attached_pic":1}}],
		"chapters":[{"start_time":"0","end_time":"15","tags":{"title":"Opening"}}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "A Book" || metadata.Artist != "Writer" || metadata.AlbumArtist != "Publisher" || metadata.Series != "A Series" || metadata.SeriesIndex != 3.5 || metadata.Narrator != "Reader" || metadata.Duration != 90.5 || metadata.Year != 2021 || metadata.Bitrate != 256000 || metadata.Codec != "flac" || metadata.SampleRate != 48000 || metadata.BitDepth != 24 || !metadata.HasArtwork || len(metadata.Chapters) != 1 {
		t.Fatalf("metadata=%+v", metadata)
	}
}

func TestAudioYearAndBitrateAreBounded(t *testing.T) {
	if parseAudioYear("1999-10-01") != 1999 || parseAudioYear("999") != 0 || parseAudioYear("9999") != 0 {
		t.Fatal("unexpected year parsing")
	}
	if parseAudioBitrate("1411200") != 1411200 || parseAudioBitrate("999") != 0 || parseAudioBitrate("1000000001") != 0 {
		t.Fatal("unexpected bitrate parsing")
	}
}

func TestParseSeriesIndexBoundsAndLabels(t *testing.T) {
	for _, test := range []struct {
		value string
		want  float64
		ok    bool
	}{{"3.5", 3.5, true}, {"Book 12", 12, true}, {"Volume 2.0 extra", 2, true}, {"0", 0, false}, {"NaN", 0, false}, {"1000001", 0, false}} {
		got, ok := parseSeriesIndex(test.value)
		if got != test.want || ok != test.ok {
			t.Fatalf("parseSeriesIndex(%q) = %v, %v; want %v, %v", test.value, got, ok, test.want, test.ok)
		}
	}
}

func TestParseReplayGain(t *testing.T) {
	for _, test := range []struct {
		value string
		want  float64
		ok    bool
	}{{"-7.23 dB", -7.23, true}, {"3.0 DB", 3, true}, {"", 0, false}, {"NaN dB", 0, false}, {"99 dB", 0, false}} {
		got, ok := parseReplayGain(test.value)
		if ok != test.ok || got != test.want {
			t.Fatalf("parseReplayGain(%q) = %v, %v; want %v, %v", test.value, got, ok, test.want, test.ok)
		}
	}
}

func TestParseR128GainAndPeak(t *testing.T) {
	if gain, ok := parseR128Gain("-1536"); !ok || gain != -6 {
		t.Fatalf("R128 gain = %v, %v", gain, ok)
	}
	if peak, ok := parseReplayPeak("0.9876"); !ok || peak != 0.9876 {
		t.Fatalf("peak = %v, %v", peak, ok)
	}
	if _, ok := parseReplayPeak("0"); ok {
		t.Fatal("zero peak was accepted")
	}
}
