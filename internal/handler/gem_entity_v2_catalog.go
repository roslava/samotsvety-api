package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/roslava/samotsvety-api/internal/repository"
	"net/http"
	"strconv"
	"strings"
)

func (h *GemEntityV2Handler) listPage(c *gin.Context) {
	q := repository.V2CatalogQuery{Page: 1, Limit: 20, Lang: c.DefaultQuery("lang", "ru"), Query: strings.TrimSpace(c.Query("q")), Rarity: c.Query("rarity"), BaseColor: c.Query("base_color"), CountryCode: c.Query("country_code"), Letter: c.Query("letter")}
	for _, field := range []struct {
		name   string
		target *int
		max    int
	}{{"page", &q.Page, 1000000}, {"limit", &q.Limit, 100}} {
		if value, present := c.GetQuery(field.name); present {
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 1 || parsed > field.max {
				respondV2Error(c, &v2PayloadError{Path: field.name, Message: "must be a positive integer within the supported range"})
				return
			}
			*field.target = parsed
		}
	}
	if q.Lang != "ru" && q.Lang != "en" {
		respondV2Error(c, &v2PayloadError{Path: "lang", Message: "must be ru or en"})
		return
	}
	if q.CountryCode != "" && !countryCodeQueryPattern.MatchString(q.CountryCode) {
		respondV2Error(c, &v2PayloadError{Path: "country_code", Message: "must be two uppercase ASCII letters"})
		return
	}
	if len([]rune(q.Letter)) > 1 {
		respondV2Error(c, &v2PayloadError{Path: "letter", Message: "must be one letter"})
		return
	}
	repo, ok := h.repo.(repository.V2CatalogRepository)
	if !ok {
		RespondInternalError(c, "Catalog pagination is unavailable")
		return
	}
	result, err := repo.QueryV2(c.Request.Context(), q)
	if err != nil {
		RespondInternalError(c, "Failed to fetch catalog page")
		return
	}
	c.JSON(http.StatusOK, result)
}
