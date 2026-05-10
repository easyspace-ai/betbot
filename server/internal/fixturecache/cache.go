package fixturecache

import "sync"

// Global holds SX fixture rows keyed by sxEventId (dashboard `FixtureState` JSON shape).
var Global = &Cache{byID: make(map[string]map[string]any)}

type Cache struct {
	mu   sync.RWMutex
	byID map[string]map[string]any
}

func (c *Cache) Put(id string, state map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byID[id] = state
}

func (c *Cache) Get(id string) (map[string]any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.byID[id]
	return v, ok
}

// All returns a snapshot of every cached fixture (for WS `fixtureSnapshot`).
func (c *Cache) All() []map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]map[string]any, 0, len(c.byID))
	for _, v := range c.byID {
		out = append(out, v)
	}
	return out
}
