package icall

import (
	"reflect"
	"sync"
	"testing"

	"github.com/goplus/reflectx/abi"
)

type providerTestReceiver struct{}

func TestProviderConcurrentAccess(t *testing.T) {
	mp.Clear()
	t.Cleanup(mp.Clear)

	info := &abi.MethodInfo{
		Func:    reflect.ValueOf(func(*providerTestReceiver) {}),
		Type:    reflect.TypeOf(providerTestReceiver{}),
		Pointer: true,
	}

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_, index := mp.Insert(info)
				if index >= 0 {
					mp.Remove([]int{index})
				}
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = mp.lookup(0)
				_ = mp.Used()
				_ = mp.Available()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 1000; j++ {
			mp.Clear()
		}
	}()
	wg.Wait()
}
