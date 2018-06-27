package stats

import "testing"

func TestInc(t *testing.T) {
	s := New()
	peer := "1.2.3.4"
	code := 200
	s.Inc(peer, code)
	if s.StatusCode[peer][code] != 1 {
		t.Errorf("%s 's %d count should be 1", peer, code)
	}
}

func TestString(t *testing.T) {
	s := New()
	s.Inc("web", 200)
	s.Inc("web", 200)
	s.Inc("web", 400)
	s.Inc("db", 500)
	expect := "web, 200:2, 400:1\ndb, 500:1"
	ret := s.String()
	if ret != expect {
		t.Errorf("expect %s, but got %s", expect, ret)
	}
}

func TestRemove(t *testing.T) {
	s := New()
	s.Inc("web", 200)
	s.Inc("db", 500)
	s.Remove("web")
	if _, ok := s.StatusCode["web"]; ok {
		t.Errorf("web should be removed")
	}
}
