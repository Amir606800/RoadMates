package v1

import (
	"encoding/json"
	"fmt"
	"roadmates/backend/internal/domain"
	"time"
)

type DistrictResponse struct {
	ID             int    `json:"id"`
	NameAZ         string `json:"name_az"`
	Slug           string `json:"slug"`
	RequiresPermit bool   `json:"requires_permit"`
}
type PublishRideRequest struct {
	FromID      int         `json:"from_district_id"`
	ToID        int         `json:"to_district_id"`
	DepartureAt time.Time   `json:"departure_at"`
	SeatsTotal  int         `json:"seats_total"`
	PriceAZN    json.Number `json:"price_azn"`
	Notes       string      `json:"notes"`
}
type RideDriverResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
type RideResponse struct {
	ID             string             `json:"id"`
	Driver         RideDriverResponse `json:"driver"`
	FromID         int                `json:"from_district_id"`
	ToID           int                `json:"to_district_id"`
	From           DistrictResponse   `json:"from_district"`
	To             DistrictResponse   `json:"to_district"`
	DepartureAt    time.Time          `json:"departure_at"`
	SeatsTotal     int                `json:"seats_total"`
	SeatsAvailable int                `json:"seats_available"`
	PriceAZN       json.Number        `json:"price_azn"`
	Notes          string             `json:"notes"`
	Status         string             `json:"status"`
	CreatedAt      time.Time          `json:"created_at"`
}

func districtResponse(d domain.District) DistrictResponse {
	return DistrictResponse{d.ID, d.NameAZ, d.Slug, d.RequiresPermit}
}
func rideResponse(r domain.Ride) RideResponse {
	return RideResponse{r.ID, RideDriverResponse{r.DriverID, r.DriverFirstName, r.DriverLastName}, r.From.ID, r.To.ID, districtResponse(r.From), districtResponse(r.To), r.DepartureAt, r.SeatsTotal, r.SeatsAvailable, json.Number(fmt.Sprintf("%d.%02d", r.PriceCents/100, r.PriceCents%100)), r.Notes, r.Status, r.CreatedAt}
}
