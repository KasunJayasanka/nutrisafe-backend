// controllers/analytics_controller.go
package controllers

import (
	"net/http"
	"time"

	"backend/services"
	"github.com/gin-gonic/gin"
)

type AnalyticsController struct {
	Svc *services.AnalyticsService
}

func NewAnalyticsController(svc *services.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{Svc: svc}
}

func (h *AnalyticsController) GetAnalyticsSummary(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	now := time.Now()
	first := time.Date(now.Year(), now.Month(), 1, 0,0,0,0, now.Location())
	last  := first.AddDate(0,1,-1)

	fromStr := c.DefaultQuery("from", first.Format("2006-01-02"))
	toStr   := c.DefaultQuery("to",   last.Format("2006-01-02"))
	includeMissing := c.DefaultQuery("includeMissingDays","false") == "true"

	from, err := time.ParseInLocation("2006-01-02", fromStr, now.Location()); if err != nil { c.JSON(400, gin.H{"error":"invalid from date"}); return }
	to,   err := time.ParseInLocation("2006-01-02", toStr,   now.Location()); if err != nil { c.JSON(400, gin.H{"error":"invalid to date"});   return }
	if to.Before(from) { c.JSON(400, gin.H{"error":"`to` must be on/after `from`"}); return }

	out, err := h.Svc.Summary(c.Request.Context(), userID, from, to, includeMissing)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, out)
}

func (h *AnalyticsController) GetWeeklyOverview(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	now := time.Now()
	weekStart := startOfWeek(now)
	if v := c.Query("week_start"); v != "" {
		if ws, err := time.ParseInLocation("2006-01-02", v, now.Location()); err == nil {
			weekStart = startOfWeek(ws)
		} else {
			c.JSON(400, gin.H{"error":"invalid week_start"}); return
		}
	}
	mode := c.DefaultQuery("mode","detailed")

	out, err := h.Svc.WeeklyOverview(c.Request.Context(), userID, weekStart, mode)
	if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	c.JSON(200, out)
}

// --- helpers ---

func userIDFromCtx(c *gin.Context) (uint, bool) {
	v, ok := c.Get("userID") // adjust if your middleware uses another key
	if !ok {
		return 0, false
	}
	switch id := v.(type) {
	case uint:
		return id, true
	case int:
		return uint(id), true
	case int64:
		return uint(id), true
	default:
		return 0, false
	}
}

func startOfWeek(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return tt.AddDate(0, 0, -(wd - 1)) // Monday
}