package controllers

import (
	"sync"
	"time"
)

var cache map[string]time.Time
var mu sync.Mutex

func init() {
	cache = make(map[string]time.Time)
}

func rateLimit(name string, every time.Duration, f func()) {
	mu.Lock()
	defer mu.Unlock()

	if last, ok := cache[name]; !ok || time.Since(last) >= every {
		go f()
		cache[name] = time.Now()
	}
}
