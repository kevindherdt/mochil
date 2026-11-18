package webpub

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const maximumOverlayClips = 200000

// Overlay is the narration of a read-along EPUB (EPUB 3 Media Overlays): which
// stretch of which audio file speaks which element of which chapter. Clips are
// [element id, index into Audio, begin seconds, end seconds].
type Overlay struct {
	Audio    []string         `json:"audio"`
	Chapters []OverlayChapter `json:"chapters"`
}

type OverlayChapter struct {
	Href  string  `json:"href"`
	Clips [][]any `json:"clips"`
}

// BuildOverlay reads every media overlay in the EPUB. It returns an empty
// overlay (no chapters) for a book without narration.
func BuildOverlay(reader io.ReaderAt, size int64) (*Overlay, error) {
	_, files, opfPath, pkg, err := openPackage(reader, size)
	if err != nil {
		return nil, err
	}
	base := path.Dir(opfPath)
	byID := map[string]int{}
	for i, item := range pkg.Manifest {
		byID[item.ID] = i
	}
	overlay := &Overlay{Chapters: []OverlayChapter{}, Audio: []string{}}
	audioIndex := map[string]int{}
	total := 0
	for _, ref := range pkg.Spine.Items {
		i, ok := byID[ref.IDRef]
		if !ok || pkg.Manifest[i].Overlay == "" {
			continue
		}
		smilItem, ok := byID[pkg.Manifest[i].Overlay]
		if !ok {
			continue
		}
		smilPath := resolveIn(base, pkg.Manifest[smilItem].Href)
		file := files[smilPath]
		if file == nil {
			continue
		}
		chapter := OverlayChapter{Href: resolveIn(base, pkg.Manifest[i].Href), Clips: [][]any{}}
		pars, err := readSMIL(file, path.Dir(smilPath))
		if err != nil {
			continue
		}
		for _, par := range pars {
			if par.text != chapter.Href || files[par.audio] == nil || par.end <= par.begin {
				continue
			}
			index, seen := audioIndex[par.audio]
			if !seen {
				index = len(overlay.Audio)
				audioIndex[par.audio] = index
				overlay.Audio = append(overlay.Audio, par.audio)
			}
			chapter.Clips = append(chapter.Clips, []any{par.id, index, par.begin, par.end})
			if total++; total > maximumOverlayClips {
				return nil, errors.New("webpub: overlay is too large")
			}
		}
		if len(chapter.Clips) > 0 {
			overlay.Chapters = append(overlay.Chapters, chapter)
		}
	}
	return overlay, nil
}

// resolveIn resolves an EPUB href against base and drops its fragment. The
// fragment has to be split off BEFORE unescaping: a literal "#" in a file name is
// percent-encoded in the package document and the SMIL, so unescaping first turns
// "chapter%231.xhtml" into "chapter#1.xhtml" and then truncates it to "chapter".
func resolveIn(base, href string) string {
	if i := strings.IndexByte(href, '#'); i >= 0 {
		href = href[:i]
	}
	if unescaped, err := url.PathUnescape(href); err == nil {
		href = unescaped
	}
	return path.Clean(path.Join(base, href))
}

type smilPar struct {
	id, text, audio string
	begin, end      float64
}

// readSMIL lists the text/audio pairs of one SMIL document in order.
func readSMIL(file *zip.File, dir string) ([]smilPar, error) {
	if file.UncompressedSize64 > 8*maximumDocumentBytes {
		return nil, errors.New("webpub: overlay document is too large")
	}
	handle, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	decoder := xml.NewDecoder(io.LimitReader(handle, 8*maximumDocumentBytes))
	decoder.Strict = false
	var pars []smilPar
	var current *smilPar
	var textFragment string
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return pars, nil
			}
			return pars, err
		}
		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "par":
				current = &smilPar{}
				for _, attr := range element.Attr {
					if attr.Name.Local == "id" {
						current.id = attr.Value
					}
				}
			case "text":
				if current != nil {
					for _, attr := range element.Attr {
						if attr.Name.Local == "src" {
							current.text = resolveIn(dir, attr.Value)
							textFragment = ""
							if i := strings.IndexByte(attr.Value, '#'); i >= 0 {
								textFragment = attr.Value[i+1:]
							}
						}
					}
				}
			case "audio":
				if current != nil {
					for _, attr := range element.Attr {
						switch attr.Name.Local {
						case "src":
							current.audio = resolveIn(dir, attr.Value)
						case "clipBegin":
							current.begin = clock(attr.Value)
						case "clipEnd":
							current.end = clock(attr.Value)
						}
					}
				}
			}
		case xml.EndElement:
			if element.Name.Local == "par" && current != nil {
				if textFragment != "" {
					current.id = textFragment
				}
				if current.text != "" && current.audio != "" && current.id != "" {
					pars = append(pars, *current)
				}
				current, textFragment = nil, ""
			}
		}
	}
}

// clock parses a SMIL clock value: "12.5s", "1500ms", "1:02:03.5", "2min", "3".
func clock(value string) float64 {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return 0
	case strings.Contains(value, ":"):
		seconds := 0.0
		for _, part := range strings.Split(value, ":") {
			number, _ := strconv.ParseFloat(part, 64)
			seconds = seconds*60 + number
		}
		return seconds
	}
	// Longest suffixes first: "ms" and "min" also end in "s" and "n".
	for _, unit := range []struct {
		suffix string
		length time.Duration
	}{{"ms", time.Millisecond}, {"min", time.Minute}, {"h", time.Hour}, {"s", time.Second}} {
		if strings.HasSuffix(value, unit.suffix) {
			number, _ := strconv.ParseFloat(strings.TrimSuffix(value, unit.suffix), 64)
			return number * unit.length.Seconds()
		}
	}
	number, _ := strconv.ParseFloat(value, 64)
	return number
}

// OpenSeekable returns a stored (uncompressed) archive entry as an io.ReadSeeker
// so audio and video can be served with range requests without unpacking them.
func OpenSeekable(reader io.ReaderAt, size int64, name string) (io.ReadSeeker, int64, bool) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, 0, false
	}
	name = path.Clean(strings.TrimPrefix(name, "/"))
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		if file.Method != zip.Store {
			return nil, 0, false
		}
		offset, err := file.DataOffset()
		if err != nil {
			return nil, 0, false
		}
		length := int64(file.UncompressedSize64)
		return io.NewSectionReader(reader, offset, length), length, true
	}
	return nil, 0, false
}
