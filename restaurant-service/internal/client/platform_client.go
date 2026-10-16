package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PlatformClient вызывает API основного сервиса Авито.Кухня, чтобы
// продвинуть статус заказа. Это "своё API для обмена информацией" в
// терминах ТЗ, только со стороны заведения — оно является клиентом
// платформы для смены статусов.
type PlatformClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPlatformClient(baseURL string, timeout time.Duration) *PlatformClient {
	return &PlatformClient{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateOrderStatus переводит заказ этого заведения в новый статус.
func (c *PlatformClient) UpdateOrderStatus(ctx context.Context, restaurantID, orderID int64, status string) error {
	body, err := json.Marshal(updateStatusRequest{Status: status})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/restaurant/%d/orders/%d/status", c.baseURL, restaurantID, orderID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("platform returned status %d for order %d -> %s", resp.StatusCode, orderID, status)
	}
	return nil
}
