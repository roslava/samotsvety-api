// internal/handler/media_handler.go
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/storage"
)

const maxUploadSize = 10 << 20 // 10 MB

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type MediaHandler struct {
	storage storage.MediaStorage
}

func NewMediaHandler(s storage.MediaStorage) *MediaHandler {
	return &MediaHandler{storage: s}
}

// UploadMedia godoc
// @Summary      Загрузить иллюстрацию для статьи
// @Description  Принимает multipart-файл (jpeg/png/webp/gif, до 10 МБ), сохраняет в Object Storage,
// @Description  возвращает публичный URL, который затем кладётся в content_blocks статьи.
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Файл изображения"
// @Success      201  {object}  gin.H
// @Failure      400  {object}  handler.ErrorResponse
// @Router       /api/v1/media [post]
func (h *MediaHandler) UploadMedia(c *gin.Context) {
	if h.storage == nil {
		RespondWithError(c, http.StatusServiceUnavailable, "Хранилище файлов не настроено")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Файл не передан (ожидается поле form-data \"file\")")
		return
	}

	if fileHeader.Size > maxUploadSize {
		RespondWithError(c, http.StatusBadRequest, "Файл слишком большой (максимум 10 МБ)")
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		RespondWithError(c, http.StatusBadRequest, "Недопустимый тип файла: разрешены jpeg, png, webp, gif")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondInternalError(c, "Не удалось прочитать файл")
		return
	}
	defer file.Close()

	url, err := h.storage.Upload(c.Request.Context(), "articles", fileHeader.Filename, contentType, file, fileHeader.Size)
	if err != nil {
		slog.Error("Failed to upload media", "error", err, "filename", fileHeader.Filename)
		RespondInternalError(c, "Не удалось загрузить файл")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": url})
}
