package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func getAssetPath(mediaType string) string {
	key := make([]byte, 32)
	rand.Read(key)
	encodedKey := base64.RawURLEncoding.EncodeToString(key)
	ext := mediaTypetoExt(mediaType)
	return fmt.Sprintf("%s%s", encodedKey, ext)
}

func mediaTypetoExt(mediaType string) string {
	mediatypesplit := strings.Split(mediaType, "/")
	if len(mediatypesplit) != 2 {
		return ".bin"
	}
	return "." + mediatypesplit[1]
}
