package readpos

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

const (
	maximumPackageBytes = 4 << 20
	maximumChapterBytes = 16 << 20
	maximumSpineItems   = 20000
)

type spineItem struct {
	file *zip.File
	size int64 // uncompressed bytes, a cheap proxy for reading progress
}

func openSpine(reader io.ReaderAt, size int64) ([]spineItem, error) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, err
	}
	files := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		files[file.Name] = file
	}
	var container struct {
		Rootfiles []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := readXML(files["META-INF/container.xml"], &container); err != nil || len(container.Rootfiles) == 0 {
		return nil, errors.New("missing META-INF/container.xml rootfile")
	}
	opfPath := container.Rootfiles[0].Path
	var pkg struct {
		Manifest []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err := readXML(files[opfPath], &pkg); err != nil {
		return nil, fmt.Errorf("package document: %w", err)
	}
	if len(pkg.Spine) == 0 || len(pkg.Spine) > maximumSpineItems {
		return nil, errors.New("package spine is empty or too large")
	}
	hrefs := make(map[string]string, len(pkg.Manifest))
	for _, item := range pkg.Manifest {
		hrefs[item.ID] = item.Href
	}
	base := path.Dir(opfPath)
	spine := make([]spineItem, 0, len(pkg.Spine))
	for _, ref := range pkg.Spine {
		href, ok := hrefs[ref.IDRef]
		if !ok {
			return nil, fmt.Errorf("spine item %q is not in the manifest", ref.IDRef)
		}
		if unescaped, err := url.PathUnescape(href); err == nil {
			href = unescaped
		}
		file := files[path.Join(base, href)]
		if file == nil {
			return nil, fmt.Errorf("spine file %q is missing", href)
		}
		spine = append(spine, spineItem{file: file, size: int64(file.UncompressedSize64)})
	}
	return spine, nil
}

func readXML(file *zip.File, into any) error {
	if file == nil {
		return errors.New("file is missing")
	}
	data, err := readZip(file, maximumPackageBytes)
	if err != nil {
		return err
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false
	return decoder.Decode(into)
}

func readZip(file *zip.File, limit int64) ([]byte, error) {
	if int64(file.UncompressedSize64) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", file.Name, limit)
	}
	handle, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	data, err := io.ReadAll(io.LimitReader(handle, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", file.Name, limit)
	}
	return data, nil
}
