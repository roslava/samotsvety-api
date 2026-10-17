// internal/storage/media_storage.go
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/roslava/samotsvety-api/internal/config"
)

// MediaStorage — хранилище файлов для иллюстраций статей и медиа минералов.
// Реализация — Yandex Object Storage, который S3-совместим, поэтому используется
// обычный AWS SDK с кастомным endpoint, без отдельного Yandex-специфичного клиента.
type MediaStorage interface {
	// Upload сохраняет файл под СГЕНЕРИРОВАННЫМ уникальным именем внутри folder
	// и возвращает публичный URL. Используется для иллюстраций статей, где
	// строгая структура папок/имён не требуется.
	Upload(ctx context.Context, folder string, filename string, contentType string, data io.Reader, size int64) (string, error)

	// UploadAt сохраняет файл под ТОЧНО заданным ключом (перезаписывая существующий
	// объект, если он там уже есть) и возвращает публичный URL. Используется там, где
	// имя и путь файла — часть контракта (hero.webp, thumbnail.webp, gallery/rubyNN.webp).
	UploadAt(ctx context.Context, key string, contentType string, data io.Reader, size int64) (string, error)

	// ListKeys возвращает ключи объектов бакета с данным префиксом.
	// Используется, например, чтобы определить следующий порядковый номер
	// изображения в галерее минерала (ruby01.webp, ruby02.webp, ...).
	ListKeys(ctx context.Context, prefix string) ([]string, error)
}

// ArticleMediaStorage exposes server-authenticated reads for Article V2.
type ArticleMediaStorage interface {
	MediaStorage
	UploadUnique(ctx context.Context, key, contentType string, data io.Reader, size int64) (string, error)
	PublicURL(key string) string
	ArticleKey(publicURL string) (string, error)
	ReadObject(ctx context.Context, key string, maxBytes int64) ([]byte, error)
}

var ErrInvalidArticleURL = fmt.Errorf("invalid article media URL")
var ErrObjectNotFound = fmt.Errorf("media object not found")
var ErrObjectAccessDenied = fmt.Errorf("media object access denied")
var ErrObjectTooLarge = fmt.Errorf("media object too large")
var ErrObjectExists = fmt.Errorf("media object already exists")

type yandexS3Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// NewYandexS3Storage создаёт клиент для Yandex Object Storage по настройкам из конфига.
func NewYandexS3Storage(ctx context.Context, cfg config.StorageConfig) (MediaStorage, error) {
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("storage: не заданы YC_S3_BUCKET / YC_S3_ACCESS_KEY / YC_S3_SECRET_KEY")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		// Yandex Object Storage работает в virtual-hosted стиле (bucket.storage.yandexcloud.net),
		// но path-style тоже поддерживается и надёжнее для кастомного endpoint.
		o.UsePathStyle = true
	})

	publicURL := cfg.PublicURL
	if publicURL == "" {
		// Дефолт: путь вида https://storage.yandexcloud.net/<bucket>
		publicURL = strings.TrimRight(cfg.Endpoint, "/") + "/" + cfg.Bucket
	}

	return &yandexS3Storage{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(publicURL, "/"),
	}, nil
}

func (s *yandexS3Storage) Upload(ctx context.Context, folder, filename, contentType string, data io.Reader, size int64) (string, error) {
	key := path.Join(folder, uniqueFilename(filename))
	return s.putObject(ctx, key, contentType, data, size)
}

func (s *yandexS3Storage) UploadAt(ctx context.Context, key, contentType string, data io.Reader, size int64) (string, error) {
	key = strings.TrimLeft(key, "/")
	return s.putObject(ctx, key, contentType, data, size)
}

// UploadUnique uses S3's atomic create-if-absent condition. Article uploads
// cannot overwrite an existing object even if a random filename collides.
func (s *yandexS3Storage) UploadUnique(ctx context.Context, key, contentType string, data io.Reader, size int64) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: data,
		ContentType: aws.String(contentType), ContentLength: aws.Int64(size),
		ACL: "public-read", IfNoneMatch: aws.String("*"),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "PreconditionFailed" || apiErr.ErrorCode() == "ConditionalRequestConflict") {
			return "", ErrObjectExists
		}
		return "", fmt.Errorf("storage: upload unique failed: %w", err)
	}
	return s.PublicURL(key), nil
}

func (s *yandexS3Storage) PublicURL(key string) string { return s.publicURL + "/" + key }

func (s *yandexS3Storage) putObject(ctx context.Context, key, contentType string, data io.Reader, size int64) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          data,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
		ACL:           "public-read",
	})
	if err != nil {
		return "", fmt.Errorf("storage: upload failed: %w", err)
	}

	return s.PublicURL(key), nil
}

func (s *yandexS3Storage) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("storage: list objects failed: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}

	return keys, nil
}

// ArticleKey resolves only URLs in the configured bucket. Query strings,
// fragments and unsafe escaped path segments are rejected.
func (s *yandexS3Storage) ArticleKey(raw string) (string, error) {
	// The same Lithoteka bucket has both Yandex path-style and virtual-hosted
	// public URLs. Resolve either spelling to this configured bucket only.
	if s.bucket == "lithoteka-cdn" {
		u, err := url.Parse(raw)
		if err == nil && u.Scheme == "https" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Port() == "" {
			escapedKey := ""
			switch u.Hostname() {
			case "lithoteka-cdn.storage.yandexcloud.net":
				escapedKey = strings.TrimPrefix(u.EscapedPath(), "/")
			case "storage.yandexcloud.net":
				escapedKey = strings.TrimPrefix(u.EscapedPath(), "/lithoteka-cdn/")
				if escapedKey == u.EscapedPath() {
					escapedKey = ""
				}
			}
			parts := strings.Split(escapedKey, "/")
			for i, part := range parts {
				decoded, err := url.PathUnescape(part)
				if err != nil || decoded == "" || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, "/\\") {
					parts = nil
					break
				}
				parts[i] = decoded
			}
			key := strings.Join(parts, "/")
			if safeImagePath(key) && path.Clean(key) == key && raw == "https://"+u.Host+u.EscapedPath() {
				return key, nil
			}
		}
	}
	base, err := url.Parse(s.publicURL)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.RawQuery != "" || base.Fragment != "" || base.User != nil {
		return "", ErrInvalidArticleURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || strings.Contains(raw, "%") {
		return "", ErrInvalidArticleURL
	}
	prefix := strings.TrimRight(base.Path, "/") + "/"
	if !strings.HasPrefix(u.Path, prefix) {
		return "", ErrInvalidArticleURL
	}
	key := strings.TrimPrefix(u.Path, prefix)
	if key == "" || strings.Contains(key, "//") || strings.Contains(key, "\\") || path.Clean(key) != key || strings.ContainsAny(key, "?#") || raw != s.publicURL+"/"+key {
		return "", ErrInvalidArticleURL
	}
	return key, nil
}

func (s *yandexS3Storage) ReadObject(ctx context.Context, key string, maxBytes int64) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound") {
			return nil, ErrObjectNotFound
		}
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "AccessDenied" || apiErr.ErrorCode() == "Forbidden") {
			return nil, fmt.Errorf("storage: read object: %w", ErrObjectAccessDenied)
		}
		return nil, fmt.Errorf("storage: read object: %w", err)
	}
	defer out.Body.Close()
	if out.ContentLength != nil && *out.ContentLength > maxBytes {
		return nil, ErrObjectTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(out.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("storage: read object body: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrObjectTooLarge
	}
	return data, nil
}

// uniqueFilename добавляет к оригинальному имени файла дату и случайный суффикс,
// чтобы избежать коллизий и не хранить в URL ничего, зависящего от локали ОС.
func uniqueFilename(original string) string {
	ext := strings.ToLower(path.Ext(original))
	randBytes := make([]byte, 6)
	_, _ = rand.Read(randBytes)
	return fmt.Sprintf("%s-%s%s", time.Now().UTC().Format("20060102-150405"), hex.EncodeToString(randBytes), ext)
}
