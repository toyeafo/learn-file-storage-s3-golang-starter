package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type VidStreams struct {
	Streams []struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
	} `json:"streams"`
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func (cfg apiConfig) getS3AssetURL(assetKey string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, assetKey)
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

func getVideoAspectRatio(inputPath string) (string, error) {
	fileProbeJson := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", inputPath)
	var out bytes.Buffer
	fileProbeJson.Stdout = &out
	err := fileProbeJson.Run()
	if err != nil {
		return "", err
	}
	var streams VidStreams
	err = json.Unmarshal(out.Bytes(), &streams)
	if err != nil {
		return "", err
	}
	return calculateAspectRatio(streams.Streams[0].Width, streams.Streams[0].Height), nil
}

func calculateAspectRatio(width, height int) string {
	if height == 0 {
		return "0:1"
	}

	gcd := func(a, b int) int {
		for b != 0 {
			temp := b
			b = a % b
			a = temp
		}
		return a
	}
	divisor := gcd(width, height)
	simplifiedWidth := width / divisor
	simplifiedHeight := height / divisor
	// ratio := fmt.Sprintf("%d:%d", simplifiedWidth, simplifiedHeight)
	switch {
	default:
		return "other"
	case simplifiedWidth > simplifiedHeight:
		return "landscape/"
	case simplifiedHeight > simplifiedWidth:
		return "portrait/"
	}
}

func processVideoForFastStart(inputPath string) (string, error) {
	outputFile := inputPath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFile)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}
	return outputFile, nil
}
