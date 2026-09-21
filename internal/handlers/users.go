package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"nivelador-api/internal/middleware"
	"nivelador-api/internal/models"
	"nivelador-api/internal/repository"
)

type UsersHandler struct{ Repo *repository.Repository }

func userValid(name, cpf, email, role string) bool {
	return strings.TrimSpace(name) != "" && models.ValidCPF(cpf) && strings.Contains(email, "@") && models.Contains(models.Roles, role)
}
func (h UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	all, e := h.Repo.Users()
	if e != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	q := r.URL.Query()
	search, role := strings.ToLower(q.Get("search")), q.Get("role")
	activeFilter := q.Get("active")
	if role != "" && !models.Contains(models.Roles, role) {
		BadRequest(w, "Perfil inválido.")
		return
	}
	if activeFilter != "" && activeFilter != "true" && activeFilter != "false" {
		BadRequest(w, "Status ativo inválido.")
		return
	}
	out := []models.PublicUser{}
	for _, u := range all {
		if (search == "" || strings.Contains(strings.ToLower(u.Name+" "+u.CPF+" "+u.Email), search)) && (role == "" || u.Role == role) && (activeFilter == "" || (activeFilter == "true") == u.Active) {
			out = append(out, u.Public())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	JSON(w, 200, map[string]any{"items": out})
}
func (h UsersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := ID(r.URL.Path)
	if !ok {
		BadRequest(w, "ID inválido.")
		return
	}
	all, e := h.Repo.Users()
	if e != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	for _, u := range all {
		if u.ID == id {
			JSON(w, 200, u.Public())
			return
		}
	}
	middleware.Error(w, 404, "NOT_FOUND", "Usuário não encontrado.")
}
func (h UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		CPF      string `json:"cpf"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Active   *bool  `json:"active"`
	}
	if Decode(r, &in) != nil || !userValid(in.Name, in.CPF, in.Email, in.Role) || len(in.Password) < 6 {
		BadRequest(w, "Nome, CPF válido, e-mail, senha (mínimo 6) e perfil válidos são obrigatórios.")
		return
	}
	var created models.User
	e := h.Repo.UpdateUsers(func(all []models.User) ([]models.User, error) {
		for _, u := range all {
			if strings.EqualFold(u.Email, in.Email) {
				return nil, errors.New("duplicate")
			}
			if models.NormalizeCPF(u.CPF) == models.NormalizeCPF(in.CPF) {
				return nil, errors.New("duplicate_cpf")
			}
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			return nil, e
		}
		id := 0
		for _, u := range all {
			if u.ID > id {
				id = u.ID
			}
		}
		active := true
		if in.Active != nil {
			active = *in.Active
		}
		now := time.Now().UTC()
		created = models.User{ID: id + 1, Name: strings.TrimSpace(in.Name), CPF: in.CPF, Email: strings.TrimSpace(in.Email), PasswordHash: string(hash), Role: in.Role, Active: active, CreatedAt: now, UpdatedAt: now}
		return append(all, created), nil
	})
	if e != nil {
		if e.Error() == "duplicate" {
			middleware.Error(w, 409, "EMAIL_ALREADY_EXISTS", "E-mail já está em uso.")
		} else if e.Error() == "duplicate_cpf" {
			middleware.Error(w, 409, "CPF_ALREADY_EXISTS", "CPF já está em uso.")
		} else {
			middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		}
		return
	}
	JSON(w, 201, created.Public())
}
func (h UsersHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id, ok := ID(r.URL.Path)
	if !ok {
		BadRequest(w, "ID inválido.")
		return
	}
	var in struct {
		Name   *string `json:"name"`
		CPF    *string `json:"cpf"`
		Email  *string `json:"email"`
		Role   *string `json:"role"`
		Active *bool   `json:"active"`
	}
	if Decode(r, &in) != nil || (in.Name == nil && in.CPF == nil && in.Email == nil && in.Role == nil && in.Active == nil) {
		BadRequest(w, "Informe ao menos um campo para atualizar.")
		return
	}
	if (in.Name != nil && strings.TrimSpace(*in.Name) == "") || (in.CPF != nil && !models.ValidCPF(*in.CPF)) || (in.Email != nil && !strings.Contains(*in.Email, "@")) || (in.Role != nil && !models.Contains(models.Roles, *in.Role)) {
		BadRequest(w, "Dados de usuário inválidos.")
		return
	}
	caller, _ := middleware.UserFromContext(r.Context())
	var updated models.User
	e := h.Repo.UpdateUsers(func(all []models.User) ([]models.User, error) {
		found := false
		for i := range all {
			if all[i].ID != id {
				continue
			}
			found = true
			if in.Email != nil {
				for _, u := range all {
					if u.ID != id && strings.EqualFold(u.Email, *in.Email) {
						return nil, errors.New("duplicate")
					}
				}
				all[i].Email = strings.TrimSpace(*in.Email)
			}
			if in.CPF != nil {
				for _, u := range all {
					if u.ID != id && models.NormalizeCPF(u.CPF) == models.NormalizeCPF(*in.CPF) {
						return nil, errors.New("duplicate_cpf")
					}
				}
				all[i].CPF = *in.CPF
			}
			if in.Name != nil {
				all[i].Name = strings.TrimSpace(*in.Name)
			}
			if in.Role != nil {
				all[i].Role = *in.Role
			}
			if in.Active != nil {
				if caller.ID == id && !*in.Active {
					return nil, errors.New("self")
				}
				all[i].Active = *in.Active
			}
			all[i].UpdatedAt = time.Now().UTC()
			updated = all[i]
		}
		if !found {
			return nil, repository.ErrNotFound
		}
		return all, nil
	})
	if e != nil {
		switch e {
		case repository.ErrNotFound:
			middleware.Error(w, 404, "NOT_FOUND", "Usuário não encontrado.")
		case errors.New("duplicate"):
			middleware.Error(w, 409, "EMAIL_ALREADY_EXISTS", "E-mail já está em uso.")
		default:
			if e.Error() == "duplicate" {
				middleware.Error(w, 409, "EMAIL_ALREADY_EXISTS", "E-mail já está em uso.")
			} else if e.Error() == "duplicate_cpf" {
				middleware.Error(w, 409, "CPF_ALREADY_EXISTS", "CPF já está em uso.")
			} else if e.Error() == "self" {
				middleware.Error(w, 400, "SELF_DEACTIVATION", "Você não pode desativar sua própria conta.")
			} else {
				middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
			}
		}
		return
	}
	JSON(w, 200, updated.Public())
}
func (h UsersHandler) Password(w http.ResponseWriter, r *http.Request) {
	id, ok := ID(strings.TrimSuffix(r.URL.Path, "/password"))
	if !ok {
		BadRequest(w, "ID inválido.")
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if Decode(r, &in) != nil || len(in.Password) < 6 {
		BadRequest(w, "Senha deve ter pelo menos 6 caracteres.")
		return
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if e != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	e = h.Repo.UpdateUsers(func(all []models.User) ([]models.User, error) {
		for i := range all {
			if all[i].ID == id {
				all[i].PasswordHash = string(hash)
				all[i].UpdatedAt = time.Now().UTC()
				return all, nil
			}
		}
		return nil, repository.ErrNotFound
	})
	if e == repository.ErrNotFound {
		middleware.Error(w, 404, "NOT_FOUND", "Usuário não encontrado.")
		return
	}
	if e != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	w.WriteHeader(204)
}
func (h UsersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := ID(r.URL.Path)
	if !ok {
		BadRequest(w, "ID inválido.")
		return
	}
	caller, _ := middleware.UserFromContext(r.Context())
	e := h.Repo.UpdateUsers(func(all []models.User) ([]models.User, error) {
		for i := range all {
			if all[i].ID == id {
				if id == caller.ID {
					return nil, errors.New("self")
				}
				all[i].Active = false
				all[i].UpdatedAt = time.Now().UTC()
				return all, nil
			}
		}
		return nil, repository.ErrNotFound
	})
	if e == repository.ErrNotFound {
		middleware.Error(w, 404, "NOT_FOUND", "Usuário não encontrado.")
		return
	}
	if e != nil {
		if e.Error() == "self" {
			BadRequest(w, "Você não pode desativar sua própria conta.")
		} else {
			middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		}
		return
	}
	w.WriteHeader(204)
}
