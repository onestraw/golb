package roundrobin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testGetPeer(t *testing.T, pool *Pool, getCount int, expected string) {
	t.Logf("%v", pool)
	result := []string{}
	for i := 0; i < getCount; i++ {
		peer := pool.Get()
		result = append(result, peer)
	}
	assert.Equal(t, expected, strings.Join(result, ","))
}

func testEqualGetPeer(t *testing.T, pool *Pool, getCount int, expected string) {
	t.Logf("%v", pool)
	result := []string{}
	for i := 0; i < getCount; i++ {
		peer := pool.EqualGet()
		result = append(result, peer)
	}
	assert.Equal(t, expected, strings.Join(result, ","))
}

func TestGetPeerWithDifferentWeight(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 5),
		CreatePeer("b", 1),
		CreatePeer("c", 1),
	}
	expected_order := "a,a,b,a,c,a,a"
	pool := &Pool{peers: peers}
	testGetPeer(t, pool, 7, expected_order)
}

func TestGetPeerWithSameWeight(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 1),
		CreatePeer("b", 1),
		CreatePeer("c", 1),
	}
	expected_order := "a,b,c,a,b,c"
	pool := &Pool{peers: peers}
	testGetPeer(t, pool, 6, expected_order)
}

func TestGetPeerWithSameWeightNotOne(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 2),
		CreatePeer("b", 2),
		CreatePeer("c", 2),
	}
	expected_order := "a,b,c,a,b,c"
	pool := &Pool{peers: peers}
	testGetPeer(t, pool, 6, expected_order)
}

func TestEqualGetPeer(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 5),
		CreatePeer("b", 3),
		CreatePeer("c", 1),
	}
	expected := "a,b,c,a,b,c"
	pool := &Pool{peers: peers}
	testEqualGetPeer(t, pool, 6, expected)

	pool.current = 1<<64 - 1
	peer := pool.EqualGet()
	assert.Equal(t, uint64(0), pool.current)
	assert.Equal(t, "a", peer)
}

func TestDownWithEqualGet(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 5),
		CreatePeer("b", 3),
		CreatePeer("c", 1),
	}
	expected := "a,b,c,a,b,c"
	pool := &Pool{peers: peers}
	testEqualGetPeer(t, pool, 6, expected)

	pool.DownPeer("b")
	expected = "a,c,a,c,a,c"
	testEqualGetPeer(t, pool, 6, expected)

	pool.DownPeer("a")
	pool.DownPeer("c")
	expected = ",,,,,"
	testEqualGetPeer(t, pool, 6, expected)
}

func TestAddPeer(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 1),
	}
	pool := &Pool{peers: peers}
	assert.Equal(t, 1, pool.Size())

	pool.Add("b", 1)
	assert.Equal(t, 2, pool.Size())

	pool.Add("b", 1)
	assert.Equal(t, 2, pool.Size())
}

func TestRemovePeer(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 1),
		CreatePeer("b", 1),
	}
	pool := &Pool{peers: peers}
	assert.Equal(t, 2, pool.Size())

	pool.Remove("b")
	assert.Equal(t, 1, pool.Size())
	pool.Remove("b")
	assert.Equal(t, 1, pool.Size())

	pool.Remove("a")
	assert.Equal(t, 0, pool.Size())
}

func TestEmpty(t *testing.T) {
	pool := CreatePool(map[string]int{})
	assert.Equal(t, "", pool.Get())
	assert.Equal(t, "", pool.EqualGet())

	pool.Add("", 1)
	assert.Equal(t, 0, pool.Size())

	pool.Remove("")
	t.Logf("%v", pool)
}

func TestDownPeer(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 1),
		CreatePeer("b", 1),
	}
	pool := &Pool{peers: peers}
	assert.Equal(t, 2, pool.Size())

	expected_order := "a,b,a,b,a,b"
	testGetPeer(t, pool, 6, expected_order)

	pool.DownPeer("b")
	expected_order = "a,a,a,a,a,a"
	testGetPeer(t, pool, 6, expected_order)

	pool.UpPeer("b")
	expected_order = "a,b,a,b,a,b"
	testGetPeer(t, pool, 6, expected_order)

	pool.DownPeer("a")
	pool.DownPeer("b")
	expected_order = ",,,,,"
	testGetPeer(t, pool, 6, expected_order)
}
