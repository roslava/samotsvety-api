package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/repository"
)

type MineralHandler struct {
	repo repository.MineralRepository
}

func NewMineralHandler(repo repository.MineralRepository) *MineralHandler {
	return &MineralHandler{repo: repo}
}

// ListMinerals godoc
// @Summary      Получить список минералов
// @Description  Возвращает список минералов с фильтрацией и пагинацией
// @Tags         minerals
// @Accept       json
// @Produce      json
// @Param        lang         query     string  false  "Язык (ru/en)"            default(ru)
// @Param        view         query     string  false  "Режим (normal/esoteric)" default(normal)
// @Param        rarity       query     string  false  "Редкость"
// @Param        hardness_min query     number  false  "Минимальная твёрдость"
// @Param        hardness_max query     number  false  "Максимальная твёрдость"
// @Param        russian_only query     bool    false  "Только российские"
// @Param        limit        query     int     false  "Лимит"                   default(20)
// @Param        page         query     int     false  "Страница"                default(1)
// @Success      200          {object}  ListResponse
// @Router       /api/v1/minerals [get]
func (h *MineralHandler) ListMinerals(c *gin.Context) {
	var filters domain.FilterParams
	if err := c.ShouldBindQuery(&filters); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	if filters.Limit == 0 {
		filters.Limit = 20
	}
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Lang == "" {
		filters.Lang = "ru"
	}
	if filters.View == "" {
		filters.View = "normal"
	}
	if filters.Order == "" {
		filters.Order = "desc"
	}

	minerals, total, err := h.repo.List(c.Request.Context(), filters)
	if err != nil {
		slog.Error("Failed to list minerals", "error", err)
		RespondInternalError(c, "Failed to fetch minerals")
		return
	}

	if minerals == nil {
		minerals = []domain.Mineral{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  minerals,
		"total": total,
		"page":  filters.Page,
		"limit": filters.Limit,
	})
}

// GetMineral godoc
// @Summary      Получить минерал по slug
// @Description  Возвращает полную карточку минерала
// @Tags         minerals
// @Accept       json
// @Produce      json
// @Param        slug path      string true  "Slug минерала"
// @Param        lang query     string false "Язык (ru/en)"            default(ru)
// @Param        view query     string false "Режим (normal/esoteric)" default(normal)
// @Success      200  {object}  domain.Mineral
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/minerals/{slug} [get]
func (h *MineralHandler) GetMineral(c *gin.Context) {
	slug := c.Param("slug")
	lang := c.DefaultQuery("lang", "ru")
	view := c.DefaultQuery("view", "normal")

	mineral, err := h.repo.GetBySlug(c.Request.Context(), slug, lang, view)
	if err != nil {
		RespondNotFound(c, "Mineral not found")
		return
	}

	c.JSON(http.StatusOK, mineral)
}

// SearchMinerals godoc
// @Summary      Поиск минералов
// @Description  Полнотекстовый поиск по названию, синонимам, лору и химической формуле
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        q     query  string true   "Поисковый запрос"
// @Param        lang  query  string false  "Язык (ru/en)"            default(ru)
// @Param        view  query  string false  "Режим (normal/esoteric)" default(normal)
// @Param        limit query  int    false  "Лимит"                   default(20)
// @Param        page  query  int    false  "Страница"                default(1)
// @Success      200   {object} ListResponse
// @Router       /api/v1/search [get]
func (h *MineralHandler) SearchMinerals(c *gin.Context) {
	q := c.Query("q")
	lang := c.DefaultQuery("lang", "ru")
	view := c.DefaultQuery("view", "normal")

	limitStr := c.DefaultQuery("limit", "20")
	pageStr := c.DefaultQuery("page", "1")

	limit, _ := strconv.Atoi(limitStr)
	page, _ := strconv.Atoi(pageStr)

	if limit == 0 {
		limit = 20
	}
	if page == 0 {
		page = 1
	}

	offset := (page - 1) * limit

	results, total, err := h.repo.Search(c.Request.Context(), q, lang, view, limit, offset)
	if err != nil {
		slog.Error("Search failed", "error", err)
		RespondInternalError(c, "Search error")
		return
	}

	if results == nil {
		results = []domain.Mineral{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetFilters godoc
// @Summary      Получить доступные значения фильтров
// @Description  Возвращает списки для фильтров фронтенда
// @Tags         filters
// @Produce      json
// @Success      200 {object} repository.FilterValues
// @Router       /api/v1/filters [get]
func (h *MineralHandler) GetFilters(c *gin.Context) {
	filters, err := h.repo.GetFilters(c.Request.Context())
	if err != nil {
		slog.Error("GetFilters failed", "error", err)
		RespondInternalError(c, "Failed to get filters")
		return
	}

	c.JSON(http.StatusOK, filters)
}
