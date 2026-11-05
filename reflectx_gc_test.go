//go:build !llgo

package reflectx_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/goplus/reflectx"
)

func TestTypeLinksAndTypesByString(t *testing.T) {
	links := reflectx.TypeLinks()
	if len(links) == 0 {
		t.Fatal("TypeLinks empty")
	}
	_ = reflectx.TypesByString("int")
	_ = reflectx.TypesByString("this.type.does.not.exist.zzz")
	var buf bytes.Buffer
	reflectx.DumpType(&buf, reflect.TypeOf(nPoint{}))
	if buf.Len() == 0 {
		t.Fatal("DumpType empty")
	}
	reflectx.DumpType(&buf, reflect.TypeOf((*fmt.Stringer)(nil)).Elem())
}
