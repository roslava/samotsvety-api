package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/roslava/samotsvety-api/internal/domain"
)

var (
	postSlugPattern                 = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	postUUIDPattern                 = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	ErrMediaVerificationUnavailable = errors.New("article media verification is unavailable")
)

type SlideMediaVerifier interface {
	// A successful result must prove that this exact object was accepted by the
	// uploader and passed its 16:9 check. A matching URL host is not proof.
	VerifySlide(context.Context, string, domain.PostSlide) error
	VerifyAvatar(context.Context, string, string) error
}

type PostFieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type PostValidationError struct{ Fields []PostFieldError }

func (e *PostValidationError) Error() string { return "article validation failed" }
func fieldError(path, message string) error {
	return &PostValidationError{Fields: []PostFieldError{{Path: path, Message: message}}}
}

type postJSONObject map[string]json.RawMessage

func parseJSONObject(raw json.RawMessage, path string) (postJSONObject, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fieldError(path, "object is required")
	}
	var obj postJSONObject
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, fieldError(path, "object is required")
	}
	return obj, nil
}

func stringValue(raw json.RawMessage, path string) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", fieldError(path, "string is required")
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fieldError(path, "string is required")
	}
	return value, nil
}

func boolValue(raw json.RawMessage, path string) (bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return false, fieldError(path, "boolean is required")
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fieldError(path, "boolean is required")
	}
	return value, nil
}

func stringArrayValue(raw json.RawMessage, path string) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fieldError(path, "array is required")
	}
	var value []string
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, fieldError(path, "array of strings is required")
	}
	for i, item := range value {
		if strings.TrimSpace(item) == "" {
			return nil, fieldError(fmt.Sprintf("%s[%d]", path, i), "must not be empty")
		}
	}
	return value, nil
}

func forbidManagedMediaKeys(media postJSONObject) error {
	if _, ok := media["schema_version"]; ok {
		return fieldError("i18n.media.schema_version", "server-managed field")
	}
	if _, ok := media["revision"]; ok {
		return fieldError("i18n.media.revision", "server-managed field")
	}
	return nil
}

func prepareNewI18n(raw json.RawMessage) (json.RawMessage, error) {
	root, err := parseJSONObject(raw, "i18n")
	if err != nil {
		return nil, err
	}
	for _, lang := range []string{"ru", "en"} {
		if _, err := parseJSONObject(root[lang], "i18n."+lang); err != nil {
			return nil, err
		}
	}
	mediaRaw := root["media"]
	if len(mediaRaw) == 0 {
		mediaRaw = json.RawMessage(`{}`)
	}
	media, err := parseJSONObject(mediaRaw, "i18n.media")
	if err != nil {
		return nil, err
	}
	if err := forbidManagedMediaKeys(media); err != nil {
		return nil, err
	}
	if _, ok := media["slides"]; !ok {
		media["slides"] = json.RawMessage(`[]`)
	}
	media["schema_version"] = json.RawMessage(`2`)
	root["media"], err = json.Marshal(media)
	if err != nil {
		return nil, err
	}
	return json.Marshal(root)
}

func mergeJSONObjects(existing, patch json.RawMessage, path string) (json.RawMessage, error) {
	base, err := parseJSONObject(existing, path)
	if err != nil {
		return nil, err
	}
	incoming, err := parseJSONObject(patch, path)
	if err != nil {
		return nil, err
	}
	for key, value := range incoming {
		childPath := path + "." + key
		if string(value) == "null" {
			return nil, fieldError(childPath, "null is not allowed")
		}
		if key == "slides" {
			var slides []json.RawMessage
			if err := json.Unmarshal(value, &slides); err != nil || slides == nil {
				return nil, fieldError(childPath, "array is required")
			}
			base[key] = value
			continue
		}
		if key == "ru" || key == "en" || key == "media" || key == "caption" || key == "alt" {
			current := base[key]
			if len(current) == 0 {
				current = json.RawMessage(`{}`)
			}
			merged, err := mergeJSONObjects(current, value, childPath)
			if err != nil {
				return nil, err
			}
			base[key] = merged
			continue
		}
		// Forward-compatible unknown object keys are recursively merged too.
		baseValue := bytes.TrimSpace(base[key])
		patchValue := bytes.TrimSpace(value)
		if len(baseValue) > 0 && len(patchValue) > 0 && patchValue[0] == '{' && baseValue[0] == '{' {
			merged, err := mergeJSONObjects(base[key], value, childPath)
			if err != nil {
				return nil, err
			}
			base[key] = merged
		} else {
			base[key] = value
		}
	}
	return json.Marshal(base)
}

func mergePostI18n(existing, patch json.RawMessage) (json.RawMessage, error) {
	incoming, err := parseJSONObject(patch, "i18n")
	if err != nil {
		return nil, err
	}
	if mediaRaw, ok := incoming["media"]; ok {
		media, err := parseJSONObject(mediaRaw, "i18n.media")
		if err != nil {
			return nil, err
		}
		if err := forbidManagedMediaKeys(media); err != nil {
			return nil, err
		}
	}
	return mergeJSONObjects(existing, patch, "i18n")
}

func validHTTPSURL(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}

func validateArticle(ctx context.Context, post *domain.Post, verifier SlideMediaVerifier) error {
	issues := make([]PostFieldError, 0)
	add := func(path, message string) { issues = append(issues, PostFieldError{Path: path, Message: message}) }
	if !postSlugPattern.MatchString(post.Slug) || len(post.Slug) > 255 {
		add("slug", "invalid slug")
	}
	switch post.Type {
	case domain.PostTypeBlog, domain.PostTypeGuide, domain.PostTypeHistory, domain.PostTypeEsoteric, domain.PostTypeReview:
	default:
		add("type", "invalid type")
	}
	root, err := parseJSONObject(post.I18n, "i18n")
	if err != nil {
		return err
	}
	for _, lang := range []string{"ru", "en"} {
		obj, err := parseJSONObject(root[lang], "i18n."+lang)
		if err != nil {
			return err
		}
		if _, legacyCover := obj["cover_image"]; legacyCover {
			add("i18n."+lang+".cover_image", "not part of Article V2")
		}
		for _, name := range []string{"title", "excerpt", "content"} {
			if raw, ok := obj[name]; ok {
				value, err := stringValue(raw, "i18n."+lang+"."+name)
				if err != nil {
					return err
				}
				if post.IsPublished && strings.TrimSpace(value) == "" {
					add("i18n."+lang+"."+name, "required for publication")
				}
			} else if post.IsPublished {
				add("i18n."+lang+"."+name, "required for publication")
			}
		}
	}
	media, err := parseJSONObject(root["media"], "i18n.media")
	if err != nil {
		return err
	}
	var schemaVersion int
	if json.Unmarshal(media["schema_version"], &schemaVersion) != nil || schemaVersion != 2 {
		add("i18n.media.schema_version", "must be 2")
	}
	avatarURL := ""
	if avatar, ok := media["author_avatar_url"]; ok {
		value, err := stringValue(avatar, "i18n.media.author_avatar_url")
		if err != nil {
			return err
		}
		if value != "" && !validHTTPSURL(value) {
			add("i18n.media.author_avatar_url", "invalid URL")
		}
		avatarURL = value
	}
	var slides []json.RawMessage
	if err := json.Unmarshal(media["slides"], &slides); err != nil || slides == nil {
		return fieldError("i18n.media.slides", "array is required")
	}
	if post.IsPublished && len(slides) == 0 {
		add("i18n.media.slides", "at least one slide is required")
	}
	seen := map[string]bool{}
	verified := make([]domain.PostSlide, 0, len(slides))
	for i, slideRaw := range slides {
		path := fmt.Sprintf("i18n.media.slides[%d]", i)
		slide, err := parseJSONObject(slideRaw, path)
		if err != nil {
			return err
		}
		id, err := stringValue(slide["id"], path+".id")
		if err != nil {
			return err
		}
		if !postUUIDPattern.MatchString(id) {
			add(path+".id", "invalid UUID")
		}
		if seen[id] {
			add(path+".id", "duplicate UUID")
		}
		seen[id] = true
		urlText, err := stringValue(slide["url"], path+".url")
		if err != nil {
			return err
		}
		if !validHTTPSURL(urlText) {
			add(path+".url", "invalid HTTPS URL")
		}
		item := domain.PostSlide{ID: id, URL: urlText}
		for _, field := range []string{"caption", "alt"} {
			raw, ok := slide[field]
			if !ok {
				if field == "alt" && post.IsPublished {
					add(path+".alt.ru", "required for publication")
					add(path+".alt.en", "required for publication")
				}
				continue
			}
			localized, err := parseJSONObject(raw, path+"."+field)
			if err != nil {
				return err
			}
			for _, lang := range []string{"ru", "en"} {
				value := ""
				if raw, ok := localized[lang]; ok {
					value, err = stringValue(raw, path+"."+field+"."+lang)
					if err != nil {
						return err
					}
				}
				if field == "alt" {
					if post.IsPublished && strings.TrimSpace(value) == "" {
						add(path+".alt."+lang, "required for publication")
					}
					if lang == "ru" {
						item.Alt.Ru = value
					} else {
						item.Alt.En = value
					}
				} else if lang == "ru" {
					item.Caption.Ru = value
				} else {
					item.Caption.En = value
				}
			}
		}
		verified = append(verified, item)
	}
	if post.IsPublished && strings.TrimSpace(post.Author) == "" {
		add("author", "required for publication")
	}
	if len(issues) > 0 {
		return &PostValidationError{Fields: issues}
	}
	if avatarURL != "" {
		if verifier == nil {
			return ErrMediaVerificationUnavailable
		}
		if err := verifier.VerifyAvatar(ctx, post.Slug, avatarURL); err != nil {
			if errors.Is(err, ErrMediaVerificationUnavailable) {
				return err
			}
			return fieldError("i18n.media.author_avatar_url", "media file is not verified")
		}
	}
	if len(verified) > 0 {
		if verifier == nil {
			return ErrMediaVerificationUnavailable
		}
		for i, slide := range verified {
			if err := verifier.VerifySlide(ctx, post.Slug, slide); err != nil {
				if errors.Is(err, ErrMediaVerificationUnavailable) {
					return err
				}
				return fieldError(fmt.Sprintf("i18n.media.slides[%d].url", i), "media file has not been verified as 16:9")
			}
		}
	}
	return nil
}
