package apperrors

import (
	"errors"
	"net/http"

	"roadmates/backend/internal/httpjson"
	"roadmates/backend/internal/requestmeta"
)

var (
	ErrDriverHasActiveRide   = errors.New("finish or cancel your active ride before publishing another")
	ErrRideNotActive         = errors.New("this ride has already been completed or cancelled")
	ErrNotFound              = errors.New("resource not found")
	ErrAPIVersionNotFound    = errors.New("API version not found")
	ErrMethodNotAllowed      = errors.New("method not allowed")
	ErrAppVersionInvalid     = errors.New("X-App-Version must be a full semantic version")
	ErrAppVersionUnsupported = errors.New("please update or reload the app")
	ErrInternal              = errors.New("an internal error occurred")
	ErrUnauthorized          = errors.New("authentication is required")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrRefreshReused         = errors.New("session token was reused; sign in again")
	ErrForbidden             = errors.New("you do not have permission for this action")
	ErrCSRF                  = errors.New("request origin or CSRF header is invalid")
	ErrInvalidInput          = errors.New("check the request fields and try again")
	ErrPasswordPolicy        = errors.New("password must be at least 8 characters, at most 72 UTF-8 bytes, with uppercase, lowercase, and a special character; spaces are not allowed")
	ErrEmailTaken            = errors.New("an account with that email already exists")
	ErrRateLimited           = errors.New("too many requests; try again later")
	ErrAlreadyDriver         = errors.New("you are already a verified driver")
	ErrApplicationPending    = errors.New("you already have an application under review")
	ErrApplicationReviewed   = errors.New("this application has already been reviewed")
	ErrApplicantInactive     = errors.New("this applicant account is inactive")
)

type Response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", ErrInternal.Error()
	switch {
	case errors.Is(err, ErrDriverHasActiveRide):
		status, code, message = 409, "DRIVER_HAS_ACTIVE_RIDE", ErrDriverHasActiveRide.Error()
	case errors.Is(err, ErrRideNotActive):
		status, code, message = 409, "RIDE_NOT_ACTIVE", ErrRideNotActive.Error()
	case errors.Is(err, ErrAlreadyDriver):
		status, code, message = 409, "DRIVER_ALREADY_VERIFIED", ErrAlreadyDriver.Error()
	case errors.Is(err, ErrApplicationPending):
		status, code, message = 409, "APPLICATION_PENDING", ErrApplicationPending.Error()
	case errors.Is(err, ErrApplicationReviewed):
		status, code, message = 409, "APPLICATION_ALREADY_REVIEWED", ErrApplicationReviewed.Error()
	case errors.Is(err, ErrApplicantInactive):
		status, code, message = 409, "APPLICANT_INACTIVE", ErrApplicantInactive.Error()
	case errors.Is(err, ErrUnauthorized):
		status, code, message = 401, "UNAUTHORIZED", ErrUnauthorized.Error()
	case errors.Is(err, ErrInvalidCredentials):
		status, code, message = 401, "INVALID_CREDENTIALS", ErrInvalidCredentials.Error()
	case errors.Is(err, ErrRefreshReused):
		status, code, message = 401, "REFRESH_TOKEN_REUSED", ErrRefreshReused.Error()
	case errors.Is(err, ErrForbidden):
		status, code, message = 403, "FORBIDDEN", ErrForbidden.Error()
	case errors.Is(err, ErrCSRF):
		status, code, message = 403, "CSRF_INVALID", ErrCSRF.Error()
	case errors.Is(err, ErrInvalidInput):
		status, code, message = 400, "INVALID_INPUT", ErrInvalidInput.Error()
	case errors.Is(err, ErrPasswordPolicy):
		status, code, message = 400, "PASSWORD_POLICY", ErrPasswordPolicy.Error()
	case errors.Is(err, ErrEmailTaken):
		status, code, message = 409, "EMAIL_TAKEN", ErrEmailTaken.Error()
	case errors.Is(err, ErrRateLimited):
		status, code, message = 429, "RATE_LIMITED", ErrRateLimited.Error()
	case errors.Is(err, ErrNotFound):
		status, code, message = 404, "NOT_FOUND", ErrNotFound.Error()
	case errors.Is(err, ErrAPIVersionNotFound):
		status, code, message = 404, "API_VERSION_NOT_FOUND", ErrAPIVersionNotFound.Error()
	case errors.Is(err, ErrMethodNotAllowed):
		status, code, message = 405, "METHOD_NOT_ALLOWED", ErrMethodNotAllowed.Error()
	case errors.Is(err, ErrAppVersionInvalid):
		status, code, message = 400, "APP_VERSION_INVALID", ErrAppVersionInvalid.Error()
	case errors.Is(err, ErrAppVersionUnsupported):
		status, code, message = 426, "APP_VERSION_UNSUPPORTED", ErrAppVersionUnsupported.Error()
	}
	httpjson.Write(w, status, Response{code, message, requestmeta.From(r.Context()).ID})
}
