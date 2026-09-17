package repository

import (
	"sync"

	"goshort/model"
)

// URLRepository defines the storage operations required by the service.
// Keeping this contract separate allows the storage implementation
// to change later without changing the service layer.
type URLRepository interface {
	Save(url model.URL) error
	FindByShortCode(shortCode string) (model.URL, bool)
}

// MemoryURLRepository implements URLRepository using an in-memory map.
// This matches the storage requirement for the MVP.
type MemoryURLRepository struct {
	mu   sync.RWMutex
	data map[string]model.URL
}

// NewMemoryURLRepository creates an empty in-memory URL repository.
func NewMemoryURLRepository() *MemoryURLRepository {
	return &MemoryURLRepository{
		data: make(map[string]model.URL),
	}
}

// Save stores a URL using its short code as the lookup key.
func (r *MemoryURLRepository) Save(url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[url.ShortCode] = url

	return nil
}

// FindByShortCode retrieves a stored URL by its short code.
func (r *MemoryURLRepository) FindByShortCode(shortCode string) (model.URL, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, exists := r.data[shortCode]

	return url, exists
}
