package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"food-ordering-api/internal/coupon"
)

// Config holds runtime configuration. Values come from the environment (and optional
// `.env` file loaded in main) so the same binary can run locally, in Docker, or behind
// a load balancer without code changes.
type Config struct {
	Port              string
	APIKey            string
	LogJSON           bool
	LogLevel          string
	MaxBodyBytes      int64
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	RequestTimeout    time.Duration
	ImageBaseURL      string
	CouponFiles       [3]coupon.File
}

// FromEnv loads configuration. Defaults match the public OpenAPI demo (`apitest`, port 8080).
func FromEnv() Config {
	port := getenv("PORT", "8080")
	key := getenv("API_KEY", "apitest")

	logJSON := truthy(os.Getenv("LOG_JSON")) || strings.EqualFold(os.Getenv("LOG_FORMAT"), "json")
	logLevel := getenv("LOG_LEVEL", "info")

	maxBody := int64(1 << 20) // 1 MiB
	if v := os.Getenv("MAX_BODY_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxBody = n
		}
	}

	shutdown := secondsEnv("SHUTDOWN_TIMEOUT_SEC", 30)
	readHdr := secondsEnv("READ_HEADER_TIMEOUT_SEC", 5)
	read := secondsEnv("READ_TIMEOUT_SEC", 15)
	write := secondsEnv("WRITE_TIMEOUT_SEC", 60)
	idle := secondsEnv("IDLE_TIMEOUT_SEC", 120)
	reqTO := secondsEnv("CHI_REQUEST_TIMEOUT_SEC", 60)

	imageBase := getenv("IMAGE_BASE_URL", "https://orderfoodonline.deno.dev/public/images/")
	imageBase = strings.TrimRight(imageBase, "/") + "/"

	couponFiles := [3]coupon.File{
		{
			URL:   getenv("COUPON_1_URL", "https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase1.gz"),
			GZ:    getenv("COUPON_1_GZ", "couponbase1.gz"),
			Cache: getenv("COUPON_1_IDX", "couponbase1.idx"),
		},
		{
			URL:   getenv("COUPON_2_URL", "https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase2.gz"),
			GZ:    getenv("COUPON_2_GZ", "couponbase2.gz"),
			Cache: getenv("COUPON_2_IDX", "couponbase2.idx"),
		},
		{
			URL:   getenv("COUPON_3_URL", "https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase3.gz"),
			GZ:    getenv("COUPON_3_GZ", "couponbase3.gz"),
			Cache: getenv("COUPON_3_IDX", "couponbase3.idx"),
		},
	}

	return Config{
		Port:              port,
		APIKey:            key,
		LogJSON:           logJSON,
		LogLevel:          logLevel,
		MaxBodyBytes:      maxBody,
		ShutdownTimeout:   shutdown,
		ReadHeaderTimeout: readHdr,
		ReadTimeout:       read,
		WriteTimeout:      write,
		IdleTimeout:       idle,
		RequestTimeout:    reqTO,
		ImageBaseURL:      imageBase,
		CouponFiles:       couponFiles,
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func secondsEnv(key string, defSec int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(defSec) * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return time.Duration(defSec) * time.Second
	}
	return time.Duration(n) * time.Second
}
