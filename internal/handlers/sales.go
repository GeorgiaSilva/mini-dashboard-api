package handlers

import (
	"net/http"
	"sort"
	"time"

	"nivelador-api/internal/middleware"
	"nivelador-api/internal/models"
	"nivelador-api/internal/repository"
)

// SalesHandler expõe somente consultas: os dados vêm do produto de pagamentos.
type SalesHandler struct{ Repo *repository.Repository }

func saleFilter(r *http.Request, all []models.Sale) ([]models.Sale, bool) {
	q := r.URL.Query()
	start, ok := Date(q.Get("startDate"))
	if !ok {
		return nil, false
	}
	end, ok := Date(q.Get("endDate"))
	if !ok || (!start.IsZero() && !end.IsZero() && end.Before(start)) {
		return nil, false
	}
	brand, status := q.Get("brand"), q.Get("status")
	if (brand != "" && !models.Contains(models.Brands, brand)) || (status != "" && !models.Contains(models.SaleStatuses, status)) {
		return nil, false
	}
	entity := 0
	if q.Get("publicEntityId") != "" {
		entity, ok = atoi(q.Get("publicEntityId"))
		if !ok || entity < 1 {
			return nil, false
		}
	}
	out := []models.Sale{}
	for _, s := range all {
		d := day(s.CreatedAt)
		if (!start.IsZero() && d.Before(start)) || (!end.IsZero() && d.After(end)) || (brand != "" && s.Brand != brand) || (status != "" && s.Status != status) || (entity > 0 && s.PublicEntity.ID != entity) {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, true
}

func atoi(s string) (int, bool) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func (h SalesHandler) List(w http.ResponseWriter, r *http.Request) {
	all, err := h.Repo.Sales()
	if err != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	all, ok := saleFilter(r, all)
	if !ok {
		BadRequest(w, "Filtros de vendas inválidos.")
		return
	}
	total := 0.0
	approved, cancelled := 0, 0
	for _, sale := range all {
		total += sale.Amount
		if sale.Status == "APPROVED" {
			approved++
		}
		if sale.Status == "CANCELLED" {
			cancelled++
		}
	}
	groups := []map[string]any{}
	for _, sale := range all {
		date := sale.CreatedAt.Format("2006-01-02")
		if len(groups) == 0 || groups[len(groups)-1]["date"] != date {
			groups = append(groups, map[string]any{"date": date, "totalAmount": 0.0, "totalSales": 0, "sales": []models.SaleView{}})
		}
		group := groups[len(groups)-1]
		group["totalAmount"] = round(group["totalAmount"].(float64) + sale.Amount)
		group["totalSales"] = group["totalSales"].(int) + 1
		group["sales"] = append(group["sales"].([]models.SaleView), sale.View())
	}
	JSON(w, 200, map[string]any{
		"summary": map[string]any{"totalAmount": round(total), "totalSales": len(all), "approvedSales": approved, "cancelledSales": cancelled},
		"groups":  groups,
	})
}

func (h SalesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := ID(r.URL.Path)
	if !ok {
		BadRequest(w, "ID inválido.")
		return
	}
	all, err := h.Repo.Sales()
	if err != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	for _, sale := range all {
		if sale.ID == id {
			JSON(w, 200, sale)
			return
		}
	}
	middleware.Error(w, 404, "NOT_FOUND", "Venda não encontrada.")
}

func (h SalesHandler) Filters(w http.ResponseWriter, r *http.Request) {
	all, err := h.Repo.Sales()
	if err != nil {
		middleware.Error(w, 500, "INTERNAL_ERROR", "Não foi possível processar a solicitação.")
		return
	}
	seen := map[int]bool{}
	entities := []models.PublicEntity{}
	for _, sale := range all {
		if !seen[sale.PublicEntity.ID] {
			seen[sale.PublicEntity.ID] = true
			entities = append(entities, sale.PublicEntity)
		}
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].ID < entities[j].ID })
	JSON(w, 200, map[string]any{"brands": models.Brands, "statuses": models.SaleStatuses, "publicEntities": entities})
}

func day(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
func round(n float64) float64   { return float64(int(n*100+0.5)) / 100 }
