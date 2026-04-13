package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"food-ordering-api/internal/catalog"
	"food-ordering-api/internal/coupon"
	"food-ordering-api/internal/models"
)

func testServer(t *testing.T, promo PromoValidator) *httptest.Server {
	t.Helper()
	srv := &Server{
		Log:              slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})),
		Catalog:          catalog.NewMemory("https://example.test/images/"),
		Coupon:           promo,
		APIKey:           "test-key",
		MaxBodyBytes:     1 << 20,
		RequestTimeout:   60 * time.Second,
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestHealthz(t *testing.T) {
	ts := testServer(t, &coupon.Validator{})
	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestListProducts(t *testing.T) {
	ts := testServer(t, &coupon.Validator{})
	res, err := http.Get(ts.URL + "/product")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var list []models.Product
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 9 {
		t.Fatalf("want 9 products, got %d", len(list))
	}
}

func TestGetProduct_notFound(t *testing.T) {
	ts := testServer(t, &coupon.Validator{})
	res, err := http.Get(ts.URL + "/product/999")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestGetProduct_badID(t *testing.T) {
	ts := testServer(t, &coupon.Validator{})
	res, err := http.Get(ts.URL + "/product/x")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestPlaceOrder_unauthorized(t *testing.T) {
	ts := testServer(t, &coupon.Validator{})
	body := `{"items":[{"productId":"1","quantity":1}]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestPlaceOrder_ok(t *testing.T) {
	ts := testServer(t, &coupon.Validator{})
	body := `{"items":[{"productId":"1","quantity":2}]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", "test-key")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var o models.Order
	if err := json.NewDecoder(res.Body).Decode(&o); err != nil {
		t.Fatal(err)
	}
	if o.ID == "" || len(o.Items) != 1 || o.Items[0].Quantity != 2 {
		t.Fatalf("unexpected order: %+v", o)
	}
}

func TestPlaceOrder_invalidCoupon(t *testing.T) {
	// Zero Validator: no corpus loaded → any non-empty code fails validation.
	ts := testServer(t, &coupon.Validator{})
	body := `{"items":[{"productId":"1","quantity":1}],"couponCode":"12345678"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", "test-key")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", res.StatusCode)
	}
}

type stubPromo struct {
	ok bool
}

func (s stubPromo) Valid(string) bool { return s.ok }

func TestPlaceOrder_validCouponWithStub(t *testing.T) {
	ts := testServer(t, stubPromo{ok: true})
	body := `{"items":[{"productId":"1","quantity":1}],"couponCode":"VALIDCODE8"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", "test-key")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestPlaceOrder_bodyTooLarge(t *testing.T) {
	// MaxBytesReader must trip while reading a long JSON string (decoder may return
	// 400 for truncated JSON if the limit is too small — use a large coupon field).
	srv := &Server{
		Log:            slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})),
		Catalog:        catalog.NewMemory("https://example.test/images/"),
		Coupon:         stubPromo{ok: true},
		APIKey:         "test-key",
		MaxBodyBytes:   200,
		RequestTimeout: 60 * time.Second,
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	huge := strings.Repeat("A", 50_000)
	body := `{"items":[{"productId":"1","quantity":1}],"couponCode":"` + huge + `"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", "test-key")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d", res.StatusCode)
	}
}
