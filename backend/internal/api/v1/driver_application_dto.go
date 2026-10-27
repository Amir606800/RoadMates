package v1

import (
	"time"

	"roadmates/backend/internal/domain"
)

type DriverApplicationRequest struct {
	ContactPhone string `json:"contact_phone"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	PINCode      string `json:"pin_code"`
	CarMake      string `json:"car_make"`
	CarModel     string `json:"car_model"`
	CarYear      int    `json:"car_year"`
	CarPlate     string `json:"car_plate"`
	CarColor     string `json:"car_color"`
	Seats        int    `json:"seats"`
}

// Explicit public DTO: neither the PIN nor its ciphertext can be serialized.
type DriverApplicationResponse struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	ContactPhone string     `json:"contact_phone"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	CarMake      string     `json:"car_make"`
	CarModel     string     `json:"car_model"`
	CarYear      int        `json:"car_year"`
	CarPlate     string     `json:"car_plate"`
	CarColor     string     `json:"car_color"`
	Seats        int        `json:"seats"`
	Status       string     `json:"status"`
	ReviewedBy   *string    `json:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	RejectReason *string    `json:"reject_reason"`
	CreatedAt    time.Time  `json:"created_at"`
}

type AdminApplicationDetail struct {
	DriverApplicationResponse
	PINCode string `json:"pin_code"`
}
type ReviewApplicationRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}
type ApplicationPagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}
type ApplicationListResponse struct {
	Applications []DriverApplicationResponse `json:"applications"`
	Pagination   ApplicationPagination       `json:"pagination"`
}

func applicationResponse(app domain.DriverApplication) DriverApplicationResponse {
	return DriverApplicationResponse{ID: app.ID, UserID: app.UserID, ContactPhone: app.ContactPhone, FirstName: app.FirstName, LastName: app.LastName, CarMake: app.CarMake, CarModel: app.CarModel, CarYear: app.CarYear, CarPlate: app.CarPlate, CarColor: app.CarColor, Seats: app.Seats, Status: app.Status, ReviewedBy: app.ReviewedBy, ReviewedAt: app.ReviewedAt, RejectReason: app.RejectReason, CreatedAt: app.CreatedAt}
}
