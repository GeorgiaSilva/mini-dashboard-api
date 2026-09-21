package handlers

import (
	"net/http"
	"nivelador-api/internal/middleware"
	"nivelador-api/internal/repository"
	"sort"
)

type DashboardHandler struct{ Repo *repository.Repository }

func (h DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	all, e := h.Repo.Sales()
	if e != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	all, ok := saleFilter(r, all)
	if !ok {
		BadRequest(w, "Filtros de período inválidos.")
		return
	}
	gross := 0.0
	count := 0
	byDay := map[string]map[string]any{}
	byBrand := map[string]map[string]any{}
	byStatus := map[string]int{}
	for _, s := range all {
		byStatus[s.Status]++
		if s.Status != "APPROVED" {
			continue
		}
		gross += s.Amount
		count++
		d := s.CreatedAt.Format("2006-01-02")
		if byDay[d] == nil {
			byDay[d] = map[string]any{"date": d, "amount": 0.0, "quantity": 0}
		}
		byDay[d]["amount"] = byDay[d]["amount"].(float64) + s.Amount
		byDay[d]["quantity"] = byDay[d]["quantity"].(int) + 1
		if byBrand[s.Brand] == nil {
			byBrand[s.Brand] = map[string]any{"brand": s.Brand, "amount": 0.0, "quantity": 0, "percentage": 0.0}
		}
		byBrand[s.Brand]["amount"] = byBrand[s.Brand]["amount"].(float64) + s.Amount
		byBrand[s.Brand]["quantity"] = byBrand[s.Brand]["quantity"].(int) + 1
	}
	days := []map[string]any{}
	for _, v := range byDay {
		v["amount"] = round(v["amount"].(float64))
		days = append(days, v)
	}
	sort.Slice(days, func(i, j int) bool { return days[i]["date"].(string) < days[j]["date"].(string) })
	brands := []map[string]any{}
	for _, brand := range []string{"VISA", "MASTERCARD", "ELO", "AMEX"} {
		if v := byBrand[brand]; v != nil {
			if gross > 0 {
				v["percentage"] = round(v["amount"].(float64) * 100 / gross)
			}
			v["amount"] = round(v["amount"].(float64))
			brands = append(brands, v)
		}
	}
	statuses := []map[string]any{}
	for _, status := range []string{"PENDING", "APPROVED", "CANCELLED", "REFUNDED"} {
		if q := byStatus[status]; q > 0 {
			statuses = append(statuses, map[string]any{"status": status, "quantity": q})
		}
	}
	average := 0.0
	if count > 0 {
		average = round(gross / float64(count))
	}
	JSON(w, 200, map[string]any{"cards": map[string]any{"grossRevenue": round(gross), "netRevenue": round(gross * .9), "salesCount": count, "averageTicket": average}, "salesByDay": days, "salesByBrand": brands, "salesByStatus": statuses})
}
