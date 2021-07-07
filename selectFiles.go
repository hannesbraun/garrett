package main

import (
	"github.com/gabriel-vasile/mimetype"
	"io/ioutil"
	"path"
)

var supportedMimeTypes = []string{
	"audio/mpeg",
	"audio/wave",
	"audio/aiff",
	"audio/flac",
	"application/ogg",
}

func filesFromDirectory(dir string) []string {
	result := make([]string, 0)

	items, err := ioutil.ReadDir(dir)
	if err != nil {
		return result
	}

	for _, item := range items {
		if !item.IsDir() {
			mimeType, err := mimetype.DetectFile(path.Join(dir, item.Name()))
			if err != nil {
				continue
			}

			supported := false
			for _, supportedMimeType := range supportedMimeTypes {
				if supportedMimeType == mimeType.String() {
					supported = true
					break
				}
			}

			if supported {
				result = append(result, item.Name())
			}
		} else {
			result = append(result, filesFromDirectory(path.Join(dir, item.Name()))...)
		}
	}

	return result
}
