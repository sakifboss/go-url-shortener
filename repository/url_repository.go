package repository

import (
	"sync"

	"goshort/model"
)

// URLRepository defines the storage operations required by the service.
// Keeping storage behind an interface lets the service remain independent
// of the actual storage implementation.
type URLRepository interface {
	Save(url model.URL) error
	FindByShortCode(shortCode string) (model.URL, bool)
	Exists(shortCode string) bool
}

// MemoryURLRepository stores URL mappings in memory for the MVP.
// A mutex protects the map because HTTP requests can run concurrently.
type MemoryURLRepository struct {
	mu   sync.RWMutex
	data map[string]model.URL
}

// NewMemoryURLRepository creates an empty in-memory repository.
func NewMemoryURLRepository() *MemoryURLRepository {
	return &MemoryURLRepository{
		data: make(map[string]model.URL),
	}
}

// Save stores a URL using its short code as the key.
func (r *MemoryURLRepository) Save(url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[url.ShortCode] = url

	return nil
}

// FindByShortCode retrieves a URL using its short code.
func (r *MemoryURLRepository) FindByShortCode(shortCode string) (model.URL, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, exists := r.data[shortCode]

	return url, exists
}

// Exists checks whether a short code is already stored.
func (r *MemoryURLRepository) Exists(shortCode string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.data[shortCode]

	return exists
}
