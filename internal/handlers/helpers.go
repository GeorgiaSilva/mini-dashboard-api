package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nivelador-api/internal/middleware"
)

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func Decode(r *http.Request, target any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(target)
}
func ID(path string) (int, bool) {
	p := strings.Trim(path, "/")
	parts := strings.Split(p, "/")
	if len(parts) == 0 {
		return 0, false
	}
	n, e := strconv.Atoi(parts[len(parts)-1])
	return n, e == nil && n > 0
}
func Date(q string) (time.Time, bool) {
	if q == "" {
		return time.Time{}, true
	}
	d, e := time.Parse("2006-01-02", q)
	return d, e == nil
}
func BadRequest(w http.ResponseWriter, msg string) {
	middleware.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", msg)
}
