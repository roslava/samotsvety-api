package handler

import (
	"log/slog"
	"net/http"

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
