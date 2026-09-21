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
func loginToken(t *testing.T, app http.Handler, cpf, password string) string {
	t.Helper()
	w := call(t, app, "POST", "/auth/login", "", `{"cpf":"`+cpf+`","password":"`+password+`"}`)
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
	w := call(t, testApp(t), "POST", "/auth/login", "", `{"cpf":"529.982.247-25","password":"123456"}`)
	if w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestLoginIncorrectPassword(t *testing.T) {
	w := call(t, testApp(t), "POST", "/auth/login", "", `{"cpf":"529.982.247-25","password":"errada"}`)
	if w.Code != 401 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestLoginRejectsUnformattedCPF(t *testing.T) {
	w := call(t, testApp(t), "POST", "/auth/login", "", `{"cpf":"52998224725","password":"123456"}`)
	if w.Code != 400 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestProtectedWithoutToken(t *testing.T) {
	w := call(t, testApp(t), "GET", "/sales", "", "")
	if w.Code != 401 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestDirectorCannotAccessUsers(t *testing.T) {
	app := testApp(t)
	w := call(t, app, "GET", "/users", loginToken(t, app, "935.411.347-80", "123456"), "")
	if w.Code != 403 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestSalesFilter(t *testing.T) {
	app := testApp(t)
	w := call(t, app, "GET", "/sales?brand=VISA&status=APPROVED", loginToken(t, app, "529.982.247-25", "123456"), "")
	if w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
	body := w.Body.Bytes()
	var x struct {
		Summary struct {
			TotalSales int `json:"totalSales"`
		} `json:"summary"`
	}
	json.Unmarshal(body, &x)
	if x.Summary.TotalSales == 0 {
		t.Fatal("expected filtered sales")
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	if _, exists := raw["pagination"]; exists {
		t.Fatal("sales response must not contain pagination")
	}
}
func TestCreateAndUpdateUser(t *testing.T) {
	app := testApp(t)
	jwt := loginToken(t, app, "529.982.247-25", "123456")
	w := call(t, app, "POST", "/users", jwt, `{"name":"Novo Usuário","cpf":"123.456.789-09","email":"novo@empresa.com","password":"123456","role":"DIRECTOR"}`)
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
	w := call(t, app, "POST", "/users", loginToken(t, app, "529.982.247-25", "123456"), `{"name":"Outro","cpf":"123.456.789-09","email":"dev@empresa.com","password":"123456","role":"DIRECTOR"}`)
	if w.Code != 409 {
		t.Fatalf("got %d", w.Code)
	}
}
func TestChangePasswordReturnsSuccessMessage(t *testing.T) {
	app := testApp(t)
	jwt := loginToken(t, app, "529.982.247-25", "123456")
	w := call(t, app, "PATCH", "/users/2/password", jwt, `{"password":"654321"}`)
	if w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "Senha alterada com sucesso." {
		t.Fatalf("unexpected message: %q", body.Message)
	}
	if w = call(t, app, "POST", "/auth/login", "", `{"cpf":"111.444.777-35","password":"654321"}`); w.Code != 200 {
		t.Fatalf("new password login got %d", w.Code)
	}
}
func TestInactiveUserCannotLogin(t *testing.T) {
	app := testApp(t)
	jwt := loginToken(t, app, "529.982.247-25", "123456")
	w := call(t, app, "DELETE", "/users/2", jwt, "")
	if w.Code != 204 {
		t.Fatalf("disable got %d", w.Code)
	}
	w = call(t, app, "POST", "/auth/login", "", `{"cpf":"111.444.777-35","password":"123456"}`)
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
