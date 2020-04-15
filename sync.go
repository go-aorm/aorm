package aorm

import "sync"

type syncedMap struct {
	data map[interface{}]interface{}
	mu   sync.RWMutex
}

func (this *syncedMap) Set(key interface{}, value interface{}) *syncedMap {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.data == nil {
		this.data = map[interface{}]interface{}{}
	}
	this.data[key] = value
	return this
}

func (this *syncedMap) Get(key interface{}) (value interface{}, ok bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.data == nil {
		return
	}
	value, ok = this.data[key]
	return
}
