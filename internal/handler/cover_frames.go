package handler

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/roslava/samotsvety-api/internal/domain"
)

type coverSequences struct {
	Default []string `json:"default"`
	Ru      []string `json:"ru"`
	En      []string `json:"en"`
}

// nil means that the article has not adopted the URL list yet. Empty slices
// are intentional and must never trigger the legacy S3 fallback.
func articleCoverFrames(post *domain.Post) (*coverSequences, error) {
	var root struct {
		Media struct {
			CoverFrames json.RawMessage `json:"cover_frames"`
		} `json:"media"`
	}
	if err := json.Unmarshal(post.I18n, &root); err != nil {
		return nil, err
	}
	if len(root.Media.CoverFrames) == 0 {
		return nil, nil
	}
	var sequences coverSequences
	if err := json.Unmarshal(root.Media.CoverFrames, &sequences); err != nil {
		return nil, err
	}
	return &sequences, nil
}

func (frames *coverSequences) forLang(lang string) []string {
	switch lang {
	case "ru":
		return frames.Ru
	case "en":
		return frames.En
	default:
		return frames.Default
	}
}

func (frames *coverSequences) resolved(lang string) []string {
	selected := frames.forLang(lang)
	if lang != "" && len(selected) == 0 {
		return frames.Default
	}
	return selected
}

func validLithotekaImageURL(raw string) bool {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.Contains(raw, "\\") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Port() != "" {
		return false
	}
	var escapedKey string
	switch u.Hostname() {
	case "lithoteka-cdn.storage.yandexcloud.net":
		escapedKey = strings.TrimPrefix(u.EscapedPath(), "/")
	case "storage.yandexcloud.net":
		escapedKey = strings.TrimPrefix(u.EscapedPath(), "/lithoteka-cdn/")
		if escapedKey == u.EscapedPath() {
			return false
		}
	default:
		return false
	}
	if escapedKey == "" || strings.HasPrefix(escapedKey, "/") || strings.Contains(escapedKey, "//") {
		return false
	}
	segments := strings.Split(escapedKey, "/")
	for i, escaped := range segments {
		segment, err := url.PathUnescape(escaped)
		if err != nil || segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, "/\\") || strings.ContainsAny(segment, "\x00\r\n") {
			return false
		}
		segments[i] = segment
	}
	return supportedArticleImageKey(strings.Join(segments, "/"))
}

func supportedArticleImageKey(key string) bool {
	switch strings.ToLower(path.Ext(key)) {
	case ".webp", ".png", ".jpg", ".jpeg", ".gif":
		return true
	}
	return false
}

func validateCoverFrames(raw json.RawMessage) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return fieldError("i18n.media.cover_frames", "object is required")
	}
	if len(obj) != 3 {
		return fieldError("i18n.media.cover_frames", "default, ru and en arrays are required")
	}
	for _, lang := range []string{"default", "ru", "en"} {
		value, ok := obj[lang]
		if !ok {
			return fieldError("i18n.media.cover_frames."+lang, "array is required")
		}
		var urls []string
		if err := json.Unmarshal(value, &urls); err != nil || urls == nil {
			return fieldError("i18n.media.cover_frames."+lang, "array of URLs is required")
		}
		for i, rawURL := range urls {
			if !validLithotekaImageURL(rawURL) {
				return fieldError(fmt.Sprintf("i18n.media.cover_frames.%s[%d]", lang, i), "invalid Lithoteka CDN WebP URL")
			}
		}
	}
	return nil
}
