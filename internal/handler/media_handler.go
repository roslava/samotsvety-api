// internal/handler/media_handler.go
package handler

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/imaging"
	"github.com/roslava/samotsvety-api/internal/storage"
)

const maxUploadSize = 10 << 20 // 10 MB

// webpQuality — качество WebP-конвертации (0..100). Для каталожных фото минералов
// держим высоким, чтобы не терять резкость/детали при хорошем сжатии.
const webpQuality = 90

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// slugPattern — тот же формат, что уже используется для slug минерала/статьи на
// фронте (lowercase, латиница/цифры/дефис). Проверяется здесь только для того,
// чтобы не создавать в бакете директории с пробелами/слэшами/юникодом — это НЕ
// замена валидации самой сущности, а защита структуры бакета.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// galleryNamePattern вытаскивает числовой суффикс из имени файла галереи вида
// "ruby07.webp" -> 7, чтобы определить следующий свободный номер.
var galleryNamePattern = regexp.MustCompile(`(\d+)\.webp$`)

// allowedLangs — допустимые значения поля "lang" для картинок с языковым
// оверрайдом (диаграммы/схемы с текстом внутри изображения).
var allowedLangs = map[string]bool{"": true, "ru": true, "en": true}

type MediaHandler struct {
	storage storage.MediaStorage
}

func NewMediaHandler(s storage.MediaStorage) *MediaHandler {
	return &MediaHandler{storage: s}
}

// UploadMedia godoc
// @Summary      Загрузить изображение
// @Description  Принимает multipart-файл (jpeg/png/webp/gif, до 10 МБ) и поле "kind":
// @Description    - "hero"        — минерал, требует "slug", сохраняется как <slug>/hero.webp
// @Description    - "thumbnail"   — минерал, требует "slug", сохраняется как <slug>/thumbnail.webp
// @Description    - "gallery"     — минерал, требует "slug", сохраняется как <slug>/gallery/<slug><NN>.webp
// @Description                      (номер вычисляется автоматически)
// @Description    - "cover"       — обложка статьи, требует "slug" (+ опц. "lang" ru/en),
// @Description                      сохраняется как articles/<slug>/cover.webp или cover-<lang>.webp
// @Description    - "block_image" — картинка в блоке статьи, требует "slug" и "block_index"
// @Description                      (+ опц. "lang"), сохраняется как articles/<slug>/image-<NN>.webp
// @Description                      или image-<NN>-<lang>.webp
// @Description    - "block_pair"  — картинка в блоке-паре статьи, требует "slug", "block_index"
// @Description                      и "pair_index" (1|2) (+ опц. "lang"), сохраняется как
// @Description                      articles/<slug>/image-<NN>-<pair_index>.webp или с суффиксом языка
// @Description  Все изображения всегда конвертируются в WebP независимо от исходного формата.
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Param        file         formData  file    true   "Файл изображения"
// @Param        kind         formData  string  true   "hero | thumbnail | gallery | cover | block_image | block_pair"
// @Param        slug         formData  string  true   "Slug минерала или статьи"
// @Param        block_index  formData  int     false  "Порядковый номер блока в статье (для block_image/block_pair)"
// @Param        pair_index   formData  int     false  "1 или 2 — какая картинка в паре (для block_pair)"
// @Param        lang         formData  string  false  "ru | en — языковой оверрайд (для cover/block_image/block_pair)"
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

	kind := c.PostForm("kind")
	slug := strings.ToLower(strings.TrimSpace(c.PostForm("slug")))
	lang := strings.ToLower(strings.TrimSpace(c.PostForm("lang")))

	if !allowedLangs[lang] {
		RespondWithError(c, http.StatusBadRequest, "Некорректное значение поля \"lang\": ожидается ru или en")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondInternalError(c, "Не удалось прочитать файл")
		return
	}
	defer file.Close()

	switch kind {
	case "hero", "thumbnail", "gallery":
		h.uploadMineralImage(c, kind, slug, file, fileHeader)
	case "cover", "block_image", "block_pair":
		h.uploadArticleImage(c, kind, slug, lang, file, fileHeader)
	default:
		RespondWithError(c, http.StatusBadRequest, "Неизвестное значение поля \"kind\": ожидается hero, thumbnail, gallery, cover, block_image или block_pair")
	}
}

// uploadArticleImage реализует структуру бакета для статей:
//
//	articles/<slug>/cover.webp                — kind=cover, без lang
//	articles/<slug>/cover-<lang>.webp         — kind=cover, с lang (ru/en)
//	articles/<slug>/image-<NN>.webp           — kind=block_image, без lang
//	articles/<slug>/image-<NN>-<lang>.webp    — kind=block_image, с lang
//	articles/<slug>/image-<NN>-<P>.webp       — kind=block_pair (P = pair_index 1|2), без lang
//	articles/<slug>/image-<NN>-<P>-<lang>.webp — kind=block_pair, с lang
//
// NN — порядковый номер блока в статье (поле "block_index" из формы, 1-based).
// Изображение всегда конвертируется в WebP независимо от исходного формата.
func (h *MediaHandler) uploadArticleImage(c *gin.Context, kind, slug, lang string, file io.Reader, fileHeader *multipart.FileHeader) {
	if slug == "" {
		RespondWithError(c, http.StatusBadRequest, "Для изображений статьи обязательно поле \"slug\" — сначала укажите slug в форме статьи")
		return
	}
	if !slugPattern.MatchString(slug) {
		RespondWithError(c, http.StatusBadRequest, "Некорректный slug: разрешены строчные латинские буквы, цифры и дефис")
		return
	}

	var blockIndex, pairIndex int
	if kind == "block_image" || kind == "block_pair" {
		var err error
		blockIndex, err = strconv.Atoi(c.PostForm("block_index"))
		if err != nil || blockIndex < 1 {
			RespondWithError(c, http.StatusBadRequest, "Для картинки блока обязательно поле \"block_index\" (целое число ≥ 1)")
			return
		}
	}
	if kind == "block_pair" {
		var err error
		pairIndex, err = strconv.Atoi(c.PostForm("pair_index"))
		if err != nil || (pairIndex != 1 && pairIndex != 2) {
			RespondWithError(c, http.StatusBadRequest, "Для картинки пары обязательно поле \"pair_index\" (1 или 2)")
			return
		}
	}

	raw, err := io.ReadAll(file)
	if err != nil {
		RespondInternalError(c, "Не удалось прочитать файл")
		return
	}

	webpBytes, err := imaging.ToWebP(c.Request.Context(), raw, webpQuality)
	if err != nil {
		slog.Error("Failed to convert image to webp", "error", err, "filename", fileHeader.Filename, "slug", slug)
		RespondInternalError(c, "Не удалось сконвертировать изображение в WebP")
		return
	}

	langSuffix := ""
	if lang != "" {
		langSuffix = "-" + lang
	}

	var key string
	switch kind {
	case "cover":
		key = fmt.Sprintf("articles/%s/cover%s.webp", slug, langSuffix)
	case "block_image":
		key = fmt.Sprintf("articles/%s/image-%02d%s.webp", slug, blockIndex, langSuffix)
	case "block_pair":
		key = fmt.Sprintf("articles/%s/image-%02d-%d%s.webp", slug, blockIndex, pairIndex, langSuffix)
	}

	url, err := h.storage.UploadAt(c.Request.Context(), key, "image/webp", bytes.NewReader(webpBytes), int64(len(webpBytes)))
	if err != nil {
		slog.Error("Failed to upload article media", "error", err, "key", key)
		RespondInternalError(c, "Не удалось загрузить файл")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": url})
}

// uploadMineralImage реализует структуру бакета для минералов:
//
//	<slug>/hero.webp
//	<slug>/thumbnail.webp
//	<slug>/gallery/<slug><NN>.webp
//
// Изображение всегда конвертируется в WebP независимо от исходного формата.
func (h *MediaHandler) uploadMineralImage(c *gin.Context, kind, slug string, file io.Reader, fileHeader *multipart.FileHeader) {
	if slug == "" {
		RespondWithError(c, http.StatusBadRequest, "Для изображений минерала обязательно поле \"slug\" — сначала укажите slug в форме минерала")
		return
	}
	if !slugPattern.MatchString(slug) {
		RespondWithError(c, http.StatusBadRequest, "Некорректный slug: разрешены строчные латинские буквы, цифры и дефис")
		return
	}

	raw, err := io.ReadAll(file)
	if err != nil {
		RespondInternalError(c, "Не удалось прочитать файл")
		return
	}

	webpBytes, err := imaging.ToWebP(c.Request.Context(), raw, webpQuality)
	if err != nil {
		slog.Error("Failed to convert image to webp", "error", err, "filename", fileHeader.Filename, "slug", slug)
		RespondInternalError(c, "Не удалось сконвертировать изображение в WebP")
		return
	}

	var key string
	switch kind {
	case "hero":
		key = fmt.Sprintf("%s/hero.webp", slug)
	case "thumbnail":
		key = fmt.Sprintf("%s/thumbnail.webp", slug)
	case "gallery":
		nextKey, err := h.nextGalleryKey(c, slug)
		if err != nil {
			slog.Error("Failed to determine next gallery filename", "error", err, "slug", slug)
			RespondInternalError(c, "Не удалось определить имя файла для галереи")
			return
		}
		key = nextKey
	}

	url, err := h.storage.UploadAt(c.Request.Context(), key, "image/webp", bytes.NewReader(webpBytes), int64(len(webpBytes)))
	if err != nil {
		slog.Error("Failed to upload mineral media", "error", err, "key", key)
		RespondInternalError(c, "Не удалось загрузить файл")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": url})
}

// nextGalleryKey вычисляет следующий свободный ключ вида "<slug>/gallery/<slug><NN>.webp",
// глядя на уже загруженные файлы в этой папке. Нумерация всегда двузначная (01, 02, ...),
// расширяется до трёх цифр сама, если файлов набирается больше 99.
func (h *MediaHandler) nextGalleryKey(c *gin.Context, slug string) (string, error) {
	prefix := fmt.Sprintf("%s/gallery/", slug)

	keys, err := h.storage.ListKeys(c.Request.Context(), prefix)
	if err != nil {
		return "", err
	}

	maxN := 0
	for _, key := range keys {
		matches := galleryNamePattern.FindStringSubmatch(key)
		if len(matches) != 2 {
			continue
		}
		n, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if n > maxN {
			maxN = n
		}
	}

	next := maxN + 1
	numStr := strconv.Itoa(next)
	if next < 100 {
		numStr = fmt.Sprintf("%02d", next)
	}

	return fmt.Sprintf("%s%s%s.webp", prefix, slug, numStr), nil
}
