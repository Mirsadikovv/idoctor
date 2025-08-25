package i18n

import (
	"sync"
)

type LanguageCache struct {
	mu       sync.RWMutex
	languages map[int64]string
}

func NewLanguageCache() *LanguageCache {
	return &LanguageCache{
		languages: make(map[int64]string),
	}
}

func (lc *LanguageCache) Set(userID int64, language string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.languages[userID] = language
}

func (lc *LanguageCache) Get(userID int64) string {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	if lang, exists := lc.languages[userID]; exists {
		return lang
	}
	return "ru" // Русский по умолчанию
}

func (lc *LanguageCache) Delete(userID int64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.languages, userID)
}

func (lc *LanguageCache) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.languages = make(map[int64]string)
}