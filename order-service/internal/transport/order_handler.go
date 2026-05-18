package transport

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/cache"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type idempotencyEntry struct {
	status int
	body   any
}

type OrderHandler struct {
	uc          *usecase.OrderUsecase
	cache       *cache.OrderCache
	idempotency sync.Map
}

func NewOrderHandler(uc *usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

func NewOrderHandlerWithCache(uc *usecase.OrderUsecase, orderCache *cache.OrderCache) *OrderHandler {
	return &OrderHandler{uc: uc, cache: orderCache}
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

	order, err := h.uc.CreateOrder(c.Request.Context(), req.CustomerID, req.ItemName, req.Amount)

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
		// Cache the order even if payment service is unavailable
		if h.cache != nil && order != nil {
			if err := h.cache.Set(c.Request.Context(), order.ID, order); err != nil {
				log.Printf("failed to cache order: %v", err)
			}
		}
	case err != nil:
		log.Printf("create order error: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	default:
		status = http.StatusCreated
		body = order
		// Cache newly created orders
		if h.cache != nil && order != nil {
			if err := h.cache.Set(c.Request.Context(), order.ID, order); err != nil {
				log.Printf("failed to cache order: %v", err)
			}
		}
	}

	if key != "" {
		h.idempotency.Store(key, idempotencyEntry{status: status, body: body})
	}
	c.JSON(status, body)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	// Cache-aside pattern: check cache first
	if h.cache != nil {
		var cachedOrder any
		if err := h.cache.Get(ctx, id, &cachedOrder); err != nil {
			log.Printf("cache get error (non-fatal): %v", err)
		} else if cachedOrder != nil {
			log.Printf("cache hit for order %s", id)
			c.JSON(http.StatusOK, cachedOrder)
			return
		}
	}

	// Cache miss or no cache: query database
	order, err := h.uc.GetOrder(ctx, id)
	switch {
	case errors.Is(err, usecase.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	default:
		// Cache the retrieved order
		if h.cache != nil && order != nil {
			if err := h.cache.Set(ctx, id, order); err != nil {
				log.Printf("failed to cache order: %v", err)
			}
		}
		c.JSON(http.StatusOK, order)
	}
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	order, err := h.uc.CancelOrder(ctx, id)
	switch {
	case errors.Is(err, usecase.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
	case errors.Is(err, usecase.ErrNotCancellable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	default:
		// Invalidate cache on status change
		if h.cache != nil {
			if err := h.cache.InvalidateOrder(ctx, id); err != nil {
				log.Printf("failed to invalidate cache for order %s: %v", id, err)
			}
		}
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

	orders, err := h.uc.OrderList(c.Request.Context(), minAmount, maxAmount)
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
