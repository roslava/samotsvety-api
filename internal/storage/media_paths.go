package storage

import (
	"fmt"
	"regexp"
	"strings"
)

var mediaFolderPattern = regexp.MustCompile(`^[a-z0-9]+(?:[a-z0-9_-]*[a-z0-9])?$`)

// EntityImageKey resolves a V2 image path without changing its stored representation.
func EntityImageKey(storageKey, imagePath string) (string, error) {
	if !mediaFolderPattern.MatchString(storageKey) || !safeImagePath(imagePath) || strings.HasPrefix(imagePath, storageKey+"/") || strings.HasPrefix(imagePath, "entities/") || strings.HasPrefix(imagePath, "images/") {
		return "", fmt.Errorf("invalid entity image path")
	}
	return "entities/" + storageKey + "/images/" + imagePath, nil
}

func EntityGalleryPrefix(storageKey string) (string, error) {
	key, err := EntityImageKey(storageKey, "gallery/placeholder.webp")
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(key, "placeholder.webp"), nil
}

func ArticleCoverPrefix(slug, lang string) (string, error) {
	if !mediaFolderPattern.MatchString(slug) || strings.Contains(slug, "_") || (lang != "" && lang != "ru" && lang != "en") {
		return "", fmt.Errorf("invalid article cover path")
	}
	prefix := "articles/" + slug + "/images/cover/"
	if lang != "" {
		prefix += lang + "/"
	}
	return prefix, nil
}

func ArticleCoverKey(slug, lang string, index int) (string, error) {
	prefix, err := ArticleCoverPrefix(slug, lang)
	if err != nil || index < 1 {
		return "", fmt.Errorf("invalid article cover index")
	}
	return fmt.Sprintf("%s%02d.webp", prefix, index), nil
}

func ArticleContentKey(slug, name string) (string, error) {
	if !mediaFolderPattern.MatchString(slug) || strings.Contains(slug, "_") || !safeImagePath(name) || strings.HasPrefix(name, "images/") || strings.HasPrefix(name, "articles/") {
		return "", fmt.Errorf("invalid article content path")
	}
	return "articles/" + slug + "/images/content/" + name, nil
}

func safeImagePath(value string) bool {
	if value == "" || strings.Contains(value, "\\") || strings.ContainsAny(value, "?#%") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
