package main

import (
	"context"
	"log"
	"net/http"
	"nivelador-api/internal/handlers"
	"nivelador-api/internal/middleware"
	"nivelador-api/internal/models"
	"nivelador-api/internal/repository"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	repo := repository.New(dataDir)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "chave-local-apenas-para-desenvolvimento"
		log.Print("JWT_SECRET não definido; usando chave local de desenvolvimento")
	}
	server := newServer(repo, []byte(secret))
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	httpServer := &http.Server{Addr: ":" + addr, Handler: server, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("API disponível em http://localhost:%s", addr)
		if e := httpServer.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatal(e)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e := httpServer.Shutdown(ctx); e != nil {
		log.Printf("erro ao encerrar: %v", e)
	}
}
func newServer(repo *repository.Repository, secret []byte) http.Handler {
	auth := handlers.AuthHandler{Repo: repo, Secret: secret}
	users := handlers.UsersHandler{Repo: repo}
	sales := handlers.SalesHandler{Repo: repo}
	dashboard := handlers.DashboardHandler{Repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { handlers.JSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /auth/login", auth.Login)
	protected := func(roles []string, h http.HandlerFunc) http.Handler {
		return middleware.Auth(repo, secret, middleware.RequireRoles(roles...)(h))
	}
	mux.Handle("GET /auth/me", middleware.Auth(repo, secret, http.HandlerFunc(auth.Me)))
	mux.Handle("GET /dashboard", protected([]string{models.RoleDeveloper, models.RoleDirector}, dashboard.Get))
	mux.Handle("GET /sales/filters", protected([]string{models.RoleDeveloper, models.RoleDirector}, sales.Filters))
	mux.Handle("GET /sales", protected([]string{models.RoleDeveloper, models.RoleDirector}, sales.List))
	mux.Handle("GET /sales/{id}", protected([]string{models.RoleDeveloper, models.RoleDirector}, sales.Get))
	mux.Handle("GET /users", protected([]string{models.RoleDeveloper}, users.List))
	mux.Handle("POST /users", protected([]string{models.RoleDeveloper}, users.Create))
	mux.Handle("GET /users/{id}", protected([]string{models.RoleDeveloper}, users.Get))
	mux.Handle("PATCH /users/{id}", protected([]string{models.RoleDeveloper}, users.Patch))
	mux.Handle("DELETE /users/{id}", protected([]string{models.RoleDeveloper}, users.Delete))
	mux.Handle("PATCH /users/{id}/password", protected([]string{models.RoleDeveloper}, users.Password))
	return cors(mux)
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
