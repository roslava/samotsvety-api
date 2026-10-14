package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/roslava/samotsvety-api/internal/domain"
)

// PrepareArticleV2Seed uses the same create path as POST /api/v1/posts.
// Seed files cannot supply server-managed fields or bypass media verification.
func PrepareArticleV2Seed(ctx context.Context, data []byte, verifier SlideMediaVerifier) (*domain.Post, error) {
	var payload postJSONObject
	if err := json.Unmarshal(data, &payload); err != nil || payload == nil {
		return nil, describeSeedValidation(fieldError("body", "JSON object is required"))
	}
	if err := checkNewPostPayloadFields(payload); err != nil {
		return nil, describeSeedValidation(err)
	}
	root, err := parseJSONObject(payload["i18n"], "i18n")
	if err != nil {
		return nil, describeSeedValidation(err)
	}
	media, err := parseJSONObject(root["media"], "i18n.media")
	if err != nil {
		return nil, describeSeedValidation(err)
	}
	if err := forbidManagedMediaKeys(media); err != nil {
		return nil, describeSeedValidation(err)
	}
	if _, ok := media["slides"]; !ok {
		return nil, describeSeedValidation(fieldError("i18n.media.slides", "required in Article V2 seed file"))
	}
	post, err := prepareNewPost(ctx, payload, verifier)
	return post, describeSeedValidation(err)
}

func describeSeedValidation(err error) error {
	if err == nil {
		return nil
	}
	var validation *PostValidationError
	if !errors.As(err, &validation) {
		return err
	}
	parts := make([]string, 0, len(validation.Fields))
	for _, field := range validation.Fields {
		parts = append(parts, field.Path+": "+field.Message)
	}
	return fmt.Errorf("Article V2 validation: %s", strings.Join(parts, "; "))
}
