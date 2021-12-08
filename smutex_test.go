// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package smutex

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const shards = 16

// Store represents a concurrent store for testing
type Store interface {
	Set(int64, string)
	Get(int64) string
}

/*
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkLock/single/procs=64-8         	    7354	    159872 ns/op	     559 B/op	       0 allocs/op
BenchmarkLock/single/procs=256-8        	   10000	    145686 ns/op	     492 B/op	       0 allocs/op
BenchmarkLock/single/procs=1024-8       	    9230	    127378 ns/op	     455 B/op	       3 allocs/op
BenchmarkLock/single/procs=4096-8       	    9932	    160553 ns/op	    1222 B/op	       7 allocs/op
BenchmarkLock/single/procs=16384-8      	   11259	    123632 ns/op	    1077 B/op	      24 allocs/op
BenchmarkLock/single/procs=65536-8      	   11756	    119162 ns/op	    3698 B/op	      90 allocs/op
BenchmarkLock/sharded/procs=64-8        	   41248	     29622 ns/op	     652 B/op	       0 allocs/op
BenchmarkLock/sharded/procs=256-8       	   50426	     27941 ns/op	     129 B/op	       0 allocs/op
BenchmarkLock/sharded/procs=1024-8      	   50235	     24113 ns/op	     206 B/op	       0 allocs/op
BenchmarkLock/sharded/procs=4096-8      	   53744	     24013 ns/op	      93 B/op	       1 allocs/op
BenchmarkLock/sharded/procs=16384-8     	   53571	     23334 ns/op	     230 B/op	       5 allocs/op
BenchmarkLock/sharded/procs=65536-8     	   49945	     25731 ns/op	     863 B/op	      21 allocs/op
*/
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

/*
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkLockUnlock/rwmutex-8         	62175843	        18.78 ns/op	       0 B/op	       0 allocs/op
BenchmarkLockUnlock/smutex-8          	38551368	        38.84 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkLockUnlock(b *testing.B) {
	b.Run("rwmutex", func(b *testing.B) {
		var mu sync.RWMutex
		for i := 0; i < b.N; i++ {
			mu.Lock()
			mu.Unlock()
		}
	})

	b.Run("smutex", func(b *testing.B) {
		mu := New(128)
		for i := 0; i < b.N; i++ {
			mu.Lock(1)
			mu.Unlock(1)
		}
	})
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
	mu := New(shards)
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

func TestRLockAll(t *testing.T) {
	m := newSharded()
	var wg sync.WaitGroup
	wg.Add(512)
	for i := 0; i < 512; i++ {
		go func() {
			for i := 0; i < 100; i++ {
				time.Sleep(1 * time.Millisecond)
				m.Set(rand.Int63n(shards), "ok")
			}
			wg.Done()
		}()
	}

	m.mu.RLockAll()
	time.Sleep(10 * time.Millisecond)

	// Unlock all
	for i := 0; i < shards; i++ {
		m.mu.RUnlock(uint(i))
	}

	// Wait for all writers to finish
	wg.Wait()
	assert.True(t, true)
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
	mu   *SMutex
	data []map[int64]string
}

func newSharded() *shardedMap {
	m := &shardedMap{
		mu: New(shards),
	}
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
