package api

import (
	"net/http"
	"time"

	"github.com/Carlos-Marrugo/pigbank-card-service/internal/models"
	"github.com/Carlos-Marrugo/pigbank-card-service/internal/service"
	"github.com/gin-gonic/gin"
)

type CardHandler struct{}

func (h *CardHandler) ProcessTransaction(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := service.ProcessTransaction(c.Request.Context(), req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Transaction processed successfully",
		"transaction": tx,
	})
}

func (h *CardHandler) GenerateReport(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	from := c.Query("from")
	to := c.Query("to")

	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	url, err := service.GenerateReport(c.Request.Context(), userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Report generated successfully",
		"report_url": url,
	})
}

func (h *CardHandler) GetCards(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	cards, err := service.GetUserCards(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cards": cards})
}

func (h *CardHandler) GetBalance(c *gin.Context) {
	userID := c.GetString("user_id")
	cardID := c.Param("card_id")

	balance, err := service.GetCardBalance(c.Request.Context(), cardID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"card_id": cardID, "balance": balance})
}
