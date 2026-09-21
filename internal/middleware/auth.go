package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"nivelador-api/internal/models"
	"nivelador-api/internal/repository"
)

type contextKey string

const userKey contextKey = "authenticated-user"

type Claims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func UserFromContext(ctx context.Context) (models.User, bool) {
	u, ok := ctx.Value(userKey).(models.User)
	return u, ok
}
func Auth(repo *repository.Repository, secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token de acesso ausente ou inválido.")
			return
		}
		token, err := jwt.ParseWithClaims(strings.TrimPrefix(h, "Bearer "), &Claims{}, func(t *jwt.Token) (interface{}, error) { return secret, nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token de acesso ausente, inválido ou expirado.")
			return
		}
		claims := token.Claims.(*Claims)
		id64, err := claims.GetSubject()
		if err != nil {
			Error(w, 401, "UNAUTHORIZED", "Token inválido.")
			return
		}
		var id int
		_, err = fmtSscan(id64, &id)
		if err != nil {
			Error(w, 401, "UNAUTHORIZED", "Token inválido.")
			return
		}
		users, err := repo.Users()
		if err != nil {
			Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
			return
		}
		for _, u := range users {
			if u.ID == id && u.Active {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
				return
			}
		}
		Error(w, 401, "UNAUTHORIZED", "Usuário não encontrado ou inativo.")
	})
}

// variável facilita não expor detalhes de conversão no middleware.
var fmtSscan = func(s string, v *int) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, http.ErrNotSupported
		}
		n = n*10 + int(c-'0')
	}
	*v = n
	return 1, nil
}

func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok {
				Error(w, 401, "UNAUTHORIZED", "Token de acesso ausente ou inválido.")
				return
			}
			if !models.Contains(roles, u.Role) {
				Error(w, 403, "FORBIDDEN", "Seu perfil não possui permissão para este recurso.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func NewToken(secret []byte, u models.User) (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{Name: u.Name, Email: u.Email, Role: u.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: strconvItoa(u.ID), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}}).SignedString(secret)
}

var strconvItoa = func(n int) string {
	if n == 0 {
		return "0"
	}
	out := []byte{}
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	x := errorResponse{}
	x.Error.Code = code
	x.Error.Message = message
	_ = json.NewEncoder(w).Encode(x)
}
