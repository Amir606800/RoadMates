package v1

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/httpjson"
	"roadmates/backend/internal/service"
	"roadmates/backend/internal/validation"
)

type rideHandlers struct {
	rides     *service.Rides
	districts *service.Districts
}

func (h rideHandlers) listDistricts(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if !validation.Text(query, 80, false) {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	items, err := h.districts.List(r.Context(), query)
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	result := make([]DistrictResponse, 0, len(items))
	for _, item := range items {
		result = append(result, districtResponse(item))
	}
	httpjson.Write(w, 200, map[string]any{"districts": result})
}

var pricePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,5})(\.[0-9]{1,2})?$`)

// Parse money as integer cents; PostgreSQL must never round an invalid price.
func priceCents(value string) (int64, bool) {
	if !pricePattern.MatchString(value) {
		return 0, false
	}
	parts := strings.Split(value, ".")
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	fraction := int64(0)
	if len(parts) == 2 {
		fraction, _ = strconv.ParseInt((parts[1] + "0")[:2], 10, 64)
	}
	return whole*100 + fraction, true
}

func (h rideHandlers) publish(w http.ResponseWriter, r *http.Request) {
	var input PublishRideRequest
	if err := decodeBody(w, r, &input); err != nil {
		apperrors.Write(w, r, err)
		return
	}
	input.Notes = strings.TrimSpace(input.Notes)
	cents, valid := priceCents(string(input.PriceAZN))
	if !valid || input.FromID < 1 || input.FromID > 32767 || input.ToID < 1 || input.ToID > 32767 || input.FromID == input.ToID || !input.DepartureAt.After(time.Now()) || input.DepartureAt.Year() > 9999 || input.SeatsTotal < 1 || input.SeatsTotal > 8 || !validation.Text(input.Notes, 300, true) {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	ride, err := h.rides.Publish(r.Context(), domain.Ride{DriverID: auth.Current(r.Context()).UserID, From: domain.District{ID: input.FromID}, To: domain.District{ID: input.ToID}, DepartureAt: input.DepartureAt.UTC(), SeatsTotal: input.SeatsTotal, PriceCents: cents, Notes: input.Notes})
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, 201, rideResponse(ride))
}
func (h rideHandlers) mine(w http.ResponseWriter, r *http.Request) {
	ride, err := h.rides.Active(r.Context(), auth.Current(r.Context()).UserID)
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, 200, rideResponse(ride))
}
func districtFilter(raw string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	id, err := strconv.Atoi(raw)
	return id, err == nil && id > 0 && id <= 32767
}
func (h rideHandlers) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	from, validFrom := districtFilter(query.Get("from_district_id"))
	to, validTo := districtFilter(query.Get("to_district_id"))
	date := query.Get("date")
	parsedDate, dateErr := time.Parse("2006-01-02", date)
	if !validFrom || !validTo || (from != 0 && from == to) || (date != "" && (dateErr != nil || len(date) != 10 || parsedDate.Year() < 1)) {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	rides, err := h.rides.Search(r.Context(), domain.RideSearch{FromID: from, ToID: to, Date: date})
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	items := make([]RideResponse, 0, len(rides))
	for _, ride := range rides {
		items = append(items, rideResponse(ride))
	}
	httpjson.Write(w, 200, map[string]any{"rides": items})
}
func (h rideHandlers) detail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 36 || uuid.Validate(id) != nil {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	ride, err := h.rides.Detail(r.Context(), id)
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, 200, rideResponse(ride))
}
func (h rideHandlers) finish(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if len(id) != 36 || uuid.Validate(id) != nil {
			apperrors.Write(w, r, apperrors.ErrInvalidInput)
			return
		}
		ride, err := h.rides.Finish(r.Context(), id, auth.Current(r.Context()).UserID, status)
		if err != nil {
			apperrors.Write(w, r, err)
			return
		}
		httpjson.Write(w, 200, rideResponse(ride))
	}
}
