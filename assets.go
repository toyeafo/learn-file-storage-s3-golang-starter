package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
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

func getAssetPath(videoID uuid.UUID, mediaType string) string {
	ext := mediaTypetoExt(mediaType)
	return fmt.Sprintf("%s%s", videoID, ext)
}

func mediaTypetoExt(mediaType string) string {
	mediatypesplit := strings.Split(mediaType, "/")
	if len(mediatypesplit) != 2 {
		return ".bin"
	}
	return "." + mediatypesplit[1]
}
