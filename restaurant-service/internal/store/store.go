package store

import (
	"sync"

	"github.com/farik/avito-kitchen/restaurant-service/internal/domain"
)

// Store — упрощение MVP: заведение хранит принятые заказы в памяти.
// В реальной интеграции у заведения была бы своя БД/учётная система,
// платформа лишь дёргает её API.
type Store struct {
	mu     sync.Mutex
	orders map[int64]domain.IncomingOrder
}

func New() *Store {
	return &Store{orders: make(map[int64]domain.IncomingOrder)}
}

// Save сохраняет заказ и возвращает false, если заказ с таким order_id уже
// был получен раньше. Это делает приём вебхука идемпотентным: повторная
// доставка того же заказа не запустит готовку второй раз.
func (s *Store) Save(order domain.IncomingOrder) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.orders[order.OrderID]; exists {
		return false
	}
	s.orders[order.OrderID] = order
	return true
}

func (s *Store) List() []domain.IncomingOrder {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]domain.IncomingOrder, 0, len(s.orders))
	for _, o := range s.orders {
		result = append(result, o)
	}
	return result
}
