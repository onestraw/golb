package roundrobin

import (
	"strings"
	"testing"
)

func testGetPeer(t *testing.T, pool *Pool, getCount int, expected string) {
	result := []string{}

	t.Logf("%v", pool)
	for i := 0; i < getCount; i++ {
		peer := pool.Get()
		result = append(result, peer.addr)
	}

	result_s := strings.Join(result, ",")
	if result_s != expected {
		t.Errorf("expected order: '%s', but got '%s'", expected, result_s)
	}
}

func TestGetPeerWithDifferentWeight(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 5),
		CreatePeer("b", 1),
		CreatePeer("c", 1),
	}
	expected_order := "a,a,b,a,c,a,a"
	pool := CreatePool(peers)
	testGetPeer(t, pool, 7, expected_order)
}

func TestGetPeerWithSameWeight(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 1),
		CreatePeer("b", 1),
		CreatePeer("c", 1),
	}
	expected_order := "a,b,c,a,b,c"
	pool := CreatePool(peers)
	testGetPeer(t, pool, 6, expected_order)
}

func TestGetPeerWithSameWeightNotOne(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 2),
		CreatePeer("b", 2),
		CreatePeer("c", 2),
	}
	expected_order := "a,b,c,a,b,c"
	pool := CreatePool(peers)
	testGetPeer(t, pool, 6, expected_order)
}

func TestEqualGetPeer(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 5),
		CreatePeer("b", 3),
		CreatePeer("c", 1),
	}
	expected := "a,b,c,a,b,c"
	pool := CreatePool(peers)
	result := []string{}

	t.Logf("%v", pool)
	for i := 0; i < 6; i++ {
		peer := pool.EqualGet()
		result = append(result, peer.addr)
	}

	result_s := strings.Join(result, ",")
	if result_s != expected {
		t.Errorf("expected order: '%s', but got '%s'", expected, result_s)
	}

	pool.current = 1<<64 - 1
	peer := pool.EqualGet()
	if pool.current != 0 || peer.addr != "a" {
		t.Errorf("the index should be 0")
	}
}

func TestAddPeer(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 1),
	}
	pool := CreatePool(peers)

	if pool.Size() != 1 {
		t.Errorf("Pool size should be 1")
	}

	pool.Add(CreatePeer("b", 1))
	if pool.Size() != 2 {
		t.Errorf("Pool size should be 2")
	}
}

func TestRemovePeer(t *testing.T) {
	peers := []*Peer{
		CreatePeer("a", 1),
		CreatePeer("b", 1),
	}
	pool := CreatePool(peers)

	if pool.Size() != 2 {
		t.Errorf("Pool size should be 2")
	}

	pool.Remove(CreatePeer("b", 3))
	if pool.Size() != 1 {
		t.Errorf("Pool size should be 1")
	}

	pool.Remove(CreatePeer("a", 3))
	if pool.Size() != 0 {
		t.Errorf("Pool size should be 0")
	}
}

func TestEmpty(t *testing.T) {
	pool := CreatePool([]*Peer{})
	if pool.Get() != nil {
		t.Errorf("Pool is empty")
	}
	if pool.EqualGet() != nil {
		t.Errorf("Pool is empty")
	}
	pool.Add(nil)
	if pool.Size() != 0 {
		t.Errorf("Pool is empty")
	}
	pool.Remove(nil)
	t.Logf("%v", pool)
}
