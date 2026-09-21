package handlers

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"nivelador-api/internal/middleware"
	"nivelador-api/internal/models"
	"nivelador-api/internal/repository"
)

type AuthHandler struct {
	Repo   *repository.Repository
	Secret []byte
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CPF      string `json:"cpf"`
		Password string `json:"password"`
	}
	if Decode(r, &in) != nil || !models.ValidCPF(in.CPF) || in.Password == "" {
		BadRequest(w, "CPF válido e senha são obrigatórios.")
		return
	}
	users, e := h.Repo.Users()
	if e != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	for _, u := range users {
		if models.NormalizeCPF(u.CPF) == models.NormalizeCPF(in.CPF) {
			if !u.Active || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
				break
			}
			token, e := middleware.NewToken(h.Secret, u)
			if e != nil {
				middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
				return
			}
			JSON(w, 200, map[string]any{"accessToken": token, "user": u.Public()})
			return
		}
	}
	middleware.Error(w, 401, "INVALID_CREDENTIALS", "CPF ou senha inválidos.")
}
func (h AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserFromContext(r.Context())
	JSON(w, 200, u.Public())
}
