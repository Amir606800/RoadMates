package v1

import (
	"roadmates/backend/internal/domain"
	"time"
)

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     string    `json:"phone"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active"`
}
type SessionResponse struct {
	AccessToken string       `json:"access_token"`
	User        UserResponse `json:"user"`
}

func userResponse(user domain.User) UserResponse {
	return UserResponse{user.ID, user.Email, user.FirstName, user.LastName, user.Phone, user.Roles, user.CreatedAt, user.UpdatedAt, user.IsActive}
}
