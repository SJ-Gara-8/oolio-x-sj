package models

// ProductImage holds URLs for different screen sizes (matches demo API; OpenAPI Product schema is minimal).
type ProductImage struct {
	Thumbnail string `json:"thumbnail"`
	Mobile    string `json:"mobile"`
	Tablet    string `json:"tablet"`
	Desktop   string `json:"desktop"`
}

// Product represents a food item available for ordering.
type Product struct {
	ID       string       `json:"id"`
	Image    ProductImage `json:"image"`
	Name     string       `json:"name"`
	Category string       `json:"category"`
	Price    float64      `json:"price"`
}

// OrderItem is a line item in an order request or response.
type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// OrderReq is the request body for POST /order.
type OrderReq struct {
	Items      []OrderItem `json:"items"`
	CouponCode string      `json:"couponCode,omitempty"`
}

// Order is the response body for POST /order.
type Order struct {
	ID         string      `json:"id"`
	Items      []OrderItem `json:"items"`
	CouponCode string      `json:"couponCode,omitempty"`
	Products   []Product   `json:"products"`
}

// ErrorResponse matches components/schemas/ApiResponse in the OpenAPI document.
type ErrorResponse struct {
	Code    int32  `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}
