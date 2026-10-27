package v1

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/auth"
	"roadmates/backend/internal/config"
	"roadmates/backend/internal/domain"
	"roadmates/backend/internal/httpjson"
	"roadmates/backend/internal/requestmeta"
	"roadmates/backend/internal/service"
	"roadmates/backend/internal/validation"
)

const refreshCookieName = "roadmates_refresh"
const refreshCookiePath = "/api/v1/auth"

type authHandlers struct {
	service *service.Auth
	config  config.Config
}

func decodeBody(w http.ResponseWriter, r *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return apperrors.ErrInvalidInput
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return apperrors.ErrInvalidInput
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperrors.ErrInvalidInput
	}
	return nil
}

func (h authHandlers) register(w http.ResponseWriter, r *http.Request) {
	var input RegisterRequest
	if err := decodeBody(w, r, &input); err != nil {
		apperrors.Write(w, r, err)
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.FirstName, input.LastName, input.Phone = strings.TrimSpace(input.FirstName), strings.TrimSpace(input.LastName), strings.TrimSpace(input.Phone)
	if !validation.Email(input.Email) || !validation.Name(input.FirstName) || !validation.Name(input.LastName) || !validation.Phone(input.Phone) {
		apperrors.Write(w, r, apperrors.ErrInvalidInput)
		return
	}
	if err := auth.ValidatePassword(input.Password); err != nil {
		apperrors.Write(w, r, err)
		return
	}
	session, err := h.service.Register(r.Context(), service.Registration{Email: input.Email, Password: input.Password, FirstName: input.FirstName, LastName: input.LastName, Phone: input.Phone}, h.metadata(r))
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	h.session(w, r, session, http.StatusCreated)
}

func (h authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var input LoginRequest
	if err := decodeBody(w, r, &input); err != nil {
		apperrors.Write(w, r, err)
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if !validation.Email(input.Email) || len(input.Password) == 0 || len(input.Password) > 72 {
		apperrors.Write(w, r, apperrors.ErrInvalidCredentials)
		return
	}
	session, err := h.service.Login(r.Context(), input.Email, input.Password, h.metadata(r))
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	h.session(w, r, session, http.StatusOK)
}

func (h authHandlers) refresh(w http.ResponseWriter, r *http.Request) {
	raw := readRefresh(r)
	if raw == "" {
		h.clearCookie(w)
		apperrors.Write(w, r, apperrors.ErrUnauthorized)
		return
	}
	session, err := h.service.Refresh(r.Context(), raw, h.metadata(r))
	if err != nil {
		if errors.Is(err, apperrors.ErrUnauthorized) || errors.Is(err, apperrors.ErrRefreshReused) {
			h.clearCookie(w)
		}
		apperrors.Write(w, r, err)
		return
	}
	h.session(w, r, session, http.StatusOK)
}

func (h authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(r.Context(), readRefresh(r)); err != nil {
		apperrors.Write(w, r, err)
		return
	}
	h.clearCookie(w)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandlers) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Me(r.Context(), auth.Current(r.Context()).UserID)
	if err != nil {
		apperrors.Write(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, userResponse(user))
}

func (h authHandlers) session(w http.ResponseWriter, r *http.Request, session domain.Session, status int) {
	requestmeta.From(r.Context()).UserID = session.User.ID
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: session.RefreshToken, Path: refreshCookiePath, HttpOnly: true, Secure: h.config.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(h.config.RefreshTokenTTL / time.Second), Expires: time.Now().Add(h.config.RefreshTokenTTL)})
	httpjson.Write(w, status, SessionResponse{session.AccessToken, userResponse(session.User)})
}

func (h authHandlers) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: "", Path: refreshCookiePath, HttpOnly: true, Secure: h.config.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}

func readRefresh(r *http.Request) string {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || len(cookie.Value) != 43 {
		return ""
	}
	bytes, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(bytes) != 32 {
		return ""
	}
	return cookie.Value
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if ip := net.ParseIP(r.Header.Get("X-Real-IP")); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return ""
}

func (h authHandlers) metadata(r *http.Request) service.ClientMetadata {
	agent := strings.Map(func(c rune) rune {
		if unicode.IsControl(c) {
			return -1
		}
		return c
	}, r.UserAgent())
	runes := []rune(agent)
	if len(runes) > 512 {
		agent = string(runes[:512])
	}
	return service.ClientMetadata{UserAgent: agent, IP: clientIP(r, h.config.TrustProxyHeaders)}
}
