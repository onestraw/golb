package chash

import (
	"testing"
)

func TestGetEmpty(t *testing.T) {
	pool := New()
	t.Logf("%v", pool)
	if pool.Get("any") != nil {
		t.Errorf("Pool is empty")
	}
}

func TestGet(t *testing.T) {
	pool := New()
	tests := []string{
		"1.1.1.1",
		"2.2.2.2",
		"3.3.3.3",
	}
	for _, addr := range tests {
		pool.Add(&Peer{addr})
	}
	t.Logf("%v", pool.Get("/order"))
	t.Logf("%v", pool.Get("/detail"))

	if len(pool.vNodes) != pool.replica*3 {
		t.Errorf("Pool's vNodes size should be %d, but got %d",
			pool.replica*3, len(pool.sortedHashes))
	}

	if len(pool.sortedHashes) != pool.replica*3 {
		t.Errorf("Pool's sortedHashes size should be %d, but got %d",
			pool.replica*3, len(pool.sortedHashes))
	}

	if pool.Size() != len(tests) {
		t.Errorf("Pool size should be %d, but got %d", len(tests), pool.Size())
	}
}

// refer to github.com/stathat/consistent/
// check the smallest hash value that greater than hash(key)
type gtest struct {
	in  string
	out string
}

func TestAdd(t *testing.T) {
	var testsBefore = []gtest{
		{"/redis-B", "1.1.1.1"},
		{"/login", "2.2.2.2"},
		{"/detail", "3.3.3.3"},
	}
	var testsAfter = []gtest{
		{"/redis-B", "1.1.1.1"},
		{"/login", "2.2.2.2"},
		{"/detail", "5.5.5.5"},
	}
	pool := New()
	for _, tc := range testsBefore {
		pool.Add(&Peer{tc.out})
		t.Logf("hash(%s)=%v", tc.in, pool.hash(tc.in))
	}
	//t.Logf("%v", pool)
	for _, tc := range testsBefore {
		result := pool.Get(tc.in).addr
		if result != tc.out {
			t.Errorf("before add, key=%s, expected %s, but got %s", tc.in, tc.out, result)
		}
	}

	pool.Add(&Peer{"4.4.4.4"})
	pool.Add(&Peer{"5.5.5.5"})
	//t.Logf("%v", pool)
	for _, tc := range testsAfter {
		result := pool.Get(tc.in).addr
		if result != tc.out {
			t.Errorf("after add, key=%s, expected %s, but got %s", tc.in, tc.out, result)
		}
	}
}

func TestRemove(t *testing.T) {
	var rtestsBefore = []gtest{
		{"/redis-B", "1.1.1.1"},
		{"/login", "2.2.2.2"},
		{"/detail", "3.3.3.3"},
	}
	var rtestsAfter = []gtest{
		{"/redis-B", "3.3.3.3"},
		{"/login", "2.2.2.2"},
		{"/detail", "3.3.3.3"},
	}

	pool := New()
	for _, tc := range rtestsBefore {
		pool.Add(&Peer{tc.out})
	}
	for i, v := range rtestsBefore {
		result := pool.Get(v.in).addr
		if result != v.out {
			t.Errorf("%d. got %q, expected %q before rm", i, result, v.out)
		}
	}
	pool.Remove(&Peer{"1.1.1.1"})
	//t.Logf("%v", pool)
	for i, v := range rtestsAfter {
		result := pool.Get(v.in).addr
		if result != v.out {
			t.Errorf("%d. got %q, expected %q after rm", i, result, v.out)
		}
	}
}
