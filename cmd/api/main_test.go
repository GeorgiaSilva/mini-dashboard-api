package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nivelador-api/internal/repository"
)

func testApp(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"users.json", "sales.json"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "data", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return newServer(repository.New(dir), []byte("test-secret"))
}
func call(t *testing.T, app http.Handler, method, path, bearer, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	return w
}
func loginToken(t *testing.T, app http.Handler, email, password string) string {
	t.Helper()
	w := call(t, app, "POST", "/auth/login", "", `{"email":"`+email+`","password":"`+password+`"}`)
	if w.Code != 200 {
		t.Fatalf("login status %d: %s", w.Code, w.Body.String())
	}
	var x struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(w.Body).Decode(&x); err != nil {
		t.Fatal(err)
	}
	return x.AccessToken
}
func TestLoginValidCredentials(t *testing.T) {
	w := call(t, testApp(t), "POST", "/auth/login", "", `{"email":"dev@empresa.com","password":"123456"}`)
	if w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestLoginIncorrectPassword(t *testing.T) {
	w := call(t, testApp(t), "POST", "/auth/login", "", `{"email":"dev@empresa.com","password":"errada"}`)
	if w.Code != 401 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestProtectedWithoutToken(t *testing.T) {
	w := call(t, testApp(t), "GET", "/sales", "", "")
	if w.Code != 401 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestSalesFilter(t *testing.T) {
	app := testApp(t)
	w := call(t, app, "GET", "/sales?brand=VISA&status=APPROVED", loginToken(t, app, "dev@empresa.com", "123456"), "")
	if w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
	var x struct {
		Summary struct {
			TotalSales int `json:"totalSales"`
		} `json:"summary"`
	}
	json.NewDecoder(w.Body).Decode(&x)
	if x.Summary.TotalSales == 0 {
		t.Fatal("expected filtered sales")
	}
}
func TestCreateAndUpdateUser(t *testing.T) {
	app := testApp(t)
	jwt := loginToken(t, app, "dev@empresa.com", "123456")
	w := call(t, app, "POST", "/users", jwt, `{"name":"Novo Usuário","email":"novo@empresa.com","password":"123456","role":"DIRECTOR"}`)
	if w.Code != 201 {
		t.Fatalf("create got %d", w.Code)
	}
	var u struct {
		ID int `json:"id"`
	}
	json.NewDecoder(w.Body).Decode(&u)
	w = call(t, app, "PATCH", "/users/"+itoa(u.ID), jwt, `{"name":"Nome Alterado","active":true}`)
	if w.Code != 200 {
		t.Fatalf("patch got %d", w.Code)
	}
}
func TestDuplicateEmail(t *testing.T) {
	app := testApp(t)
	w := call(t, app, "POST", "/users", loginToken(t, app, "dev@empresa.com", "123456"), `{"name":"Outro","email":"dev@empresa.com","password":"123456","role":"DIRECTOR"}`)
	if w.Code != 409 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestInactiveUserCannotLogin(t *testing.T) {
	app := testApp(t)
	jwt := loginToken(t, app, "dev@empresa.com", "123456")
	w := call(t, app, "DELETE", "/users/2", jwt, "")
	if w.Code != 204 {
		t.Fatalf("disable got %d", w.Code)
	}
	w = call(t, app, "POST", "/auth/login", "", `{"email":"maria.dev@empresa.com","password":"123456"}`)
	if w.Code != 401 {
		t.Fatalf("got %d", w.Code)
	}
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
