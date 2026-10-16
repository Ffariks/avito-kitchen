package service

import (
	"context"

	"github.com/farik/avito-kitchen/internal/domain"
)

type RestaurantRepo interface {
	List(ctx context.Context) ([]domain.Restaurant, error)
	GetByID(ctx context.Context, id int64) (domain.Restaurant, error)
}

type MenuRepo interface {
	ListByRestaurant(ctx context.Context, restaurantID int64) ([]domain.MenuItem, error)
	GetByID(ctx context.Context, id int64) (domain.MenuItem, error)
	Create(ctx context.Context, mi domain.MenuItem) (domain.MenuItem, error)
	SetAvailability(ctx context.Context, restaurantID, id int64, isAvailable bool) error
}

// CatalogService закрывает сценарии "пользователь смотрит список заведений
// и меню" и "заведение управляет своим меню".
type CatalogService struct {
	restaurants RestaurantRepo
	menuItems   MenuRepo
}

func NewCatalogService(restaurants RestaurantRepo, menuItems MenuRepo) *CatalogService {
	return &CatalogService{restaurants: restaurants, menuItems: menuItems}
}

func (s *CatalogService) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	return s.restaurants.List(ctx)
}

func (s *CatalogService) GetRestaurant(ctx context.Context, id int64) (domain.Restaurant, error) {
	return s.restaurants.GetByID(ctx, id)
}

func (s *CatalogService) ListMenu(ctx context.Context, restaurantID int64) ([]domain.MenuItem, error) {
	// Убеждаемся, что заведение существует, чтобы вернуть 404, а не пустой список.
	if _, err := s.restaurants.GetByID(ctx, restaurantID); err != nil {
		return nil, err
	}
	return s.menuItems.ListByRestaurant(ctx, restaurantID)
}

func (s *CatalogService) AddMenuItem(ctx context.Context, mi domain.MenuItem) (domain.MenuItem, error) {
	if _, err := s.restaurants.GetByID(ctx, mi.RestaurantID); err != nil {
		return domain.MenuItem{}, err
	}
	return s.menuItems.Create(ctx, mi)
}

func (s *CatalogService) SetMenuItemAvailability(ctx context.Context, restaurantID, itemID int64, isAvailable bool) error {
	return s.menuItems.SetAvailability(ctx, restaurantID, itemID, isAvailable)
}
