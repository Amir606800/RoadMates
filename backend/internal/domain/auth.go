package domain

import "time"

type User struct {
	ID, Email, PasswordHash, FirstName, LastName, Phone string
	CreatedAt, UpdatedAt                                time.Time
	IsActive                                            bool
	Roles                                               []string
}

type NewUser struct{ Email, PasswordHash, FirstName, LastName, Phone string }

type RefreshToken struct {
	ID, UserID, FamilyID, TokenHash string
	ExpiresAt                       time.Time
	RevokedAt                       *time.Time
	UserAgent, IP                   string
}

type Session struct {
	AccessToken, RefreshToken string
	User                      User
}
