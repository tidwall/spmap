// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package spmap

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"github.com/tidwall/lotsa"
	"github.com/tidwall/spinlock"
)

func init() {
	//var seed int64 = 1519776033517775607
	seed := (time.Now().UnixNano())
	println("seed:", seed)
	rand.Seed(seed)
}

func random(N int, perm bool) []string {
	strs := make([]string, N)
	if perm {
		for i, x := range rand.Perm(N) {
			strs[i] = fmt.Sprintf("%d", x)
		}
	} else {
		m := make(map[string]bool)
		for len(m) < N {
			m[fmt.Sprintf("%d", rand.Uint64())] = true
		}
		var i int
		for k := range m {
			strs[i] = k
			i++
		}
	}
	return strs
}

func ps(p unsafe.Pointer) string {
	if p == nil {
		return "<nil>"
	}
	return *(*string)(p)
}

func sp(s string) unsafe.Pointer {
	return unsafe.Pointer(&s)
}

func shuffle(strs []string) {
	for i := range strs {
		j := rand.Intn(i + 1)
		strs[i], strs[j] = strs[j], strs[i]
	}
}

func addstr(s string, x int64) string {
	n, _ := strconv.ParseUint(s, 10, 64)
	if x < 0 {
		return fmt.Sprintf("%d", n+uint64(x*-1))
	}
	return fmt.Sprintf("%d", n+uint64(x))
}

func TestRandomData(t *testing.T) {
	N := 1000
	start := time.Now()
	for time.Since(start) < time.Second*2 {
		strs := random(N, true)
		var opts Options
		var m *Map
		switch rand.Int() % 5 {
		default:
			opts.InitialSize = N / ((rand.Int() % 3) + 1)
			opts.Shrinkable = rand.Int()%2 == 1
			m = New(&opts)
		case 1:
			m = new(Map)
		case 2:
			m = New(nil)
		}
		v, ok := m.Get("999")
		if ok || v != nil {
			t.Fatalf("expected %v, got %v", nil, v)
		}
		v, ok = m.Delete("999")
		if ok || v != nil {
			t.Fatalf("expected %v, got %v", nil, v)
		}
		if m.Len() != 0 {
			t.Fatalf("expected %v, got %v", 0, m.Len())
		}
		// set a bunch of items
		for i := 0; i < len(strs); i++ {
			v, ok := m.Set(strs[i], sp(strs[i]))
			if ok || v != nil {
				t.Fatalf("expected %v, got %v", nil, v)
			}
		}
		if m.Len() != N {
			t.Fatalf("expected %v, got %v", N, m.Len())
		}
		// retrieve all the items
		shuffle(strs)
		for i := 0; i < len(strs); i++ {
			v, ok := m.Get(strs[i])
			if !ok || v == nil || ps(v) != strs[i] {
				t.Fatalf("expected %v, got %v", strs[i], ps(v))
			}
		}
		// replace all the items
		shuffle(strs)
		for i := 0; i < len(strs); i++ {
			v, ok := m.Set(strs[i], sp(addstr(strs[i], 1)))
			if !ok || ps(v) != strs[i] {
				t.Fatalf("expected %v, got %v", strs[i], ps(v))
			}
		}
		if m.Len() != N {
			t.Fatalf("expected %v, got %v", N, m.Len())
		}
		// retrieve all the items
		shuffle(strs)
		for i := 0; i < len(strs); i++ {
			v, ok := m.Get(strs[i])
			if !ok || ps(v) != addstr(strs[i], 1) {
				t.Fatalf("expected %v, got %v", addstr(strs[i], 1), ps(v))
			}
		}
		// remove half the items
		shuffle(strs)
		for i := 0; i < len(strs)/2; i++ {
			v, ok := m.Delete(strs[i])
			if !ok || ps(v) != addstr(strs[i], 1) {
				t.Fatalf("expected %v, got %v", addstr(strs[i], 1), ps(v))
			}
		}
		if m.Len() != N/2 {
			t.Fatalf("expected %v, got %v", N/2, m.Len())
		}
		// check to make sure that the items have been removed
		for i := 0; i < len(strs)/2; i++ {
			v, ok := m.Get(strs[i])
			if ok || v != nil {
				t.Fatalf("expected %v, got %v", nil, v)
			}
		}
		// check the second half of the items
		for i := len(strs) / 2; i < len(strs); i++ {
			v, ok := m.Get(strs[i])
			if !ok || ps(v) != addstr(strs[i], 1) {
				t.Fatalf("expected %v, got %v", addstr(strs[i], 1), ps(v))
			}
		}
		// try to delete again, make sure they don't exist
		for i := 0; i < len(strs)/2; i++ {
			v, ok := m.Delete(strs[i])
			if ok || v != nil {
				t.Fatalf("expected %v, got %v", nil, v)
			}
		}
		if m.Len() != N/2 {
			t.Fatalf("expected %v, got %v", N/2, m.Len())
		}
		m.Scan(func(key string, value unsafe.Pointer) bool {
			if ps(value) != addstr(key, 1) {
				t.Fatalf("expected %v, got %v", addstr(key, 1), ps(value))
			}
			return true
		})
		var n int
		m.Scan(func(key string, value unsafe.Pointer) bool {
			n++
			return false
		})
		if n != 1 {
			t.Fatalf("expected %v, got %v", 1, n)
		}
		for i := len(strs) / 2; i < len(strs); i++ {
			v, ok := m.Delete(strs[i])
			if !ok || ps(v) != addstr(strs[i], 1) {
				t.Fatalf("expected %v, got %v", addstr(strs[i], 1), ps(v))
			}
		}
	}
}

func TestBench(t *testing.T) {
	N, _ := strconv.ParseUint(os.Getenv("MAPBENCH"), 10, 64)
	if N == 0 {
		fmt.Printf("Enable benchmarks with MAPBENCH=1000000\n")
		return
	}
	strs := random(int(N), false)
	var pstrs []unsafe.Pointer
	for i := range strs {
		pstrs = append(pstrs, unsafe.Pointer(&strs[i]))
	}
	// t.Run("RobinHoodHintsThreads", func(t *testing.T) {
	// 	testPerf(strs, pstrs, "robinhood-hints")
	// })
	t.Run("RobinHood", func(t *testing.T) {
		testPerf(strs, pstrs, "robinhood")
	})
	// t.Run("StdlibThreads", func(t *testing.T) {
	// 	testPerf(strs, pstrs, "stdlib-threads")
	// })
	t.Run("Stdlib", func(t *testing.T) {
		testPerf(strs, pstrs, "stdlib")
	})
	// t.Run("BTree", func(t *testing.T) {
	// 	testPerf(strs, "btree")
	// })
}

func printItem(s string, size int, dir int) {
	for len(s) < size {
		if dir == -1 {
			s += " "
		} else {
			s = " " + s
		}
	}
	fmt.Printf("%s ", s)
}

// type kvitemT struct {
// 	key   string
// 	value interface{}
// }

// func (a kvitemT) Less(b btree.Item) bool {
// 	return a.key < b.(kvitemT).key
// }

func testPerf(strs []string, pstrs []unsafe.Pointer, which string) {
	var ms1, ms2 runtime.MemStats
	initSize := len(strs) * 2
	threads := 1
	defer func() {
		heapBytes := int(ms2.HeapAlloc - ms1.HeapAlloc)
		fmt.Printf("memory %13s bytes %19s/entry \n",
			commaize(heapBytes), commaize(heapBytes/len(strs)))
		fmt.Printf("\n")
	}()
	runtime.GC()
	time.Sleep(time.Millisecond * 100)
	runtime.ReadMemStats(&ms1)

	var setop, getop, delop func(int, int)
	var scnop func()
	switch which {
	case "stdlib":
		m := make(map[string]unsafe.Pointer, initSize)
		setop = func(i, _ int) { m[strs[i]] = pstrs[i] }
		getop = func(i, _ int) { _ = m[strs[i]] }
		delop = func(i, _ int) { delete(m, strs[i]) }
		scnop = func() {
			for range m {
			}
		}
	case "stdlib-threads":
		threads = 4
		m := make(map[string]unsafe.Pointer, initSize)
		var mu spinlock.Locker
		setop = func(i, _ int) {
			mu.Lock()
			m[strs[i]] = pstrs[i]
			mu.Unlock()
		}
		getop = func(i, _ int) {
			mu.Lock()
			_ = m[strs[i]]
			mu.Unlock()
		}
		delop = func(i, _ int) {
			mu.Lock()
			delete(m, strs[i])
			mu.Unlock()
		}
		scnop = func() {
			mu.Lock()
			for range m {
			}
			mu.Unlock()
		}
	case "robinhood":
		m := New(&Options{
			InitialSize: initSize,
		})
		setop = func(i, _ int) { m.Set(strs[i], pstrs[i]) }
		getop = func(i, _ int) { m.Get(strs[i]) }
		delop = func(i, _ int) { m.Delete(strs[i]) }
		scnop = func() {
			m.Scan(func(key string, value unsafe.Pointer) bool {
				return true
			})
		}
	case "robinhood-hints":
		threads = 4
		m := New(&Options{
			InitialSize: initSize,
		})
		var mu spinlock.Locker
		// hashes := make([]uint32, len(strs))
		// seeds := make([]uint32, len(strs))
		// for i := range strs {
		// 	hashes[i], seeds[i] = m.Hash(strs[i])
		// }
		setop = func(i, _ int) {
			hash, seed := m.Hash(strs[i])
			mu.Lock()
			m.SetWithHint(strs[i], hash, seed, pstrs[i])
			mu.Unlock()
		}
		getop = func(i, _ int) {
			hash, seed := m.Hash(strs[i])
			mu.Lock()
			m.GetWithHint(strs[i], hash, seed)
			mu.Unlock()
		}
		delop = func(i, _ int) {
			hash, seed := m.Hash(strs[i])
			mu.Lock()
			m.DeleteWithHint(strs[i], hash, seed)
			mu.Unlock()
		}
		scnop = func() {
			mu.Lock()
			m.Scan(func(key string, value unsafe.Pointer) bool {
				return true
			})
			mu.Unlock()
		}

		// case "btree":
		// 	tr := btree.New(128)
		// 	setop = func(i, _ int) { tr.ReplaceOrInsert(kvitemT{strs[i], strs[i]}) }
		// 	getop = func(i, _ int) { tr.Get(kvitemT{key: strs[i]}) }
		// 	delop = func(i, _ int) { tr.Delete(kvitemT{key: strs[i]}) }
		// 	scnop = func() {
		// 		tr.Ascend(func(v btree.Item) bool {
		// 			return true
		// 		})
		// 	}
	}
	fmt.Printf("-- %s --", which)
	if threads > 1 {
		fmt.Printf(" (%d threads)", threads)
	}
	fmt.Printf("\n")

	ops := []func(int, int){setop, getop, setop, nil, delop}
	tags := []string{"set", "get", "reset", "scan", "delete"}
	for i := range ops {
		shuffle(strs)
		var na bool
		var n int
		start := time.Now()
		if tags[i] == "scan" {
			op := scnop
			if op == nil {
				na = true
			} else {
				n = 20
				lotsa.Ops(n, threads, func(_, _ int) { op() })
			}

		} else {
			n = len(strs)
			lotsa.Ops(n, threads, ops[i])
		}
		dur := time.Since(start)
		if i == 0 {
			runtime.GC()
			time.Sleep(time.Millisecond * 100)
			runtime.ReadMemStats(&ms2)
		}
		printItem(tags[i], 9, -1)
		if na {
			printItem("-- unavailable --", 14, 1)
		} else {
			if n == -1 {
				printItem("unknown ops", 14, 1)
			} else {
				printItem(fmt.Sprintf("%s ops", commaize(n)), 14, 1)
			}
			printItem(fmt.Sprintf("%.0fms", dur.Seconds()*1000), 8, 1)
			if n != -1 {
				printItem(fmt.Sprintf("%s/sec", commaize(int(float64(n)/dur.Seconds()))), 18, 1)
			}
		}
		fmt.Printf("\n")
	}
}

func commaize(n int) string {
	s1, s2 := fmt.Sprintf("%d", n), ""
	for i, j := len(s1)-1, 0; i >= 0; i, j = i-1, j+1 {
		if j%3 == 0 && j != 0 {
			s2 = "," + s2
		}
		s2 = string(s1[i]) + s2
	}
	return s2
}
