package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/roslava/samotsvety-api/internal/domain"
	"github.com/roslava/samotsvety-api/internal/repository"
)

// ListResponse — стандартизированный ответ для списков
type ListResponse struct {
	Data  []domain.Mineral `json:"data"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
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

// CreateMineral godoc
// @Summary      Создать новый минерал
// @Description  Создаёт новый самоцвет/минерал в базе (админка)
// @Tags         minerals
// @Accept       json
// @Produce      json
// @Param        mineral body handler.CreateMineralRequest true "Данные минерала"
// @Success      201  {object} domain.Mineral
// @Failure      400  {object} handler.ErrorResponse
// @Failure      409  {object} handler.ErrorResponse "Slug уже существует"
// @Failure      500  {object} handler.ErrorResponse
// @Router       /api/v1/minerals [post]
func (h *MineralHandler) CreateMineral(c *gin.Context) {
	var req CreateMineralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	now := time.Now().UTC()

	mineral := &domain.Mineral{
		Slug:            req.Slug,
		Scientific:      req.Scientific,
		I18n:            req.I18n,
		Localities:      req.Localities,
		MainImageURL:    req.MainImageURL,
		Gallery:         req.Gallery,
		SafetyNotes:     req.SafetyNotes,
		RelatedMinerals: req.RelatedMinerals,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.repo.Create(c.Request.Context(), mineral); err != nil {
		if err.Error() == "slug_already_exists" {
			RespondWithError(c, http.StatusConflict, "Минерал с таким slug уже существует")
			return
		}
		slog.Error("Failed to create mineral", "error", err, "slug", req.Slug)
		RespondInternalError(c, "Не удалось создать минерал")
		return
	}

	c.JSON(http.StatusCreated, mineral)
}

// UpdateMineral godoc
// @Summary      Обновить минерал
// @Description  Обновляет данные существующего минерала по slug (админка)
// @Tags         minerals
// @Accept       json
// @Produce      json
// @Param        slug    path    string                      true  "Slug минерала"
// @Param        mineral body    handler.UpdateMineralRequest true  "Обновлённые данные"
// @Success      200     {object} domain.Mineral
// @Failure      400     {object} handler.ErrorResponse
// @Failure      404     {object} handler.ErrorResponse
// @Failure      500     {object} handler.ErrorResponse
// @Router       /api/v1/minerals/{slug} [put]
func (h *MineralHandler) UpdateMineral(c *gin.Context) {
	slug := c.Param("slug")

	var req UpdateMineralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	// Получаем текущий минерал
	mineral, err := h.repo.GetBySlug(c.Request.Context(), slug, "ru", "normal")
	if err != nil {
		RespondNotFound(c, "Минерал не найден")
		return
	}

	// Обновляем только те поля, которые пришли в запросе
	if req.Scientific != nil {
		mineral.Scientific = *req.Scientific
	}
	if req.I18n != nil {
		mineral.I18n = *req.I18n
	}
	if req.Localities != nil {
		mineral.Localities = req.Localities
	}
	if req.MainImageURL != nil {
		mineral.MainImageURL = *req.MainImageURL
	}
	if req.Gallery != nil {
		mineral.Gallery = req.Gallery
	}
	if req.SafetyNotes != nil {
		mineral.SafetyNotes = *req.SafetyNotes
	}
	if req.RelatedMinerals != nil {
		mineral.RelatedMinerals = req.RelatedMinerals
	}

	if err := h.repo.Update(c.Request.Context(), slug, mineral); err != nil {
		if err.Error() == "mineral not found" {
			RespondNotFound(c, "Минерал не найден")
			return
		}
		slog.Error("Failed to update mineral", "error", err, "slug", slug)
		RespondInternalError(c, "Не удалось обновить минерал")
		return
	}

	c.JSON(http.StatusOK, mineral)
}

// DeleteMineral godoc
// @Summary      Удалить минерал
// @Description  Удаляет минерал по slug (админка)
// @Tags         minerals
// @Accept       json
// @Produce      json
// @Param        slug  path  string  true  "Slug минерала"
// @Success      204
// @Failure      404  {object} handler.ErrorResponse
// @Failure      500  {object} handler.ErrorResponse
// @Router       /api/v1/minerals/{slug} [delete]
func (h *MineralHandler) DeleteMineral(c *gin.Context) {
	slug := c.Param("slug")

	if err := h.repo.Delete(c.Request.Context(), slug); err != nil {
		if err.Error() == "mineral not found" {
			RespondNotFound(c, "Минерал не найден")
			return
		}
		slog.Error("Failed to delete mineral", "error", err, "slug", slug)
		RespondInternalError(c, "Не удалось удалить минерал")
		return
	}

	c.Status(http.StatusNoContent)
}
