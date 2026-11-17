package auth

import (
	"errors"
	"slices"
)

func (s *Store) WorkspaceFavorites(username, role string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user := s.users[username]; user != nil && user.Active && user.Role == role {
		return append([]string{}, s.favorites[user.ID][role]...)
	}
	return []string{}
}

func (s *Store) SetWorkspaceFavorite(username, role, workspace string, starred bool) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.users[username]
	if user == nil || !user.Active || user.Role != role {
		return nil, errors.New("active account required")
	}
	previous := s.favorites[user.ID][role]
	index := slices.Index(previous, workspace)
	if starred == (index >= 0) {
		return append([]string{}, previous...), nil
	}
	next := append([]string{}, previous...)
	if starred {
		if len(next) >= 100 {
			return nil, errors.New("favorite workspace limit reached")
		}
		next = append(next, workspace)
	} else {
		next = slices.Delete(next, index, index+1)
	}
	if s.favorites == nil {
		s.favorites = make(map[string]map[string][]string)
	}
	if s.favorites[user.ID] == nil {
		s.favorites[user.ID] = make(map[string][]string)
	}
	s.favorites[user.ID][role] = next
	if err := s.save(); err != nil {
		s.favorites[user.ID][role] = previous
		return nil, err
	}
	return append([]string{}, next...), nil
}
