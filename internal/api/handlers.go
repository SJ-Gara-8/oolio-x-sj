package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"food-ordering-api/internal/catalog"
	"food-ordering-api/internal/models"

	"github.com/go-chi/chi/v5"
)

// PromoValidator checks coupon codes on POST /order (optional field in the request).
type PromoValidator interface {
	Valid(code string) bool
}

// Server wires HTTP handlers with shared dependencies (constructor injection).
type Server struct {
	Log            *slog.Logger
	Catalog        catalog.Catalog
	Coupon         PromoValidator
	APIKey         string
	MaxBodyBytes   int64
	RequestTimeout time.Duration
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.Log.Error("encode_json", "err", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, typ, message string) {
	s.writeJSON(w, status, models.ErrorResponse{
		Code:    int32(status),
		Type:    typ,
		Message: message,
	})
}

// ListProducts handles GET /product.
func (s *Server) ListProducts(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.Catalog.List())
}

// GetProduct handles GET /product/{productId}.
func (s *Server) GetProduct(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "productId")
	if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_id", "invalid product ID")
		return
	}
	p, ok := s.Catalog.ByID(raw)
	if !ok {
		s.writeError(w, http.StatusNotFound, "not_found", "product not found")
		return
	}
	s.writeJSON(w, http.StatusOK, p)
}

// PlaceOrder handles POST /order.
func (s *Server) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.MaxBodyBytes)
	if r.Header.Get("api_key") != s.APIKey {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "valid api_key header required")
		return
	}

	var req models.OrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			s.writeError(w, http.StatusBadRequest, "invalid_input", "empty JSON body")
			return
		}
		// net/http MaxBytesReader reports an error containing "body too large" (no exported sentinel).
		if strings.Contains(err.Error(), "body too large") {
			s.writeError(w, http.StatusRequestEntityTooLarge, "invalid_input", "request body too large")
			return
		}
		s.writeError(w, http.StatusBadRequest, "invalid_input", "invalid JSON body")
		return
	}

	if len(req.Items) == 0 {
		s.writeError(w, http.StatusUnprocessableEntity, "validation", "at least one item is required")
		return
	}
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			s.writeError(w, http.StatusUnprocessableEntity, "validation", "item quantity must be greater than zero")
			return
		}
		if _, ok := s.Catalog.ByID(item.ProductID); !ok {
			s.writeError(w, http.StatusUnprocessableEntity, "constraint", "invalid product specified")
			return
		}
	}

	if req.CouponCode != "" && !s.Coupon.Valid(req.CouponCode) {
		s.writeError(w, http.StatusUnprocessableEntity, "validation", "invalid coupon code")
		return
	}

	seen := make(map[string]struct{})
	var products []models.Product
	for _, item := range req.Items {
		if _, already := seen[item.ProductID]; !already {
			seen[item.ProductID] = struct{}{}
			p, _ := s.Catalog.ByID(item.ProductID)
			products = append(products, p)
		}
	}

	s.writeJSON(w, http.StatusOK, models.Order{
		ID:         newUUID(),
		Items:      req.Items,
		CouponCode: req.CouponCode,
		Products:   products,
	})
}

// Health handles GET /healthz (not part of OpenAPI; for orchestrators).
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func newUUID() string {
	var b [16]byte
	rand.Read(b[:]) //nolint:errcheck
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
