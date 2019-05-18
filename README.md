## *This project is obsolete. Please use the [`rhh`](https://github.com/tidwall/rhh) package instead*

# `spmap`

[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/tidwall/spmap)

A hashmap for Go that uses crypto random seeds, [hash hints](#hash-hints), [open addressing](https://en.wikipedia.org/wiki/Hash_table#Open_addressing), and [robin hood hashing](https://en.wikipedia.org/wiki/Hash_table#Robin_Hood_hashing).

It's a very specialized map that is similar to `map[string]interface{}`.

# Getting Started

### Installing

To start using spmap, install Go and run `go get`:

```sh
$ go get -u github.com/tidwall/spmap
```

## Hash hints

Allows for generating key hashes prior to a set/get/delete call. 
This can be useful in multithreaded environments where there're a lot of `Lock/Unlock` wrapped around map mutations, and it keeps the hashing outside of the locks.

For example:

```go
hash, seed := m.Hash(key) // expensive and slow. keep outside the lock
mu.Lock()
m.SetWithHint(key, hash, seed, value)
// ... do other sychronization stuff
mu.Unlock()
```

## Contact

Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

`spmap` source code is available under the MIT [License](/LICENSE).
