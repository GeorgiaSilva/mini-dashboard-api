package models

import (
	"strings"
	"time"
)

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
	CPF          string    `json:"cpf"`
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
	CPF       string    `json:"cpf"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (u User) Public() PublicUser {
	return PublicUser{ID: u.ID, Name: u.Name, CPF: u.CPF, Email: u.Email, Role: u.Role, Active: u.Active, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}
}

func NormalizeCPF(value string) string {
	return strings.NewReplacer(".", "", "-", "", " ", "").Replace(value)
}

func ValidCPF(value string) bool {
	if len(value) != 14 || value[3] != '.' || value[7] != '.' || value[11] != '-' {
		return false
	}
	for i := range value {
		if i == 3 || i == 7 || i == 11 {
			continue
		}
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	cpf := NormalizeCPF(value)
	allSame := true
	for i := range cpf {
		if i > 0 && cpf[i] != cpf[0] {
			allSame = false
		}
	}
	if allSame {
		return false
	}
	calculate := func(size, weight int) byte {
		total := 0
		for i := 0; i < size; i++ {
			total += int(cpf[i]-'0') * (weight - i)
		}
		digit := (total * 10) % 11
		if digit == 10 {
			digit = 0
		}
		return byte(digit) + '0'
	}
	return cpf[9] == calculate(9, 10) && cpf[10] == calculate(10, 11)
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
