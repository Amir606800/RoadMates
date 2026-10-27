package service

import (
	"context"

	"github.com/google/uuid"

	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/pii"
)

type DriverApplicationStore interface {
	Create(context.Context, domain.DriverApplication) (domain.DriverApplication, error)
	Latest(context.Context, string) (domain.DriverApplication, error)
	List(context.Context, string, int, int) ([]domain.DriverApplication, error)
	ByID(context.Context, string) (domain.DriverApplication, error)
	Review(context.Context, string, string, string, string) (domain.DriverApplication, error)
}

type DriverApplications struct {
	store  DriverApplicationStore
	cipher *pii.Cipher
}
type DriverApplicationInput struct {
	ContactPhone, FirstName, LastName, PINCode string
	CarMake, CarModel, CarPlate, CarColor      string
	CarYear, Seats                             int
}

func NewDriverApplications(store DriverApplicationStore, cipher *pii.Cipher) *DriverApplications {
	return &DriverApplications{store, cipher}
}

func (s *DriverApplications) Apply(ctx context.Context, userID string, input DriverApplicationInput) (domain.DriverApplication, error) {
	id := uuid.NewString()
	encrypted, err := s.cipher.EncryptPIN(input.PINCode, userID, id)
	if err != nil {
		return domain.DriverApplication{}, err
	}
	return s.store.Create(ctx, domain.DriverApplication{ID: id, UserID: userID, ContactPhone: input.ContactPhone, FirstName: input.FirstName, LastName: input.LastName, PINCodeEncrypted: encrypted, CarMake: input.CarMake, CarModel: input.CarModel, CarYear: input.CarYear, CarPlate: input.CarPlate, CarColor: input.CarColor, Seats: input.Seats})
}

func (s *DriverApplications) Latest(ctx context.Context, userID string) (domain.DriverApplication, error) {
	return s.store.Latest(ctx, userID)
}
func (s *DriverApplications) List(ctx context.Context, status string, limit, offset int) ([]domain.DriverApplication, bool, error) {
	items, err := s.store.List(ctx, status, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, hasMore, nil
}
func (s *DriverApplications) Detail(ctx context.Context, id string) (domain.DriverApplication, string, error) {
	application, err := s.store.ByID(ctx, id)
	if err != nil {
		return application, "", err
	}
	pin, err := s.cipher.DecryptPIN(application.PINCodeEncrypted, application.UserID, application.ID)
	return application, pin, err
}
func (s *DriverApplications) Review(ctx context.Context, id, adminID, action, reason string) (domain.DriverApplication, error) {
	return s.store.Review(ctx, id, adminID, action, reason)
}
