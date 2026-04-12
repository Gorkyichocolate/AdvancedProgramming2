package transport

import (
	"context"
	"errors"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/usecase"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type idempotencyEntry struct {
	status int
	body   any
}

type OrderHandler struct {
	uc          *usecase.OrderUsecase
	idempotency sync.Map
}

func NewOrderHandler(uc *usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name"   binding:"required"`
	Amount     int64  `json:"amount"      binding:"required,min=1"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	key := c.GetHeader("Idempotency-Key")
	if key != "" {
		if cached, ok := h.idempotency.Load(key); ok {
			e := cached.(idempotencyEntry)
			c.JSON(e.status, e.body)
			return
		}
	}

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.uc.CreateOrder(context.Background(), req.CustomerID, req.ItemName, req.Amount)

	var (
		status int
		body   any
	)
	switch {
	case errors.Is(err, usecase.ErrInvalidAmount):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	case errors.Is(err, usecase.ErrPaymentServiceUnavailable):
		status = http.StatusServiceUnavailable
		body = gin.H{"error": "payment service unavailable", "order": order}
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	default:
		status = http.StatusCreated
		body = order
	}

	if key != "" {
		h.idempotency.Store(key, idempotencyEntry{status: status, body: body})
	}
	c.JSON(status, body)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := h.uc.GetOrder(context.Background(), id)
	switch {
	case errors.Is(err, usecase.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	default:
		c.JSON(http.StatusOK, order)
	}
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := h.uc.CancelOrder(context.Background(), id)
	switch {
	case errors.Is(err, usecase.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
	case errors.Is(err, usecase.ErrNotCancellable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	default:
		c.JSON(http.StatusOK, order)
	}
}

func (h *OrderHandler) GetList(c *gin.Context) {
	rawQuery := strings.ReplaceAll(c.Request.URL.RawQuery, "?", "&")
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query string"})
		return
	}

	minAmountRaw := values.Get("min_amount")
	maxAmountRaw := values.Get("max_amount")

	if minAmountRaw == "" || maxAmountRaw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_amount and max_amount are required"})
		return
	}

	minAmount, err := strconv.ParseInt(minAmountRaw, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_amount must be an integer"})
		return
	}

	maxAmount, err := strconv.ParseInt(maxAmountRaw, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_amount must be an integer"})
		return
	}

	orders, err := h.uc.OrderList(context.Background(), minAmount, maxAmount)
	switch {
	case errors.Is(err, usecase.ErrInvalidAmountRange):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, usecase.ErrNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": "orders not found"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	default:
		c.JSON(http.StatusOK, orders)
	}
}
