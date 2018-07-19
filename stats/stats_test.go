package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInc(t *testing.T) {
	s := New()
	code := "200"
	data := &Data{
		StatusCode: code,
		InBytes:    24,
	}
	s.Inc(data)
	assert.Equal(t, uint64(1), s.StatusCode[code])
	assert.Equal(t, uint64(24), s.InBytes)
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
	assert.Equal(t, expect, s.String())
}
