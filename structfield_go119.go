//go:build go1.19
// +build go1.19

package reflectx

// Struct field
type structField struct {
	name    name    // name is always non-empty
	typ     *rtype  // type of field
	_offset uintptr // byte offset of field
}

func (f *structField) offset() uintptr {
	return f._offset
}

func (f *structField) embedded() bool {
	return f.name.embedded()
}

func setEmbedded(f *structField) {
	f.name.setEmbedded()
}
