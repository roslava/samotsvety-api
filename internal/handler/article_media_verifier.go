package handler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/imaging"
	"github.com/roslava/samotsvety-api/internal/storage"
)

var ErrInvalidArticleMedia = errors.New("invalid article media")
var ErrInvalidArticleMediaURL = errors.New("invalid article media URL")
var ErrArticleMediaMissing = errors.New("article media object not found")
var immutableName = regexp.MustCompile(`^[0-9a-f]{32}\.webp$`)

type ArticleMediaVerifier struct{ storage storage.ArticleMediaStorage }

func NewArticleMediaVerifier(s storage.ArticleMediaStorage) *ArticleMediaVerifier {
	return &ArticleMediaVerifier{storage: s}
}

func (v *ArticleMediaVerifier) VerifySlide(ctx context.Context, slug string, slide domain.PostSlide) error {
	if v == nil || v.storage == nil {
		return ErrMediaVerificationUnavailable
	}
	if !postSlugPattern.MatchString(slug) || !postUUIDPattern.MatchString(slide.ID) {
		return ErrInvalidArticleMedia
	}
	key, err := v.storage.ArticleKey(slide.URL)
	if err != nil {
		return ErrInvalidArticleMediaURL
	}
	if slide.Source == "cdn" {
		if !supportedArticleImageKey(key) {
			return ErrInvalidArticleMedia
		}
	} else {
		prefix := fmt.Sprintf("articles/%s/images/content/slides/%s/", slug, slide.ID)
		legacyPrefix := fmt.Sprintf("articles/%s/slides/%s/", slug, slide.ID)
		if !(strings.HasPrefix(key, prefix) && immutableName.MatchString(strings.TrimPrefix(key, prefix))) &&
			!(strings.HasPrefix(key, legacyPrefix) && immutableName.MatchString(strings.TrimPrefix(key, legacyPrefix))) {
			return ErrInvalidArticleMedia
		}
	}
	data, err := v.storage.ReadObject(ctx, key, maxVerifiedWebPSize)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return ErrArticleMediaMissing
		}
		if errors.Is(err, storage.ErrObjectTooLarge) {
			return ErrInvalidArticleMedia
		}
		return fmt.Errorf("%w: %w", ErrMediaVerificationUnavailable, err)
	}
	if _, _, _, err := imaging.ArticleInputDimensions(data); err != nil {
		return ErrInvalidArticleMedia
	}
	return nil
}

func (v *ArticleMediaVerifier) VerifyAvatar(ctx context.Context, slug, rawURL string) error {
	if v == nil || v.storage == nil {
		return ErrMediaVerificationUnavailable
	}
	if !postSlugPattern.MatchString(slug) {
		return ErrInvalidArticleMedia
	}
	key, err := v.storage.ArticleKey(rawURL)
	if err != nil {
		return ErrInvalidArticleMediaURL
	}
	// The article stores a reference to an existing image in this bucket. The
	// object may live anywhere in it, including after an article slug change.
	if !supportedArticleImageKey(key) {
		return ErrInvalidArticleMedia
	}
	data, err := v.storage.ReadObject(ctx, key, maxVerifiedWebPSize)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return ErrArticleMediaMissing
		}
		if errors.Is(err, storage.ErrObjectTooLarge) {
			return ErrInvalidArticleMedia
		}
		return fmt.Errorf("%w: %w", ErrMediaVerificationUnavailable, err)
	}
	if _, _, _, err := imaging.ArticleInputDimensions(data); err != nil {
		return ErrInvalidArticleMedia
	}
	return nil
}

const maxVerifiedWebPSize = 20 << 20
