package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/validation"
)

type AuthTx interface {
	LockRefresh(context.Context, string) (domain.RefreshToken, error)
	UserByID(context.Context, string) (domain.User, error)
	InsertRefresh(context.Context, domain.RefreshToken) error
	ReplaceRefresh(context.Context, string, string) error
	RevokeFamily(context.Context, string) error
}

type AuthStore interface {
	CreateUser(context.Context, domain.NewUser, domain.RefreshToken) (domain.User, error)
	UserByEmail(context.Context, string) (domain.User, error)
	UserByID(context.Context, string) (domain.User, error)
	InsertRefresh(context.Context, domain.RefreshToken) error
	Transaction(context.Context, func(AuthTx) error) error
	BootstrapAdmin(context.Context, domain.NewUser) (bool, error)
	CleanupRefresh(context.Context) error
}

type Auth struct {
	store      AuthStore
	tokens     *auth.Tokens
	refreshTTL time.Duration
	dummyHash  []byte
	logger     *slog.Logger
}
type ClientMetadata struct{ UserAgent, IP string }
type Registration struct{ Email, Password, FirstName, LastName, Phone string }

func NewAuth(store AuthStore, tokens *auth.Tokens, refreshTTL time.Duration, logger *slog.Logger) (*Auth, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("RoadMates-dummy-password!"), 12)
	if err != nil {
		return nil, err
	}
	return &Auth{store, tokens, refreshTTL, dummy, logger}, nil
}

func (s *Auth) newRefresh(userID, familyID string, meta ClientMetadata) (domain.RefreshToken, string, error) {
	raw, err := auth.OpaqueToken()
	if err != nil {
		return domain.RefreshToken{}, "", err
	}
	return domain.RefreshToken{ID: uuid.NewString(), UserID: userID, FamilyID: familyID, TokenHash: auth.HashToken(raw), ExpiresAt: time.Now().Add(s.refreshTTL), UserAgent: meta.UserAgent, IP: meta.IP}, raw, nil
}

func (s *Auth) Register(ctx context.Context, input Registration, meta ClientMetadata) (domain.Session, error) {
	if err := auth.ValidatePassword(input.Password); err != nil {
		return domain.Session{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return domain.Session{}, err
	}
	refresh, raw, err := s.newRefresh("", uuid.NewString(), meta)
	if err != nil {
		return domain.Session{}, err
	}
	user, err := s.store.CreateUser(ctx, domain.NewUser{Email: input.Email, PasswordHash: string(hash), FirstName: input.FirstName, LastName: input.LastName, Phone: input.Phone}, refresh)
	if err != nil {
		return domain.Session{}, err
	}
	access, err := s.tokens.Access(user)
	return domain.Session{AccessToken: access, RefreshToken: raw, User: user}, err
}

func (s *Auth) Login(ctx context.Context, email, password string, meta ClientMetadata) (domain.Session, error) {
	user, err := s.store.UserByEmail(ctx, email)
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		return domain.Session{}, err
	}
	hash := s.dummyHash
	if err == nil {
		hash = []byte(user.PasswordHash)
	}
	// Both unknown accounts and wrong passwords do one cost-12 comparison.
	comparison := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil || comparison != nil || !user.IsActive {
		return domain.Session{}, apperrors.ErrInvalidCredentials
	}
	refresh, raw, err := s.newRefresh(user.ID, uuid.NewString(), meta)
	if err != nil {
		return domain.Session{}, err
	}
	access, err := s.tokens.Access(user)
	if err != nil {
		return domain.Session{}, err
	}
	if err = s.store.InsertRefresh(ctx, refresh); err != nil {
		return domain.Session{}, err
	}
	return domain.Session{AccessToken: access, RefreshToken: raw, User: user}, nil
}

func (s *Auth) Refresh(ctx context.Context, raw string, meta ClientMetadata) (domain.Session, error) {
	var result domain.Session
	var denial error
	err := s.store.Transaction(ctx, func(tx AuthTx) error {
		old, err := tx.LockRefresh(ctx, auth.HashToken(raw))
		if err != nil {
			return err
		}
		if old.RevokedAt != nil {
			// Return nil from the transaction so family revocation is committed.
			if err := tx.RevokeFamily(ctx, old.FamilyID); err != nil {
				return err
			}
			denial = apperrors.ErrRefreshReused
			return nil
		}
		if !old.ExpiresAt.After(time.Now()) {
			return apperrors.ErrUnauthorized
		}
		user, err := tx.UserByID(ctx, old.UserID)
		if err != nil {
			return err
		}
		if !user.IsActive {
			if err := tx.RevokeFamily(ctx, old.FamilyID); err != nil {
				return err
			}
			denial = apperrors.ErrUnauthorized
			return nil
		}
		refresh, nextRaw, err := s.newRefresh(user.ID, old.FamilyID, meta)
		if err != nil {
			return err
		}
		access, err := s.tokens.Access(user)
		if err != nil {
			return err
		}
		if err = tx.InsertRefresh(ctx, refresh); err != nil {
			return err
		}
		if err = tx.ReplaceRefresh(ctx, old.ID, refresh.ID); err != nil {
			return err
		}
		result = domain.Session{AccessToken: access, RefreshToken: nextRaw, User: user}
		return nil
	})
	if err != nil {
		return domain.Session{}, err
	}
	if denial != nil {
		return domain.Session{}, denial
	}
	return result, nil
}

func (s *Auth) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.store.Transaction(ctx, func(tx AuthTx) error {
		token, err := tx.LockRefresh(ctx, auth.HashToken(raw))
		if errors.Is(err, apperrors.ErrUnauthorized) {
			return nil
		}
		if err != nil {
			return err
		}
		return tx.RevokeFamily(ctx, token.FamilyID)
	})
}

func (s *Auth) Me(ctx context.Context, id string) (domain.User, error) {
	user, err := s.store.UserByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) || (err == nil && !user.IsActive) {
		return domain.User{}, apperrors.ErrUnauthorized
	}
	return user, err
}

func (s *Auth) Bootstrap(ctx context.Context, email, password string) error {
	if email == "" && password == "" {
		return nil
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if !validation.Email(email) || password == "" {
		return errors.New("ADMIN_EMAIL and ADMIN_PASSWORD must both be valid and set")
	}
	if err := auth.ValidatePassword(password); err != nil {
		return errors.New("ADMIN_PASSWORD does not meet the password policy")
	}
	if _, err := s.store.UserByEmail(ctx, email); err == nil {
		return nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return errors.New("could not check admin bootstrap account")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return errors.New("could not prepare admin bootstrap password")
	}
	created, err := s.store.BootstrapAdmin(ctx, domain.NewUser{Email: email, PasswordHash: string(hash), FirstName: "RoadMates", LastName: "Admin", Phone: ""})
	if err != nil {
		return errors.New("could not bootstrap admin account")
	}
	if created {
		s.logger.Info("admin bootstrap account created")
	}
	return nil
}

func (s *Auth) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := s.store.CleanupRefresh(cleanupCtx); err != nil && ctx.Err() == nil {
			s.logger.Warn("refresh token cleanup failed")
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
