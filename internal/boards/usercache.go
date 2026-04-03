package boards

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.yaml.in/yaml/v3"
)

type UserCache struct {
	mu    sync.Mutex
	users map[string]*User // id -> User
	path  string
	dirty bool
	svc   *Service
}

func NewUserCache(svc *Service) *UserCache {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "skate")
	os.MkdirAll(cacheDir, 0o755)

	uc := &UserCache{
		users: make(map[string]*User),
		path:  filepath.Join(cacheDir, "users.yaml"),
		svc:   svc,
	}
	uc.load()
	return uc
}

func (uc *UserCache) load() {
	data, err := os.ReadFile(uc.path)
	if err != nil {
		return
	}
	var cached map[string]*User
	if yaml.Unmarshal(data, &cached) == nil && cached != nil {
		uc.users = cached
	}
}

func (uc *UserCache) save() {
	if !uc.dirty {
		return
	}
	data, err := yaml.Marshal(uc.users)
	if err != nil {
		return
	}
	os.WriteFile(uc.path, data, 0o600)
	uc.dirty = false
}

func (uc *UserCache) Resolve(userID string) string {
	if userID == "" {
		return ""
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	if u, ok := uc.users[userID]; ok {
		return formatUserName(u)
	}

	// Fetch from API
	user, err := uc.svc.GetUser(userID)
	if err != nil {
		return userID[:min(8, len(userID))]
	}

	uc.users[userID] = user
	uc.dirty = true
	uc.save()

	return formatUserName(user)
}

func formatUserName(u *User) string {
	if u.Username != "" {
		return u.Username
	}
	name := u.FirstName
	if u.LastName != "" {
		if name != "" {
			name += " "
		}
		name += u.LastName
	}
	if name != "" {
		return name
	}
	return u.ID[:min(8, len(u.ID))]
}

func (uc *UserCache) Flush() {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.save()
}

// GetUser fetches a user by ID from the API.
func (s *Service) GetUser(userID string) (*User, error) {
	data, err := s.client.Get(fmt.Sprintf("/users/%s", userID))
	if err != nil {
		return nil, err
	}
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("parsing user: %w", err)
	}
	return &user, nil
}
