//go:build !llgo
// +build !llgo

package reflectx_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/goplus/reflectx"
	"github.com/goplus/reflectx/abi"
)

func TestIcallCachedCount(t *testing.T) {
	styp := reflectx.NamedStructOf("main", "icallCachedCount", []reflect.StructField{
		{Name: "P", Type: reflect.TypeOf((*int)(nil))},
	})
	typ := reflectx.NewMethodSet(styp, 1, 1)
	method := reflectx.MakeMethod(
		"Cached",
		"main",
		false,
		reflect.TypeOf(func() {}),
		func([]reflect.Value) []reflect.Value { return nil },
	)
	method.FuncId = 0x5245464c

	_, allocatedBefore, _ := reflectx.IcallStat()
	cachedBefore := reflectx.IcallCached()
	if err := reflectx.SetMethodSet(typ, []reflectx.Method{method}, false); err != nil {
		t.Fatal(err)
	}
	_, allocatedAfter, _ := reflectx.IcallStat()
	cachedAfter := reflectx.IcallCached()

	if got, want := allocatedAfter-allocatedBefore, 2; got != want {
		t.Fatalf("allocated icalls: got %d, want %d", got, want)
	}
	if got, want := cachedAfter-cachedBefore, allocatedAfter-allocatedBefore; got != want {
		t.Fatalf("cached icalls: got %d, want %d", got, want)
	}
}

func BenchmarkIcallProviderInsert(b *testing.B) {
	provider := abi.Default.List()[0]
	info := &abi.MethodInfo{
		Func:    reflect.ValueOf(func(*struct{}) {}),
		Type:    reflect.TypeOf(struct{}{}),
		Pointer: true,
	}
	levels := []int{0, provider.Cap() / 2, provider.Cap() - 1}

	for _, used := range levels {
		b.Run(fmt.Sprintf("Used%d", used), func(b *testing.B) {
			provider.Clear()
			for i := 0; i < used; i++ {
				if _, index := provider.Insert(info); index < 0 {
					b.Fatalf("prefill failed at %d", i)
				}
			}
			b.Cleanup(provider.Clear)

			indices := make([]int, 1)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, index := provider.Insert(info)
				if index < 0 {
					b.Fatal("insert failed")
				}
				indices[0] = index
				provider.Remove(indices)
			}
		})
	}
}
