//go:build !go1.19
// +build !go1.19

package reflectx

// struct field
type structField struct {
	name        name    // name is always non-empty
	typ         *rtype  // type of field
	offsetEmbed uintptr // byte offset of field<<1 | isEmbedded
}

func (f *structField) offset() uintptr {
	return f.offsetEmbed >> 1
}

func (f *structField) embedded() bool {
	return f.offsetEmbed&1 != 0
}

func setEmbedded(f *structField) {
	f.offsetEmbed |= 1
}
