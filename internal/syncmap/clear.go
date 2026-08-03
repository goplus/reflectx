//go:build !go1.23

package syncmap

import (
	"sync"
)

func Clear(m sync.Map) {
	m.Range(func(k, v interface{}) bool {
		m.Delete(k)
		return true
	})
}
