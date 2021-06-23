// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package smutex

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Store represents a concurrent store for testing
type Store interface {
	Set(int64, string)
	Get(int64) string
}

func BenchmarkLock(b *testing.B) {
	size := int64(10000000)

	single := newLocked()
	for i := int64(64); i <= (1 << 16); i *= 4 {
		runBenchmark(b, "single", single, size, i)
	}

	sharded := newSharded()
	for i := int64(64); i <= (1 << 16); i *= 4 {
		runBenchmark(b, "sharded", sharded, size, i)
	}
}

func runBenchmark(b *testing.B, name string, store Store, size, procs int64) {
	rand.Seed(1)
	b.Run(fmt.Sprintf("%v/procs=%v", name, procs), func(b *testing.B) {
		b.SetParallelism(int(procs))
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for i := 0; i < 5; i++ {
					store.Set(rand.Int63n(size), "value")
				}
				for i := 0; i < 5; i++ {
					store.Get(rand.Int63n(size))
				}
			}
		})
	})
}

func TestMutex(t *testing.T) {
	var mu SMutex128
	var wg sync.WaitGroup
	var resource, out string

	// Acquire a write lock
	mu.Lock(1)

	// Concurrently, start a reader
	wg.Add(1)
	go func() {
		mu.RLock(1)
		defer mu.RUnlock(1)
		out = resource
		wg.Done()
	}()

	// Write the resource
	resource = "hello"
	mu.Unlock(1)

	// Wait for the reader to finish
	wg.Wait()
	assert.Equal(t, "hello", out)
}

// --------------------------- Locked Map ----------------------------

const work = 1000

// An implementation of a locked map using a mutex
type lockedMap struct {
	mu   sync.RWMutex
	data map[int64]string
}

func newLocked() *lockedMap {
	return &lockedMap{data: make(map[int64]string)}
}

// Set sets the value into a locked map
func (l *lockedMap) Set(k int64, v string) {
	l.mu.Lock()
	for i := 0; i < work; i++ {
		l.data[k] = v
	}
	runtime.Gosched()
	for i := 0; i < work; i++ {
		l.data[k] = v
	}
	l.mu.Unlock()
}

// Get gets a value from a locked map
func (l *lockedMap) Get(k int64) (v string) {
	l.mu.RLock()
	for i := 0; i < work; i++ {
		v, _ = l.data[k]
	}
	runtime.Gosched()
	for i := 0; i < work; i++ {
		v, _ = l.data[k]
	}
	l.mu.RUnlock()
	return
}

// --------------------------- Sharded Map ----------------------------

// An implementation of a locked map using a smutex
type shardedMap struct {
	mu   SMutex128
	data []map[int64]string
}

func newSharded() *shardedMap {
	m := &shardedMap{}
	for i := 0; i < shards; i++ {
		m.data = append(m.data, map[int64]string{})
	}
	return m
}

// Set sets the value into a locked map
func (l *shardedMap) Set(k int64, v string) {
	l.mu.Lock(uint(k))
	for i := 0; i < work; i++ {
		l.data[k%shards][k] = v
	}
	runtime.Gosched()
	for i := 0; i < work; i++ {
		l.data[k%shards][k] = v
	}
	l.mu.Unlock(uint(k))
}

// Get gets a value from a locked map
func (l *shardedMap) Get(k int64) (v string) {
	l.mu.RLock(uint(k))
	for i := 0; i < work; i++ {
		v, _ = l.data[k%shards][k]
	}
	runtime.Gosched()
	for i := 0; i < work; i++ {
		v, _ = l.data[k%shards][k]
	}
	l.mu.RUnlock(uint(k))
	return
}
