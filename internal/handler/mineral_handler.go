package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/repository"
)

// ListResponse — стандартизированный ответ для списков
type ListResponse struct {
	Data  []domain.Mineral `json:"data" example:"[{\"slug\":\"malachite\"}]"`
	Total int              `json:"total" example:"42"`
	Page  int              `json:"page" example:"1"`
	Limit int              `json:"limit" example:"20"`
}

type MineralHandler struct {
	repo repository.MineralRepository
}

func NewMineralHandler(repo repository.MineralRepository) *MineralHandler {
	return &MineralHandler{repo: repo}
}

// ListMinerals godoc
// @Summary      Получить список минералов
// @Description  Возвращает список самоцветов с поддержкой фильтрации, сортировки и пагинации
// @Tags         minerals
// @Accept       json
// @Produce      json
// @Param        lang         query  string   false  "Язык ответа"                  default=ru  enum(ru,en)
// @Param        view         query  string   false  "Режим отображения"            default=normal enum(normal,esoteric)
// @Param        rarity       query  string   false  "Редкость"
// @Param        mineral_group query string  false  "Группа минерала"
// @Param        color        query  string   false  "Цвет"
// @Param        russian_only query  bool     false  "Только российские"
// @Param        hardness_min query  number   false  "Минимальная твёрдость"
// @Param        hardness_max query  number   false  "Максимальная твёрдость"
// @Param        sort         query  string   false  "Сортировка"                   default=created_at enum(name,rarity,hardness,created_at)
// @Param        order        query  string   false  "Порядок сортировки"           default=desc    enum(asc,desc)
// @Param        limit        query  int      false  "Количество на странице"       default=20
// @Param        page         query  int      false  "Страница"                     default=1
// @Success      200  {object}  ListResponse
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
// @Description  Возвращает полную детальную карточку самоцвета
// @Tags         minerals
// @Accept       json
// @Produce      json
// @Param        slug  path    string  true   "Slug минерала"
// @Param        lang  query   string  false  "Язык"                  default=ru
// @Param        view  query   string  false  "Режим"                 default=normal
// @Success      200   {object}  domain.Mineral
// @Failure      404   {object}  ErrorResponse
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
// @Description  Полнотекстовый поиск по названию, синонимам, lore и химической формуле (работает на ru+en)
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        q     query  string  true   "Поисковый запрос"
// @Param        lang  query  string  false  "Язык ответа"          default=ru
// @Param        view  query  string  false  "Режим"                default=normal
// @Param        limit query  int     false  "Лимит"                default=20
// @Param        page  query  int     false  "Страница"             default=1
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
// @Summary      Получить значения для фильтров
// @Description  Возвращает списки доступных значений для фронтенда
// @Tags         filters
// @Produce      json
// @Success      200  {object} repository.FilterValues
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
