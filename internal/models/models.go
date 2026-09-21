package models

import "time"

const (
	RoleDeveloper = "DEVELOPER"
	RoleDirector  = "DIRECTOR"
)

var Roles = []string{RoleDeveloper, RoleDirector}
var Brands = []string{"VISA", "MASTERCARD", "ELO", "AMEX"}
var SaleStatuses = []string{"PENDING", "APPROVED", "CANCELLED", "REFUNDED"}

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"passwordHash"`
	Role         string    `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type PublicUser struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (u User) Public() PublicUser {
	return PublicUser{u.ID, u.Name, u.Email, u.Role, u.Active, u.CreatedAt, u.UpdatedAt}
}

type PublicEntity struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type Sale struct {
	ID              int          `json:"id"`
	TransactionCode string       `json:"transactionCode"`
	Amount          float64      `json:"amount"`
	Brand           string       `json:"brand"`
	PublicEntity    PublicEntity `json:"publicEntity"`
	Status          string       `json:"status"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}
type SaleView struct {
	ID              int          `json:"id"`
	TransactionCode string       `json:"transactionCode"`
	Amount          float64      `json:"amount"`
	Brand           string       `json:"brand"`
	PublicEntity    PublicEntity `json:"publicEntity"`
	Status          string       `json:"status"`
	CreatedAt       time.Time    `json:"createdAt"`
}

func (s Sale) View() SaleView {
	return SaleView{s.ID, s.TransactionCode, s.Amount, s.Brand, s.PublicEntity, s.Status, s.CreatedAt}
}

func Contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
