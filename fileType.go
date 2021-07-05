package main

import (
	"net/http"
	"os"
)

func getFileContentType(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	buffer := make([]byte, 512)

	_, err = f.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)
	if contentType == "application/octet-stream" {
		// Detect other content types
	}
	return contentType, nil
}
