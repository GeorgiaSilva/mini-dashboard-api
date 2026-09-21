package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"nivelador-api/internal/models"
)

// Repository centraliza os arquivos JSON. A trava protege leituras e escrita dentro deste processo.
type Repository struct {
	dir string
	mu  sync.RWMutex
}

func New(dir string) *Repository { return &Repository{dir: dir} }

func readFile[T any](path string) ([]T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result []T
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}
func writeFile[T any](path string, value []T) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(b); err == nil {
		err = tmp.Close()
	} else {
		tmp.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
func (r *Repository) Users() ([]models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return readFile[models.User](filepath.Join(r.dir, "users.json"))
}
func (r *Repository) Sales() ([]models.Sale, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return readFile[models.Sale](filepath.Join(r.dir, "sales.json"))
}
func (r *Repository) UpdateUsers(fn func([]models.User) ([]models.User, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := filepath.Join(r.dir, "users.json")
	all, e := readFile[models.User](p)
	if e != nil {
		return e
	}
	all, e = fn(all)
	if e != nil {
		return e
	}
	return writeFile(p, all)
}

var ErrNotFound = errors.New("not found")
