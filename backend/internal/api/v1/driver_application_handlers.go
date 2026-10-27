package v1

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/httpjson"
	"roadmates/backend/internal/service"
	"roadmates/backend/internal/validation"
)

type driverApplicationHandlers struct{ service *service.DriverApplications }

func (h driverApplicationHandlers) apply(w http.ResponseWriter, r *http.Request) {
	var input DriverApplicationRequest
	if err := decodeBody(w, r, &input); err != nil {
		apperrors.Write(w, r, err)
		return
	}
	input.ContactPhone = strings.TrimSpace(input.ContactPhone)
	input.FirstName, input.LastName = strings.TrimSpace(input.FirstName), strings.TrimSpace(input.LastName)
	input.PINCode = strings.ToUpper(strings.TrimSpace(input.PINCode))
	input.CarMake, input.CarModel, input.CarColor = strings.TrimSpace(input.CarMake), strings.TrimSpace(input.CarModel), strings.TrimSpace(input.CarColor)
	input.CarPlate = strings.ToUpper(strings.TrimSpace(input.CarPlate))
	if !validation.Phone(input.ContactPhone) || !validation.Name(input.FirstName) || !validation.Name(input.LastName) || !validation.PIN(input.PINCode) || !validation.Plate(input.CarPlate) || input.CarYear < 1980 || input.CarYear > time.Now().UTC().Year()+1 || input.Seats < 1 || input.Seats > 8 {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	for _, value := range []string{input.CarMake, input.CarModel, input.CarColor} {
		if value == "" || !validation.Text(value, 50, false) {
			apperrors.Write(w, r, apperrors.ErrInvalidInput)
			return
		}
	}
	app, err := h.service.Apply(r.Context(), auth.Current(r.Context()).UserID, service.DriverApplicationInput{ContactPhone: input.ContactPhone, FirstName: input.FirstName, LastName: input.LastName, PINCode: input.PINCode, CarMake: input.CarMake, CarModel: input.CarModel, CarYear: input.CarYear, CarPlate: input.CarPlate, CarColor: input.CarColor, Seats: input.Seats})
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, applicationResponse(app))
}

func (h driverApplicationHandlers) mine(w http.ResponseWriter, r *http.Request) {
	app, err := h.service.Latest(r.Context(), auth.Current(r.Context()).UserID)
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, applicationResponse(app))
}

func (h driverApplicationHandlers) list(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "approved" && status != "rejected" {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	limit, offset := 20, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			apperrors.Write(w, r, apperrors.ErrInvalidInput)
			return
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 || offset > 100000 {
			apperrors.Write(w, r, apperrors.ErrInvalidInput)
			return
		}
	}
	apps, more, err := h.service.List(r.Context(), status, limit, offset)
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	items := make([]DriverApplicationResponse, 0, len(apps))
	for _, app := range apps {
		items = append(items, applicationResponse(app))
	}
	httpjson.Write(w, http.StatusOK, ApplicationListResponse{items, ApplicationPagination{limit, offset, more}})
}

func validApplicationID(r *http.Request) bool {
	id := r.PathValue("id")
	return len(id) == 36 && uuid.Validate(id) == nil
}

func (h driverApplicationHandlers) detail(w http.ResponseWriter, r *http.Request) {
	if !validApplicationID(r) {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	app, pin, err := h.service.Detail(r.Context(), r.PathValue("id"))
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, AdminApplicationDetail{applicationResponse(app), pin})
}

func (h driverApplicationHandlers) review(w http.ResponseWriter, r *http.Request) {
	if !validApplicationID(r) {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	var input ReviewApplicationRequest
	if err := decodeBody(w, r, &input); err != nil {
		apperrors.Write(w, r, err)
		return
	}
	input.Action, input.Reason = strings.TrimSpace(input.Action), strings.TrimSpace(input.Reason)
	if (input.Action != "approve" && input.Action != "reject") || !validation.Text(input.Reason, 500, true) {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	app, err := h.service.Review(r.Context(), r.PathValue("id"), auth.Current(r.Context()).UserID, input.Action, input.Reason)
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, applicationResponse(app))
}

func (h driverApplicationHandlers) adminApplication(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.detail(w, r)
	case http.MethodPatch:
		h.review(w, r)
	default:
		w.Header().Set("Allow", "GET, HEAD, PATCH")
		apperrors.Write(w, r, apperrors.ErrMethodNotAllowed)
	}
}
