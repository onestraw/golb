package stats

import "testing"

func TestInc(t *testing.T) {
	s := New()
	code := "200"
	data := &Data{
		StatusCode: code,
		InBytes:    24,
	}
	s.Inc(data)
	if s.StatusCode[code] != 1 {
		t.Errorf("The number of code %s should be 1", code)
	}
	if s.InBytes != 24 {
		t.Errorf("The received bytes should be 24")
	}
}

func TestString(t *testing.T) {
	s := New()
	data := &Data{
		StatusCode: "200",
		Method:     "GET",
		Path:       "/test",
		InBytes:    24,
		OutBytes:   1024,
	}
	s.Inc(data)
	expect := "status_code: 200:1\nmethod: GET:1\npath: /test:1\nrecv_bytes: 24\nsend_bytes: 1024"
	ret := s.String()
	if ret != expect {
		t.Errorf("expect %s, but got %s", expect, ret)
	}
}
