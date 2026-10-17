// internal/handler/media_handler.go
package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/imaging"
	"github.com/roslava/samotsvety-api/internal/repository"
	"github.com/roslava/samotsvety-api/internal/storage"
)

const maxUploadSize = 10 << 20 // 10 MB

var articleConversionSlots = make(chan struct{}, 2)

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
var galleryNamePattern = regexp.MustCompile(`^(\d+)\.webp$`)

// allowedLangs — допустимые значения поля "lang" для картинок с языковым
// оверрайдом (диаграммы/схемы с текстом внутри изображения).
var allowedLangs = map[string]bool{"": true, "ru": true, "en": true}

type MediaHandler struct {
	storage storage.MediaStorage
	posts   repository.PostRepository
}

func NewMediaHandler(s storage.MediaStorage, posts ...repository.PostRepository) *MediaHandler {
	h := &MediaHandler{storage: s}
	if len(posts) > 0 {
		h.posts = posts[0]
	}
	return h
}

// UploadMedia godoc
// @Summary      Загрузить изображение
// @Description  Принимает multipart-файл (jpeg/png/webp/gif, до 10 МБ) и поле "kind":
// @Description    - "hero"        — сущность, требует "slug", сохраняется как entities/<storage_key>/images/hero.webp
// @Description    - "thumbnail"   — сущность, требует "slug", сохраняется как entities/<storage_key>/images/thumbnail.webp
// @Description    - "gallery"     — сущность, требует "slug", сохраняется как entities/<storage_key>/images/gallery/<storage_key><NN>.webp
// @Description                      (номер вычисляется автоматически)
// @Description    - "cover"       — обложка статьи, требует "slug" (+ опц. "lang" ru/en),
// @Description                      добавляет следующий кадр в articles/<slug>/images/cover/[<lang>/]<NN>.webp
// @Description    - "block_image" — картинка в блоке статьи, требует "slug" и "block_index"
// @Description                      (+ опц. "lang"), сохраняется как articles/<slug>/images/content/image-<NN>.webp
// @Description                      или image-<NN>-<lang>.webp
// @Description    - "block_pair"  — картинка в блоке-паре статьи, требует "slug", "block_index"
// @Description                      и "pair_index" (1|2) (+ опц. "lang"), сохраняется как
// @Description                      articles/<slug>/image-<NN>-<pair_index>.webp или с суффиксом языка
// @Description  Все изображения всегда конвертируются в WebP независимо от исходного формата.
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Param        file         formData  file    true   "Файл изображения"
// @Param        kind         formData  string  true   "hero | thumbnail | gallery | cover | block_image | block_pair | article_slide | author_avatar"
// @Param        slug         formData  string  true   "Slug минерала или статьи"
// @Param        storage_key  formData  string  false  "Ключ хранилища сущности (если отличается от slug)"
// @Param        slide_id     formData  string  false  "UUID слайда (для article_slide)"
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

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+(1<<20))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Файл не передан (ожидается поле form-data \"file\")")
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}

	if fileHeader.Size > maxUploadSize {
		RespondWithError(c, http.StatusBadRequest, "Файл слишком большой (максимум 10 МБ)")
		return
	}

	kind := c.PostForm("kind")
	contentType := fileHeader.Header.Get("Content-Type")
	if kind != "article_slide" && kind != "author_avatar" && !allowedImageTypes[contentType] {
		RespondWithError(c, http.StatusBadRequest, "Недопустимый тип файла: разрешены jpeg, png, webp, gif")
		return
	}

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
		h.uploadMineralImage(c, kind, slug, strings.TrimSpace(c.PostForm("storage_key")), file, fileHeader)
	case "cover", "block_image", "block_pair":
		h.uploadArticleImage(c, kind, slug, lang, file, fileHeader)
	case "article_slide", "author_avatar":
		h.uploadArticleV2Image(c, kind, slug, strings.TrimSpace(c.PostForm("slide_id")), file)
	default:
		RespondWithError(c, http.StatusBadRequest, "Неизвестное значение поля \"kind\"")
	}
}

func (h *MediaHandler) uploadArticleV2Image(c *gin.Context, kind, slug, slideID string, file io.Reader) {
	if !slugPattern.MatchString(slug) || len(slug) > 255 {
		NewErrorResponse(c, http.StatusBadRequest, "validation_error", "invalid slug")
		return
	}
	if kind == "article_slide" && !postUUIDPattern.MatchString(slideID) {
		NewErrorResponse(c, http.StatusBadRequest, "validation_error", "invalid slide_id")
		return
	}
	select {
	case articleConversionSlots <- struct{}{}:
		defer func() { <-articleConversionSlots }()
	case <-c.Request.Context().Done():
		return
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		RespondInternalError(c, "Не удалось прочитать файл")
		return
	}
	if len(raw) > maxUploadSize {
		NewErrorResponse(c, http.StatusBadRequest, "validation_error", "file exceeds 10 MB")
		return
	}
	prepared, w, height, err := imaging.NormalizeArticleInput(raw)
	if err != nil {
		NewErrorResponse(c, http.StatusBadRequest, "validation_error", "invalid image")
		return
	}
	webpBytes, err := imaging.ToWebP(c.Request.Context(), prepared, webpQuality)
	if err != nil {
		if errors.Is(err, imaging.ErrConverterUnavailable) {
			NewErrorResponse(c, http.StatusServiceUnavailable, "media_conversion_unavailable", "WebP conversion is unavailable")
			return
		}
		NewErrorResponse(c, http.StatusBadRequest, "validation_error", "image cannot be decoded or converted")
		return
	}
	outW, outH, err := imaging.WebPDimensions(webpBytes)
	if err != nil {
		NewErrorResponse(c, http.StatusServiceUnavailable, "media_conversion_unavailable", "Converted WebP is invalid")
		return
	}
	if len(webpBytes) > maxVerifiedWebPSize {
		NewErrorResponse(c, http.StatusBadRequest, "validation_error", "converted image is too large")
		return
	}
	if outW != w || outH != height {
		NewErrorResponse(c, http.StatusBadRequest, "invalid_image_dimensions", "converted image dimensions differ from input")
		return
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		RespondInternalError(c, "Не удалось создать имя файла")
		return
	}
	var key string
	if kind == "article_slide" {
		key, _ = storage.ArticleContentKey(slug, fmt.Sprintf("slides/%s/%s.webp", slideID, hex.EncodeToString(random)))
	} else {
		key, _ = storage.ArticleContentKey(slug, fmt.Sprintf("authors/%s.webp", hex.EncodeToString(random)))
	}
	articleStorage, ok := h.storage.(storage.ArticleMediaStorage)
	if !ok {
		NewErrorResponse(c, http.StatusServiceUnavailable, "media_verification_unavailable", "Article media storage is not configured")
		return
	}
	url, err := articleStorage.UploadUnique(c.Request.Context(), key, "image/webp", bytes.NewReader(webpBytes), int64(len(webpBytes)))
	if err != nil {
		slog.Error("Failed to upload Article V2 media", "error", err, "key", key)
		RespondInternalError(c, "Не удалось загрузить файл")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"url": url})
}

// uploadArticleImage реализует структуру бакета для статей:
//
//	articles/<slug>/images/cover/[<lang>/]<NN>.webp — kind=cover
//	articles/<slug>/images/content/image-<NN>.webp — kind=block_image
//	articles/<slug>/images/content/image-<NN>-<P>-<lang>.webp — kind=block_pair
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
		articleStorage, ok := h.storage.(storage.ArticleMediaStorage)
		if !ok {
			RespondWithError(c, http.StatusServiceUnavailable, "Хранилище кадров обложки не настроено")
			return
		}
		prefix, _ := storage.ArticleCoverPrefix(slug, lang)
		for attempt := 0; attempt < 32; attempt++ {
			keys, err := h.storage.ListKeys(c.Request.Context(), prefix)
			if err != nil {
				RespondInternalError(c, "Не удалось получить кадры обложки")
				return
			}
			index := nextNumber(keys, prefix, galleryNamePattern)
			key, _ = storage.ArticleCoverKey(slug, lang, index)
			url, err := articleStorage.UploadUnique(c.Request.Context(), key, "image/webp", bytes.NewReader(webpBytes), int64(len(webpBytes)))
			if errors.Is(err, storage.ErrObjectExists) {
				continue
			}
			if err != nil {
				RespondInternalError(c, "Не удалось загрузить кадр обложки")
				return
			}
			c.JSON(http.StatusCreated, gin.H{"url": url, "index": index})
			return
		}
		RespondWithError(c, http.StatusConflict, "Одновременные загрузки кадров конфликтуют; повторите запрос")
		return
	case "block_image":
		key, _ = storage.ArticleContentKey(slug, fmt.Sprintf("image-%02d%s.webp", blockIndex, langSuffix))
	case "block_pair":
		key, _ = storage.ArticleContentKey(slug, fmt.Sprintf("image-%02d-%d%s.webp", blockIndex, pairIndex, langSuffix))
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
//	entities/<storage_key>/images/hero.webp
//	entities/<storage_key>/images/thumbnail.webp
//	entities/<storage_key>/images/gallery/<storage_key><NN>.webp
//
// Изображение всегда конвертируется в WebP независимо от исходного формата.
func (h *MediaHandler) uploadMineralImage(c *gin.Context, kind, slug, storageKey string, file io.Reader, fileHeader *multipart.FileHeader) {
	if slug == "" {
		RespondWithError(c, http.StatusBadRequest, "Для изображений минерала обязательно поле \"slug\" — сначала укажите slug в форме минерала")
		return
	}
	if !slugPattern.MatchString(slug) {
		RespondWithError(c, http.StatusBadRequest, "Некорректный slug: разрешены строчные латинские буквы, цифры и дефис")
		return
	}
	// Existing callers can omit storage_key when it equals slug. For V2 entities
	// with a different folder, the caller supplies the persisted storage_key.
	if storageKey == "" {
		storageKey = slug
	}
	if _, err := storage.EntityImageKey(storageKey, "hero.webp"); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Некорректный storage_key")
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
		key, _ = storage.EntityImageKey(storageKey, "hero.webp")
	case "thumbnail":
		key, _ = storage.EntityImageKey(storageKey, "thumbnail.webp")
	case "gallery":
		unique, ok := h.storage.(storage.ArticleMediaStorage)
		if !ok {
			RespondWithError(c, http.StatusServiceUnavailable, "Безопасная загрузка галереи не настроена")
			return
		}
		for attempt := 0; attempt < 32; attempt++ {
			var err error
			key, err = h.nextGalleryKey(c, storageKey)
			if err != nil {
				RespondInternalError(c, "Не удалось определить имя файла для галереи")
				return
			}
			url, err := unique.UploadUnique(c.Request.Context(), key, "image/webp", bytes.NewReader(webpBytes), int64(len(webpBytes)))
			if errors.Is(err, storage.ErrObjectExists) {
				continue
			}
			if err != nil {
				RespondInternalError(c, "Не удалось загрузить файл")
				return
			}
			c.JSON(http.StatusCreated, gin.H{"url": url})
			return
		}
		RespondWithError(c, http.StatusConflict, "Одновременные загрузки галереи конфликтуют; повторите запрос")
		return
	}

	url, err := h.storage.UploadAt(c.Request.Context(), key, "image/webp", bytes.NewReader(webpBytes), int64(len(webpBytes)))
	if err != nil {
		slog.Error("Failed to upload mineral media", "error", err, "key", key)
		RespondInternalError(c, "Не удалось загрузить файл")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": url})
}

// nextGalleryKey вычисляет следующий свободный ключ галереи сущности,
// глядя на уже загруженные файлы в этой папке. Нумерация всегда двузначная (01, 02, ...),
// расширяется до трёх цифр сама, если файлов набирается больше 99.
func (h *MediaHandler) nextGalleryKey(c *gin.Context, storageKey string) (string, error) {
	prefix, err := storage.EntityGalleryPrefix(storageKey)
	if err != nil {
		return "", err
	}

	keys, err := h.storage.ListKeys(c.Request.Context(), prefix)
	if err != nil {
		return "", err
	}

	next := nextNumber(keys, prefix+storageKey, regexp.MustCompile(`^(\d+)\.webp$`))
	numStr := strconv.Itoa(next)
	if next < 100 {
		numStr = fmt.Sprintf("%02d", next)
	}

	return prefix + storageKey + numStr + ".webp", nil
}

func nextNumber(keys []string, prefix string, pattern *regexp.Regexp) int {
	maxIndex := 0
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		match := pattern.FindStringSubmatch(strings.TrimPrefix(key, prefix))
		if len(match) != 2 {
			continue
		}
		n, err := strconv.Atoi(match[1])
		if err == nil && n > maxIndex {
			maxIndex = n
		}
	}
	return maxIndex + 1
}

// ListArticleCover reads stored URL sequences, retaining S3 listing only for
// articles that have never saved cover_frames.
func (h *MediaHandler) ListArticleCover(c *gin.Context) {
	slug, lang := c.Param("slug"), c.Query("lang")
	if !slugPattern.MatchString(slug) || !allowedLangs[lang] || (c.Query("all") != "" && c.Query("all") != "1") {
		RespondWithError(c, http.StatusBadRequest, "Некорректный slug или lang")
		return
	}
	var sequences *coverSequences
	if h.posts != nil {
		admin, ok := NewPostHandler(h.posts).adminRead(c)
		if !ok {
			return
		}
		post, err := h.posts.GetBySlug(c.Request.Context(), slug, admin)
		if err != nil {
			respondPostError(c, err)
			return
		}
		sequences, err = articleCoverFrames(post)
		if err != nil {
			RespondInternalError(c, "Не удалось прочитать кадры статьи")
			return
		}
	}
	managed := sequences != nil
	if !managed {
		s, ok := h.storage.(storage.ArticleMediaStorage)
		if !ok {
			RespondWithError(c, http.StatusServiceUnavailable, "Хранилище кадров не настроено")
			return
		}
		sequences = &coverSequences{}
		var err error
		if sequences.Default, err = legacyCoverURLs(c, s, slug, ""); err != nil {
			RespondInternalError(c, "Не удалось получить кадры обложки")
			return
		}
		if lang == "ru" || c.Query("all") == "1" {
			if sequences.Ru, err = legacyCoverURLs(c, s, slug, "ru"); err != nil {
				RespondInternalError(c, "Не удалось получить кадры обложки")
				return
			}
		}
		if lang == "en" || c.Query("all") == "1" {
			if sequences.En, err = legacyCoverURLs(c, s, slug, "en"); err != nil {
				RespondInternalError(c, "Не удалось получить кадры обложки")
				return
			}
		}
	}
	type frame struct {
		Index int    `json:"index"`
		URL   string `json:"url"`
	}
	urls := sequences.resolved(lang)
	frames := make([]frame, 0, len(urls))
	for i, url := range urls {
		frames = append(frames, frame{i + 1, url})
	}
	response := gin.H{"slug": slug, "lang": lang, "frames": frames}
	if c.Query("all") == "1" {
		response["sequences"] = gin.H{"default": nonNilURLs(sequences.Default), "ru": nonNilURLs(sequences.Ru), "en": nonNilURLs(sequences.En)}
		response["managed"] = managed
	}
	c.JSON(http.StatusOK, response)
}

func nonNilURLs(urls []string) []string {
	if urls == nil {
		return []string{}
	}
	return urls
}

func legacyCoverURLs(c *gin.Context, s storage.ArticleMediaStorage, slug, lang string) ([]string, error) {
	prefix, err := storage.ArticleCoverPrefix(slug, lang)
	if err != nil {
		return nil, err
	}
	keys, err := s.ListKeys(c.Request.Context(), prefix)
	if err != nil {
		return nil, err
	}
	type frame struct {
		Index int
		URL   string
	}
	frames := make([]frame, 0)
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		match := galleryNamePattern.FindStringSubmatch(strings.TrimPrefix(key, prefix))
		if len(match) != 2 {
			continue
		}
		n, err := strconv.Atoi(match[1])
		if err != nil || n < 1 {
			continue
		}
		frames = append(frames, frame{n, s.PublicURL(key)})
	}
	sort.Slice(frames, func(i, j int) bool { return frames[i].Index < frames[j].Index })
	urls := make([]string, 0, len(frames))
	for _, item := range frames {
		urls = append(urls, item.URL)
	}
	return urls, nil
}
