package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/farik/avito-kitchen/internal/domain"
	"github.com/farik/avito-kitchen/internal/httpclient"
)

type OrderRepo interface {
	CreateWithItems(ctx context.Context, order domain.Order) (domain.Order, error)
	GetByID(ctx context.Context, id int64) (domain.Order, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Order, error)
	ListByRestaurant(ctx context.Context, restaurantID int64) ([]domain.Order, error)
	UpdateStatus(ctx context.Context, id int64, from, to domain.OrderStatus) error
}

type MenuRepoForOrders interface {
	GetByID(ctx context.Context, id int64) (domain.MenuItem, error)
}

type RestaurantRepoForOrders interface {
	GetByID(ctx context.Context, id int64) (domain.Restaurant, error)
}

// RestaurantNotifier уведомляет сервис заведения о новом заказе.
// Интерфейс, а не конкретный HTTP-клиент, — чтобы сервис можно было
// тестировать без сети.
type RestaurantNotifier interface {
	NotifyNewOrder(ctx context.Context, callbackURL string, payload httpclient.NewOrderPayload) error
}

type OrderService struct {
	orders      OrderRepo
	menuItems   MenuRepoForOrders
	restaurants RestaurantRepoForOrders
	webhook     RestaurantNotifier
	log         *slog.Logger
}

func NewOrderService(orders OrderRepo, menuItems MenuRepoForOrders, restaurants RestaurantRepoForOrders, webhook RestaurantNotifier, log *slog.Logger) *OrderService {
	return &OrderService{orders: orders, menuItems: menuItems, restaurants: restaurants, webhook: webhook, log: log}
}

type CreateOrderItemInput struct {
	MenuItemID int64
	Quantity   int32
}

type CreateOrderInput struct {
	UserID       int64
	RestaurantID int64
	Comment      string
	Items        []CreateOrderItemInput
}

// CreateOrder — основной пользовательский сценарий: "оформить заказ".
//
// Проверяет:
//   - у заказа есть хотя бы одна позиция;
//   - заведение существует и активно (закрытый список, п. "Заведения" в ТЗ);
//   - каждая позиция меню принадлежит именно этому заведению и доступна
//     (сценарий "товар закончился");
//
// затем атомарно создаёт заказ и асинхронно уведомляет заведение вебхуком.
func (s *OrderService) CreateOrder(ctx context.Context, in CreateOrderInput) (domain.Order, error) {
	if len(in.Items) == 0 {
		return domain.Order{}, domain.ErrEmptyOrder
	}

	restaurant, err := s.restaurants.GetByID(ctx, in.RestaurantID)
	if err != nil {
		return domain.Order{}, err
	}
	if !restaurant.IsActive {
		return domain.Order{}, domain.ErrRestaurantInactive
	}

	order := domain.Order{
		UserID:       in.UserID,
		RestaurantID: in.RestaurantID,
		Status:       domain.OrderStatusCreated,
		Comment:      in.Comment,
	}

	var total int64
	for _, itemIn := range in.Items {
		mi, err := s.menuItems.GetByID(ctx, itemIn.MenuItemID)
		if errors.Is(err, domain.ErrNotFound) {
			// Несуществующая позиция в корзине — ошибка запроса (422),
			// а не "ресурс /orders не найден" (404).
			return domain.Order{}, fmt.Errorf("%w: %d", domain.ErrMenuItemNotFound, itemIn.MenuItemID)
		}
		if err != nil {
			return domain.Order{}, fmt.Errorf("menu item %d: %w", itemIn.MenuItemID, err)
		}
		if mi.RestaurantID != in.RestaurantID {
			return domain.Order{}, domain.ErrMenuItemForeign
		}
		if !mi.IsAvailable {
			return domain.Order{}, fmt.Errorf("%w: %s", domain.ErrMenuItemUnavailable, mi.Name)
		}
		lineTotal := mi.PriceCents * int64(itemIn.Quantity)
		total += lineTotal
		order.Items = append(order.Items, domain.OrderItem{
			MenuItemID:        mi.ID,
			Name:              mi.Name,
			Quantity:          itemIn.Quantity,
			PriceCentsAtOrder: mi.PriceCents,
		})
	}
	order.TotalCents = total

	created, err := s.orders.CreateWithItems(ctx, order)
	if err != nil {
		return domain.Order{}, fmt.Errorf("save order: %w", err)
	}

	// Уведомление заведения не должно блокировать ответ пользователю и не
	// должно откатывать уже созданный заказ — если заведение недоступно,
	// заказ останется в статусе "created" и будет виден в его API для
	// последующего опроса/ретрая. См. README про упрощения MVP.
	go s.notifyRestaurant(context.WithoutCancel(ctx), restaurant, created)

	return created, nil
}

func (s *OrderService) notifyRestaurant(ctx context.Context, restaurant domain.Restaurant, order domain.Order) {
	payload := httpclient.NewOrderPayload{
		OrderID:      order.ID,
		RestaurantID: restaurant.ID,
	}
	for _, it := range order.Items {
		payload.Items = append(payload.Items, httpclient.NewOrderItem{
			MenuItemID: it.MenuItemID,
			Name:       it.Name,
			Quantity:   it.Quantity,
		})
	}
	if err := s.webhook.NotifyNewOrder(ctx, restaurant.CallbackURL, payload); err != nil {
		s.log.Error("failed to notify restaurant about new order", "order_id", order.ID, "restaurant_id", restaurant.ID, "error", err)
	}
}

func (s *OrderService) GetOrder(ctx context.Context, id int64) (domain.Order, error) {
	return s.orders.GetByID(ctx, id)
}

func (s *OrderService) ListUserOrders(ctx context.Context, userID int64) ([]domain.Order, error) {
	return s.orders.ListByUser(ctx, userID)
}

func (s *OrderService) ListRestaurantOrders(ctx context.Context, restaurantID int64) ([]domain.Order, error) {
	return s.orders.ListByRestaurant(ctx, restaurantID)
}

// UpdateStatus вызывается заведением через API, чтобы продвинуть заказ по
// жизненному циклу (accepted -> cooking -> ready -> ...). Переход проверяется
// доменным правилом CanTransition, поэтому заведение не может, например,
// перевести заказ из "created" сразу в "delivered".
//
// Заведение может менять только свои заказы: для чужого заказа возвращается
// ErrNotFound, чтобы не раскрывать сам факт его существования.
func (s *OrderService) UpdateStatus(ctx context.Context, restaurantID, orderID int64, to domain.OrderStatus) (domain.Order, error) {
	current, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	if current.RestaurantID != restaurantID {
		return domain.Order{}, domain.ErrNotFound
	}
	return s.transition(ctx, current, to)
}

// CancelByUser — пользователь отменяет свой заказ (ветка "отмена" из CJM).
// Разрешено, только пока заведение не начало готовить (см. UserCancellable).
func (s *OrderService) CancelByUser(ctx context.Context, userID, orderID int64) (domain.Order, error) {
	current, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	if current.UserID != userID {
		return domain.Order{}, domain.ErrNotFound
	}
	if !domain.UserCancellable(current.Status) {
		return domain.Order{}, fmt.Errorf("%w: user cannot cancel order in status %s", domain.ErrInvalidTransition, current.Status)
	}
	return s.transition(ctx, current, domain.OrderStatusCancelled)
}

func (s *OrderService) transition(ctx context.Context, current domain.Order, to domain.OrderStatus) (domain.Order, error) {
	if !domain.CanTransition(current.Status, to) {
		return domain.Order{}, fmt.Errorf("%w: %s -> %s", domain.ErrInvalidTransition, current.Status, to)
	}
	if err := s.orders.UpdateStatus(ctx, current.ID, current.Status, to); err != nil {
		return domain.Order{}, err
	}
	return s.orders.GetByID(ctx, current.ID)
}
