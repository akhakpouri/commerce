package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) RegisterRoutes(cg *gin.RouterGroup) {
	cg.GET("/live", h.GetLive)
}

// @Summary Health Check
// @Description get the status of the service
// @Tags Health
// @Produce  json
// @Success 200 {object} map[string]string
// @Router /health/status/live [get]
func (h *HealthHandler) GetLive(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}
