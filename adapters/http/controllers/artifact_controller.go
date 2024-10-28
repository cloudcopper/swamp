package controllers

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cloudcopper/swamp/adapters/http/viewmodels"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/ports"
	"github.com/go-chi/chi/v5"
)

type ArtifactController struct {
	log                ports.Logger
	render             infra.Render
	artifactRepository domain.ArtifactRepository
	aritfactStorage    ports.ArtifactStorage
}

func NewArtifactController(log ports.Logger, render infra.Render, artifactRepository domain.ArtifactRepository, aritfactStorage ports.ArtifactStorage) *ArtifactController {
	log = log.With(slog.String("entity", "ArtifactController"))
	s := &ArtifactController{
		log:                log,
		render:             render,
		artifactRepository: artifactRepository,
		aritfactStorage:    aritfactStorage,
	}
	return s
}

func (c *ArtifactController) Get(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	artifactID := chi.URLParam(r, "artifactID")
	// WARN Reroute - due to problem in go-chi
	// see also warning in app.go
	if strings.HasSuffix(artifactID, ".tar.gz") {
		c.DownloadGzip(w, r)
		return
	}
	if strings.HasSuffix(artifactID, ".zip") {
		c.DownloadZip(w, r)
		return
	}

	artifact, err := c.artifactRepository.FindByID(repoID, artifactID, ports.WithRelationship(true))
	if err == ports.ErrRecordNotFound { // 404
		c.renderArtifactNotFound(w, repoID, artifactID, err)
		return
	}
	if err != nil { // 500
		c.renderServerError(w, repoID, artifactID, err)
		return
	}

	var data *viewmodels.Artifact = viewmodels.NewArtifact(artifact)
	c.render.HTML(w, http.StatusOK, "artifact", data)
}

func (c *ArtifactController) DownloadZip(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	artifactID, _ := strings.CutSuffix(chi.URLParam(r, "artifactID"), ".zip")
	artifact, err := c.artifactRepository.FindByID(repoID, artifactID, ports.WithRelationship(true))
	if err == ports.ErrRecordNotFound { // 404
		c.renderArtifactNotFound(w, repoID, artifactID, err)
		return
	}
	if err != nil { // 500
		c.renderServerError(w, repoID, artifactID, err)
		return
	}
	if artifact.State.IsBroken() { // 422
		c.render.HTML(w, http.StatusUnprocessableEntity, "artifact-broken", artifact)
		return
	}
	files := artifact.Files

	// Set headers for zip file
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+artifact.ArtifactID+".zip")

	// Create zip writer directly on the response writer
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Loop over the files and add each one to the zip
	for _, modelFile := range files {
		if !func() (cont bool) {
			fileName := modelFile.Name
			filePath := filepath.Join(artifact.Storage, fileName)
			file, err := c.aritfactStorage.OpenFile(artifact.Storage, artifact.ArtifactID, fileName)
			if err != nil {
				c.renderFileError(w, artifact, "open file", filePath, err)
				return
			}
			defer file.Close()

			// Create a file entry in the zip archive
			fileWriter, err := zipWriter.Create(fileName)
			if err != nil {
				c.renderFileError(w, artifact, "create file in zip", filePath, err)
				return
			}
			// Copy the file content into the zip archive
			if _, err := io.Copy(fileWriter, file); err != nil {
				c.renderFileError(w, artifact, "write file to zip", filePath, err)
				return
			}

			return true
		}() {
			return
		}
	}
}

func (c *ArtifactController) DownloadGzip(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	artifactID, _ := strings.CutSuffix(chi.URLParam(r, "artifactID"), ".tar.gz")
	artifact, err := c.artifactRepository.FindByID(repoID, artifactID, ports.WithRelationship(true))
	if err == ports.ErrRecordNotFound { // 404
		c.renderArtifactNotFound(w, repoID, artifactID, err)
		return
	}
	if err != nil { // 500
		c.renderServerError(w, repoID, artifactID, err)
		return
	}
	if artifact.State.IsBroken() { // 422
		c.render.HTML(w, http.StatusUnprocessableEntity, "artifact-broken", artifact)
		return
	}
	files := artifact.Files

	// Set headers for tar.gz file
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment; filename="+artifact.ArtifactID+".tar.gz")

	// Create gzip writer directly on the response writer
	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()
	// Create tar writer inside the gzip writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Loop over the files and add each one to the tar archive
	for _, modelFile := range files {
		if !func() (cont bool) {
			fileName := modelFile.Name
			filePath := filepath.Join(artifact.Storage, fileName)
			file, err := c.aritfactStorage.OpenFile(artifact.Storage, artifact.ArtifactID, fileName)
			if err != nil {
				c.renderFileError(w, artifact, "open file", filePath, err)
				return
			}
			defer file.Close()

			// Get file information
			fileInfo, err := file.Stat()
			if err != nil {
				c.renderFileError(w, artifact, "stat file", filePath, err)
				return
			}

			// Create tar header based on the file info
			header := &tar.Header{
				Name: fileName, // TODO Check with nested subdirectories
				Size: fileInfo.Size(),
				Mode: int64(fileInfo.Mode()),
			}
			if err := tarWriter.WriteHeader(header); err != nil {
				c.renderFileError(w, artifact, "write tar header", filePath, err)
				return
			}

			// Copy the file content into the tar archive
			if _, err := io.Copy(tarWriter, file); err != nil {
				c.renderFileError(w, artifact, "write file to tar", filePath, err)
				return
			}

			return true
		}() {
			return
		}
	}
}

func (c *ArtifactController) DownloadSingleFile(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "repoID")
	artifactID, _ := strings.CutSuffix(chi.URLParam(r, "artifactID"), ".tar.gz")
	artifact, err := c.artifactRepository.FindByID(repoID, artifactID, ports.WithRelationship(true))
	if err == ports.ErrRecordNotFound { // 404
		c.renderArtifactNotFound(w, repoID, artifactID, err)
		return
	}
	if err != nil { // 500
		c.renderServerError(w, repoID, artifactID, err)
		return
	}
	if artifact.State.IsBroken() { // 422
		c.render.HTML(w, http.StatusUnprocessableEntity, "artifact-broken", artifact)
		return
	}

	filename := chi.URLParam(r, "*")
	files := artifact.Files
	for _, modelFile := range files {
		if modelFile.Name != filename {
			continue
		}

		if modelFile.State.IsBroken() {
			c.render.HTML(w, http.StatusUnprocessableEntity, "file-broken", modelFile)
			return
		}

		func() {
			// Open file
			filePath := filepath.Join(artifact.Storage, filename)
			file, err := c.aritfactStorage.OpenFile(artifact.Storage, artifact.ArtifactID, filename)
			if err != nil {
				c.renderFileError(w, artifact, "open file", filePath, err)
				return
			}
			defer file.Close()

			// Detect proper mime
			ext := filepath.Ext(filename)
			mimeType := mime.TypeByExtension(ext)
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}

			// Set headers for tar.gz file
			w.Header().Set("Content-Type", mimeType)
			w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filename))

			// Use io.Copy to stream the file to the response
			_, err = io.Copy(w, file)
			if err != nil {
				c.log.Error("write file failed", slog.Any("repoID", repoID), slog.Any("artifactID", artifactID), slog.Any("filename", filename), slog.Any("err", err))
				c.renderFileError(w, artifact, "open file", filePath, err)
			}
		}()

		return
	}

	c.renderFileNotFound(w, artifact, filename)
}

func (c *ArtifactController) renderFileNotFound(w http.ResponseWriter, artifact *models.Artifact, filename string) {
	type Data struct {
		Artifact *models.Artifact
		Filename string
	}
	c.render.HTML(w, http.StatusNotFound, "errors/artifact-file-not-found", Data{artifact, filename})
}

func (c *ArtifactController) renderArtifactNotFound(w http.ResponseWriter, repoID models.RepoID, artifactID models.ArtifactID, err error) {
	type Data struct {
		RepoID     models.RepoID
		ArtifactID models.ArtifactID
		Error      error
	}
	c.render.HTML(w, http.StatusNotFound, "errors/artifact-not-found", Data{repoID, artifactID, err})
}

func (c *ArtifactController) renderServerError(w http.ResponseWriter, repoID models.RepoID, artifactID models.ArtifactID, err error) {
	type Data struct {
		RepoID     models.RepoID
		ArtifactID models.ArtifactID
		Error      error
	}
	c.render.HTML(w, http.StatusInternalServerError, "errors/artifact-server-error", Data{repoID, artifactID, err})
}

func (c *ArtifactController) renderFileError(w http.ResponseWriter, artifact *models.Artifact, op, filename string, err error) {
	c.log.Error("file error", slog.Any("repoID", artifact.RepoID), slog.Any("artifactID", artifact.ArtifactID), slog.Any("op", op), slog.Any("filename", filename), slog.Any("err", err))
	type Data struct {
		Artifact *models.Artifact
		Op       string
		Filename string
		Error    error
	}
	c.render.HTML(w, http.StatusInternalServerError, "errors/artifact-file-error", Data{artifact, op, filename, err})
}
