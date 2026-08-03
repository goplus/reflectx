//go:build go1.23

package syncmap

import (
	"sync"
)

func Clear(m sync.Map) {
	m.Clear()
}
