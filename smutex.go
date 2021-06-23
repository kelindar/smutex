// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package smutex

import "sync"

const shards = 128

// SMutex128 represents a sharded RWMutex that supports finer-granularity concurrency
// contron hence reducing potential contention.
type SMutex128 struct {
	mu [shards]struct {
		sync.RWMutex
		_ [40]byte // Padding to prevent false sharing
	}
}

// Lock locks rw for writing. If the lock is already locked for reading or writing,
// then Lock blocks until the lock is available.
func (rw *SMutex128) Lock(shard uint) {
	rw.mu[shard%shards].Lock()
}

// Unlock unlocks rw for writing. It is a run-time error if rw is not locked for
// writing on entry to Unlock.
func (rw *SMutex128) Unlock(shard uint) {
	rw.mu[shard%shards].Unlock()
}

// RLock locks rw for reading. It should not be used for recursive read locking; a
// blocked Lock call excludes new readers from acquiring the lock.
func (rw *SMutex128) RLock(shard uint) {
	rw.mu[shard%shards].RLock()
}

// RUnlock undoes a single RLock call and does not affect other simultaneous readers.
func (rw *SMutex128) RUnlock(shard uint) {
	rw.mu[shard%shards].RUnlock()
}
