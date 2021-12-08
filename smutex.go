// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package smutex

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type shard struct {
	sync.RWMutex
	sync.Cond
}

// SMutex128 represents a sharded RWMutex that supports finer-granularity concurrency
// contron hence reducing potential contention.
type SMutex struct {
	state  uint64
	shards uint
	mu     []shard
}

// New creates a new sharded mutex with a specified number of shards
func New(shards uint) *SMutex {
	mutex := &SMutex{
		shards: shards,
		mu:     make([]shard, shards),
	}

	// Wire up conditional variables to mutexes
	for i := 0; i < int(shards); i++ {
		mutex.mu[i].Cond.L = &mutex.mu[i].RWMutex
	}
	return mutex
}

// Lock locks rw for writing. If the lock is already locked for reading or writing,
// then Lock blocks until the lock is available.
func (rw *SMutex) Lock(shard uint) {
	mx := &rw.mu[shard%rw.shards]

	// Acquire write lock. If there's a priority reader waiting, unlock and wait on
	// until the associated conditional variable has a broadcast.
	mx.Lock()
	for {
		state := atomic.LoadUint64(&rw.state)
		readers := state >> 32
		writers := (state & 0xffffffff) + 1
		if readers > 0 {
			mx.Wait()
		}

		// Increment writer count now that we've acquired the lock
		if atomic.CompareAndSwapUint64(&rw.state, state, readers<<32+writers) {
			return // lock acquired
		}
	}
}

// Unlock unlocks rw for writing. It is a run-time error if rw is not locked for
// writing on entry to Unlock.
func (rw *SMutex) Unlock(shard uint) {
	rw.mu[shard%rw.shards].Unlock()
	for { // decrement the writer count
		state := atomic.LoadUint64(&rw.state)
		readers := state >> 32
		writers := (state & 0xffffffff) - 1
		if atomic.CompareAndSwapUint64(&rw.state, state, (readers<<32)+writers) {
			break
		}

		runtime.Gosched()
	}
}

// RLockAll locks rw for reading on all shards, the unlock needs to still happen
// shard by shard. It ensures that all writers have finished their work before
// acquiring the lock, in order to avoid any potential deadlocks.
func (rw *SMutex) RLockAll() {
	for { // increment global reader count
		state := atomic.LoadUint64(&rw.state)
		readers := (state >> 32) + 1
		writers := state & 0xffffffff
		if atomic.CompareAndSwapUint64(&rw.state, state, (readers<<32)+writers) {
			break // reader set to 1
		}

		if writers > 0 {
			runtime.Gosched()
		}
	}

	// Acquire read locks for every single shard
	for i := uint(0); i < rw.shards; i++ {
		rw.mu[i].RLock()
	}

	for { // decrement global reader count
		state := atomic.LoadUint64(&rw.state)
		readers := (state >> 32) - 1
		writers := state & 0xffffffff
		if atomic.CompareAndSwapUint64(&rw.state, state, (readers<<32)+writers) {
			break
		}

		runtime.Gosched()
	}

	// Wake up all writers and let them wait on their lock
	for i := uint(0); i < rw.shards; i++ {
		rw.mu[i].Broadcast()
	}
}

// RLock locks rw for reading. It should not be used for recursive read locking; a
// blocked Lock call excludes new readers from acquiring the lock.
func (rw *SMutex) RLock(shard uint) {
	rw.mu[shard%rw.shards].RLock()
}

// RUnlock undoes a single RLock call and does not affect other simultaneous readers.
func (rw *SMutex) RUnlock(shard uint) {
	rw.mu[shard%rw.shards].RUnlock()
}
