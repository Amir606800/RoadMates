package v1

import (
	"net/http"

	"roadmates/backend/internal/apperrors"
	"roadmates/backend/internal/httpjson"
)

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		apperrors.Write(w, r, apperrors.ErrMethodNotAllowed)
		return
	}
	httpjson.Write(w, http.StatusOK, IndexResponse{APIVersion: "v1"})
}
