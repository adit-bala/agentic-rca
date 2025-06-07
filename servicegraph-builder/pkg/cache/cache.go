package cache

import "time"

type Item struct {
	Value interface{}
	Expiration int64
}

type Cache struct {
	items map[string]Item
}

func New() *Cache {
	return &Cache{
		items: make(map[string]Item),
	}
}

func (c *Cache) Set(key string, value interface{}, expiration int64) {
	c.items[key] = Item{
		Value: value,
		Expiration: expiration + time.Now().Unix(),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	if item.Expiration > 0 && item.Expiration < time.Now().Unix() {
		delete(c.items, key)
		return nil, false
	}
	return item.Value, true
}

func (c *Cache) Delete(key string) {
	delete(c.items, key)
}

func (c *Cache) Clear() {
	c.items = make(map[string]Item)
}
