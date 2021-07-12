package main

import (
	"github.com/gabriel-vasile/mimetype"
	"io/ioutil"
	"path"
)

var supportedMimeTypes = []string{
	"audio/mpeg",
	"audio/flac",
	"audio/wav",
	"audio/aiff",
}

// Get convertable files from a directory (recursively)
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

			if isSupportedMimeType(mimeType) {
				result = append(result, path.Join(dir, item.Name()))
			}
		} else {
			// Search subdirectory
			result = append(result, filesFromDirectory(path.Join(dir, item.Name()))...)
		}
	}

	return result
}

func isSupportedMimeType(mimeType *mimetype.MIME) bool {
	for _, supportedMimeType := range supportedMimeTypes {
		if supportedMimeType == mimeType.String() {
			return true
		}
	}

	return false
}
