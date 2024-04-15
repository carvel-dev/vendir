// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package bundle

import "sync"

// newKeyLock Creates a lock based on a keymap
func newKeyLock() *keyLock {
	return &keyLock{locks: make(map[string]*sync.Mutex)}
}

// keyLock is a collection of mutexes that can be addressed by a key
type keyLock struct {
	locks   map[string]*sync.Mutex
	mapLock sync.Mutex
}

// Lock on the key until there it can proceed
func (l *keyLock) Lock(key string) {
	l.getLockBy(key).Lock()
}

// Unlock on a key
func (l *keyLock) Unlock(key string) {
	l.getLockBy(key).Unlock()
}

// getLockBy retrieves a mutex specific for a particular key
func (l *keyLock) getLockBy(key string) *sync.Mutex {
	l.mapLock.Lock()
	defer l.mapLock.Unlock()

	ret, found := l.locks[key]
	if found {
		return ret
	}

	ret = &sync.Mutex{}
	l.locks[key] = ret
	return ret
}
