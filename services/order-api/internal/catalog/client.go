package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrCircuitOpen     = errors.New("catalog circuit breaker is open")
)

type Product struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PricePence int    `json:"price_pence"`
}

type ProductGetter interface {
	GetProduct(ctx context.Context, productID string) (Product, error)
}

type BreakerSettings struct {
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	breaker    *gobreaker.CircuitBreaker[Product]
}

func NewClient(baseURL string, timeout time.Duration, bs BreakerSettings) *Client {
	cb := gobreaker.NewCircuitBreaker[Product](gobreaker.Settings{
		Name:        "catalog",
		MaxRequests: bs.MaxRequests,
		Interval:    bs.Interval,
		Timeout:     bs.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		breaker: cb,
	}
}

func (c *Client) GetProduct(ctx context.Context, productID string) (Product, error) {
	product, err := c.breaker.Execute(func() (Product, error) {
		return c.doGetProduct(ctx, productID)
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return Product{}, ErrCircuitOpen
		}
		return Product{}, err
	}
	return product, nil
}

func (c *Client) doGetProduct(ctx context.Context, productID string) (Product, error) {
	endpoint := fmt.Sprintf("%s/api/catalog/products/%s", c.baseURL, url.PathEscape(productID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Product{}, fmt.Errorf("build request: %w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return Product{}, fmt.Errorf("catalog request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return Product{}, fmt.Errorf("%w: %s", ErrProductNotFound, productID)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return Product{}, fmt.Errorf("catalog returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var product Product
	if err := json.NewDecoder(res.Body).Decode(&product); err != nil {
		return Product{}, fmt.Errorf("decode product: %w", err)
	}
	return product, nil
}
