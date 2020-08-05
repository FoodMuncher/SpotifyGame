package store

import (
	"fmt"
	"sync"
)

// Store - ...
type Store struct {
	sync.RWMutex
	kvMap map[string]interface{}
}

// NewStore - ..
func NewStore() *Store {
	return &Store{
		kvMap: make(map[string]interface{}),
	}
}

// Add - Adds a session to the store (TODO: Better var names)
func (store *Store) Add(key string, value string) {
	store.Lock()
	store.kvMap[key] = value
	fmt.Println("Add", store.kvMap[key])
	store.Unlock()
}

// Read - Reads
func (store *Store) Read(key string) interface{} {
	store.RLock()
	value := store.kvMap[key]
	fmt.Println("Read:", value)
	store.RUnlock()

	return value
}
