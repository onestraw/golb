package chash

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEmpty(t *testing.T) {
	pool := New()
	t.Logf("%v", pool)
	assert.Equal(t, "", pool.Get("any"))
}

func TestGet(t *testing.T) {
	tests := []string{
		"1.1.1.1",
		"2.2.2.2",
		"3.3.3.3",
	}
	pool := CreatePool(tests)
	t.Logf("%v", pool.Get("/order"))
	t.Logf("%v", pool.Get("/detail"))

	assert.Equal(t, pool.replica*3, len(pool.vNodes))
	assert.Equal(t, pool.replica*3, len(pool.sortedHashes))
	assert.Equal(t, len(tests), pool.Size())

	assert.Equal(t, "", pool.Get())
	assert.Equal(t, "", pool.Get(10))
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
		pool.Add(tc.out)
		t.Logf("hash(%s)=%v", tc.in, pool.hash(tc.in))
	}
	//t.Logf("%v", pool)
	for _, tc := range testsBefore {
		result := pool.Get(tc.in)
		assert.Equal(t, tc.out, result, fmt.Sprintf("before add, key=%s, expected %s, but got %s", tc.in, tc.out, result))
	}

	pool.Add("4.4.4.4")
	pool.Add("5.5.5.5")
	//t.Logf("%v", pool)
	for _, tc := range testsAfter {
		result := pool.Get(tc.in)
		assert.Equal(t, tc.out, result, fmt.Sprintf("after add, key=%s, expected %s, but got %s", tc.in, tc.out, result))
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
		pool.Add(tc.out)
	}

	for i, v := range rtestsBefore {
		result := pool.Get(v.in)
		assert.Equal(t, v.out, result, fmt.Sprintf("%d. got %q, expected %q before rm", i, result, v.out))
	}
	pool.Remove("1.1.1.1")
	//t.Logf("%v", pool)
	for i, v := range rtestsAfter {
		result := pool.Get(v.in)
		assert.Equal(t, v.out, result, fmt.Sprintf("%d. got %q, expected %q after rm", i, result, v.out))
	}
}

func TestDownPeer(t *testing.T) {
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
		pool.Add(tc.out)
	}

	for i, v := range rtestsBefore {
		result := pool.Get(v.in)
		assert.Equal(t, v.out, result, fmt.Sprintf("%d. got %q, expected %q before down", i, result, v.out))
	}

	pool.DownPeer("1.1.1.1")
	for i, v := range rtestsAfter {
		result := pool.Get(v.in)
		assert.Equal(t, v.out, result, fmt.Sprintf("%d. got %q, expected %q after down", i, result, v.out))
	}

	pool.UpPeer("1.1.1.1")
	for i, v := range rtestsBefore {
		result := pool.Get(v.in)
		assert.Equal(t, v.out, result, fmt.Sprintf("%d. got %q, expected %q after up", i, result, v.out))
	}

	pool.DownPeer("1.1.1.1")
	pool.DownPeer("2.2.2.2")
	pool.DownPeer("3.3.3.3")
	for i, v := range rtestsBefore {
		result := pool.Get(v.in)
		assert.Equal(t, "", result, fmt.Sprintf("%d. got %q, expected '' after down all", i, result))
	}
}
