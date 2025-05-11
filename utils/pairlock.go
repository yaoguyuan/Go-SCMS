package utils

import (
	"fmt"
	"sync"
)

// PairLock is a structure that provides a locking mechanism for pairs of keys.
type PairLock struct {
	locks     map[string]*sync.Mutex
	innerLock sync.Mutex // to protect locks
}

// NewPairLock creates a new PairLock instance.
func NewPairLock() *PairLock {
	return &PairLock{
		locks: make(map[string]*sync.Mutex),
	}
}

// getLock retrieves or creates a mutex lock for the given prefix and pair of keys.
func (ml *PairLock) getLock(prefix string, p1 uint, p2 uint) *sync.Mutex {
	ml.innerLock.Lock()
	defer ml.innerLock.Unlock()

	key := fmt.Sprintf("%s-%d-%d", prefix, p1, p2)
	lock, exists := ml.locks[key]
	if !exists {
		lock = &sync.Mutex{}
		ml.locks[key] = lock
	}

	return lock
}

// Lock locks the mutex for the given prefix and pair of keys.
func (ml *PairLock) Lock(prefix string, p1 uint, p2 uint) {
	lock := ml.getLock(prefix, p1, p2)
	lock.Lock()
}

// Unlock unlocks the mutex for the given prefix and pair of keys.
func (ml *PairLock) Unlock(prefix string, p1 uint, p2 uint) {
	lock := ml.getLock(prefix, p1, p2)
	lock.Unlock()
}
