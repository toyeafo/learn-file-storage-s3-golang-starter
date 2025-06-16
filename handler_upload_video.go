package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

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

	vidMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "error retrieving vid details", err)
		return
	}

	if vidMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "not authorised to make the update", nil)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing memory", err)
		return
	}

	file, fileheader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error reading file from form", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(fileheader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error parsing media type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "media type not allowed", nil)
		return
	}

	imageFile := getAssetPath(mediaType)
	// imageFileLoc := filepath.Join(cfg.assetsRoot, imageFile)

	fileCreate, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating file on disk", err)
		return
	}
	defer os.Remove(fileCreate.Name())
	defer fileCreate.Close()

	_, err = io.Copy(fileCreate, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error saving file to disk", err)
		return
	}
	fileCreate.Seek(0, io.SeekStart)

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &imageFile,
		Body:        fileCreate,
		ContentType: &mediaType,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading file to S3", err)
		return
	}

	fullFileLoc := cfg.getS3AssetURL(imageFile)
	vidMetadata.VideoURL = &fullFileLoc

	err = cfg.db.UpdateVideo(vidMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vidMetadata)
}
