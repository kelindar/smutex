<p align="center">
<img width="330" height="110" src=".github/logo.png" border="0" alt="kelindar/smutex">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/smutex" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/smutex"><img src="https://pkg.go.dev/badge/github.com/kelindar/smutex" alt="PkgGoDev"></a>
<a href="https://goreportcard.com/report/github.com/kelindar/smutex"><img src="https://goreportcard.com/badge/github.com/kelindar/smutex" alt="Go Report Card"></a>
<a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://coveralls.io/github/kelindar/smutex"><img src="https://coveralls.io/repos/github/kelindar/smutex/badge.svg" alt="Coverage"></a>
</p>

# Sharded Mutex in Go

This package contains a sharded mutex which _should_ do better than a traditional `sync.RWMutex` in certain cases where you want to protect resources that are well distributed. For example, you can use this to protect a hash table as keys have no relation to each other. That being said, for the hash table use-case you should probably use `sync.Map`.

The `SMutex` works by actually creating an array of `sync.RWMutex` and providing `Lock()`, `Unlock()` methods that accept a `shard` argument. A shard argument can overflow the actual number of shards, and mutex uses a modulus operation to wrap around.

```go
// Create a new sharded mutex with 128 shards
mu := smutex.New(128)

// Acquire a write lock for shard #1
mu.Lock(1)
resourceInShard1 = "hello"

// Release the lock for shard #1
mu.Unlock(1)
```

## Global Read Lock

In addition to the ability to lock individual shards for both read and write using `Lock(shard)`, `Unlock(shard)`, `RLock(shard)` and `RUnlock(shard)`, this library also provides the ability to lock all shards for read at once, creating a priority lock that waits for all writers to finish prior to acquiring this.

```go
// Create a new sharded mutex with 128 shards
mu := smutex.New(128)

// Acquire a global lock on all shards
mu.RLockAll()

// Unlock shards one by one
for i := 0; i < shards; i++ {
    m.mu.RUnlock(uint(i))
}
```

## Caveats

Do benchmark on your own use-case if you want to use this library, I found that in certain cases the current implementation does not perform very well, but need to investigate a bit more.

## Benchmarks

```
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkLock/single/procs=64-8                     6338            181495 ns/op
BenchmarkLock/single/procs=256-8                    8844            165454 ns/op
BenchmarkLock/single/procs=1024-8                   8330            144282 ns/op
BenchmarkLock/single/procs=4096-8                   7188            142298 ns/op
BenchmarkLock/single/procs=16384-8                  8124            185415 ns/op
BenchmarkLock/single/procs=65536-8                  7146            144743 ns/op
BenchmarkLock/sharded/procs=64-8                   21411             53848 ns/op
BenchmarkLock/sharded/procs=256-8                  24202             50499 ns/op
BenchmarkLock/sharded/procs=1024-8                 26904             48445 ns/op
BenchmarkLock/sharded/procs=4096-8                 27116             47752 ns/op
BenchmarkLock/sharded/procs=16384-8                26959             50244 ns/op
BenchmarkLock/sharded/procs=65536-8                25498             48610 ns/op
```
