package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RestaurantWebhookClient уведомляет сервис заведения о новых заказах.
// Заведение — внешний сервис (в MVP это restaurant-service), реализующий
// собственный API и вызываемый платформой по CallbackURL, сохранённому
// в таблице restaurants.
type RestaurantWebhookClient struct {
	httpClient *http.Client
}

func NewRestaurantWebhookClient(timeout time.Duration) *RestaurantWebhookClient {
	return &RestaurantWebhookClient{httpClient: &http.Client{Timeout: timeout}}
}

type NewOrderPayload struct {
	OrderID      int64          `json:"order_id"`
	RestaurantID int64          `json:"restaurant_id"`
	Items        []NewOrderItem `json:"items"`
}

type NewOrderItem struct {
	MenuItemID int64  `json:"menu_item_id"`
	Name       string `json:"name"`
	Quantity   int32  `json:"quantity"`
}

// NotifyNewOrder отправляет POST {callbackURL}/orders в сервис заведения.
//
// Упрощение MVP: вызов синхронный и не ретраится. В проде здесь была бы
// очередь (outbox + брокер) с ретраями и идемпотентностью по order_id —
// см. README, раздел "Известные упрощения".
func (c *RestaurantWebhookClient) NotifyNewOrder(ctx context.Context, callbackURL string, payload NewOrderPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL+"/orders", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call restaurant webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("restaurant webhook returned status %d", resp.StatusCode)
	}
	return nil
}
