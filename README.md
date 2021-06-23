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

This package contains a sharded mutex which *should* do better than a traditional `sync.RWMutex` in certain cases where you want to protect resources that are well distributed. For example, you can use this to protect a hash table as keys have no relation to each other. That being said, for the hash table use-case you should probably use `sync.Map`.

The `SMutex128` works by actually creating 128 `sync.RWMutex` and providing `Lock()`, `Unlock()` methods that accept a `shard` argument. A shard argument can overflow the actual number of shards, and mutex uses a modulus operation to wrap around.


```go
// Acquire a write lock for shard #1
mu.Lock(1)
resourceInShard1 = "hello"

// Release the lock in shard #1
mu.Unlock(1)
```

## Caveats

* Sharded mutex would use significantly more memory and needs to be used with care. In fact, the 128 shard implementation would use 8192 bytes of memory, and would ideally be living in L1. The reason being is that the current implementation pads mutexes so only one of them is present in a cache line, to prevent false sharing. 
* Do benchmark on your own use-case if you want to use this library, I found that in certain cases the current implementation does not perform very well, but need to investigate a bit more. It's usually on-par with the `RWMutex` at the very least.

## Benchmarks

```
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkLock/single/procs=64-8                    64495            184600 ns/op
BenchmarkLock/single/procs=256-8                   75350            161236 ns/op
BenchmarkLock/single/procs=1024-8                  85765            161982 ns/op
BenchmarkLock/single/procs=4096-8                  86328            160925 ns/op
BenchmarkLock/single/procs=16384-8                 85803            153741 ns/op
BenchmarkLock/single/procs=65536-8                 85806            152246 ns/op
BenchmarkLock/sharded/procs=64-8                  342633             35435 ns/op
BenchmarkLock/sharded/procs=256-8                 390313             30818 ns/op
BenchmarkLock/sharded/procs=1024-8                416959             30493 ns/op
BenchmarkLock/sharded/procs=4096-8                443528             30246 ns/op
BenchmarkLock/sharded/procs=16384-8               427383             30118 ns/op
BenchmarkLock/sharded/procs=65536-8               451612             30922 ns/op
```
