package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Limit request size to 1GB
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// Authenticate the user
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// Retrieve video metadata
	vidMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error retrieving vid details", err)
		return
	}
	if vidMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "not authorised to make the update", nil)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing memory", err)
		return
	}

	// Read file from form
	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error reading file from form", err)
		return
	}
	defer file.Close()

	// Check content type
	mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing media type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "media type not allowed", nil)
		return
	}

	videoKey := getAssetPath(mediaType)

	// Save uploaded file to temp disk file
	tempInputFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating temp file on disk", err)
		return
	}
	defer os.Remove(tempInputFile.Name())
	defer tempInputFile.Close()

	if _, err = io.Copy(tempInputFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error saving file to disk", err)
		return
	}

	if _, err := tempInputFile.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to rewind file", err)
		return
	}

	// Process the video for fast start
	processedPath, err := processVideoForFastStart(tempInputFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video processing failed", err)
		return
	}
	defer os.Remove(processedPath)

	processedFile, err := os.Open(processedPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to open processed video", err)
		return
	}
	defer processedFile.Close()

	// Get the aspect ratio of the video
	ratio, err := getVideoAspectRatio(processedPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting video aspect ratio", err)
		return
	}
	s3Key := fmt.Sprintf("%s%s", ratio, videoKey)

	// Upload the video to S3
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(s3Key),
		Body:        processedFile,
		ContentType: aws.String(mediaType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading file to S3", err)
		return
	}

	// Update DB with video URL
	fileURL := cfg.getS3AssetURL(s3Key)
	vidMetadata.VideoURL = &fileURL
	err = cfg.db.UpdateVideo(vidMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video", err)
		return
	}

	fmt.Printf("Successfully uploaded video %s (%s) to %s\n", videoID, ratio, fileURL)
	respondWithJSON(w, http.StatusOK, vidMetadata)
}
