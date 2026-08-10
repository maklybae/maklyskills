package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"orderflow/pkg/httputil"
	"orderflow/pkg/xerrors"
	"orderflow/services/orders/internal/core"
)

const (
	defaultTimeout  = 3 * time.Second
	maxDetailLength = 512
)

// ErrUnexpectedResponse covers everything the inventory service answers that we
// have no rule for; it always ends up as a 500 for the caller.
var ErrUnexpectedResponse = errors.New("unexpected inventory response")

// Client is the inventory service seen from the order flow.
type Client struct {
	baseURL string
	http    *http.Client
}

// Option overrides a Client default.
type Option func(*Client)

type linePayload struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type reservePayload struct {
	OrderID string        `json:"order_id"`
	Lines   []linePayload `json:"lines"`
}

type releasePayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

type returnPayload struct {
	OrderID string        `json:"order_id"`
	Lines   []linePayload `json:"lines"`
}

// NewClient returns a client for the inventory service at baseURL.
func NewClient(baseURL string, opts ...Option) (*Client, error) {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("inventory: base url is required")
	}

	client := &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client, nil
}

// WithHTTPClient replaces the HTTP client, which is how tests point at a stub server.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.http = httpClient
		}
	}
}

func (c *Client) Reserve(ctx context.Context, reservation core.StockReservation) error {
	lines := make([]linePayload, 0, len(reservation.Items))
	for _, item := range reservation.Items {
		lines = append(lines, linePayload{SKU: item.SKU, Quantity: item.Quantity})
	}
	return c.post(ctx, "/reservations", reservePayload{OrderID: reservation.OrderID, Lines: lines})
}

func (c *Client) Release(ctx context.Context, release core.StockRelease) error {
	return c.post(ctx, "/reservations/release", releasePayload{OrderID: release.OrderID, Reason: release.Reason})
}

func (c *Client) Restock(ctx context.Context, ret core.StockReturn) error {
	lines := make([]linePayload, 0, len(ret.Lines))
	for _, line := range ret.Lines {
		lines = append(lines, linePayload{SKU: line.SKU, Quantity: line.Quantity})
	}
	return c.post(ctx, "/stock/returns", returnPayload{OrderID: ret.OrderID, Lines: lines})
}

func (c *Client) post(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return xerrors.Wrapf(err, "encode request for inventory %s", path)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return xerrors.Wrapf(err, "build request for inventory %s", path)
	}
	req.Header.Set("Content-Type", "application/json")
	if id := httputil.RequestIDFrom(ctx); id != "" {
		req.Header.Set(httputil.HeaderRequestID, id)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return xerrors.Wrapf(err, "call inventory %s", path)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return responseError(path, resp)
}

// responseError maps the inventory answer onto the domain errors of this
// service, so a handler can keep matching on core sentinels.
func responseError(path string, resp *http.Response) error {
	detail := errorDetail(resp.Body)

	switch resp.StatusCode {
	case http.StatusConflict:
		return xerrors.Wrapf(core.ErrOutOfStock, "inventory %s: %s", path, detail)
	case http.StatusBadRequest:
		return xerrors.Wrapf(core.ErrInvalidRequest, "inventory %s: %s", path, detail)
	default:
		return xerrors.Wrapf(ErrUnexpectedResponse, "inventory %s answered %d: %s", path, resp.StatusCode, detail)
	}
}

func errorDetail(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, maxDetailLength))
	if err != nil || len(raw) == 0 {
		return "no detail"
	}

	var envelope httputil.ErrorBody
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Error.Message != "" {
		return envelope.Error.Message
	}
	return strings.TrimSpace(string(raw))
}
