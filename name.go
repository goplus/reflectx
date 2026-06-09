package reflectx

import "unsafe"

// name is an alias to abi.Name (defined in type.go)
// Helper methods below mirror the original name methods.

// writeVarint writes n to buf in varint form. Returns the
// number of bytes written. n must be nonnegative.
// Writes at most 10 bytes.
func writeVarint(buf []byte, n int) int {
	for i := 0; ; i++ {
		b := byte(n & 0x7f)
		n >>= 7
		if n == 0 {
			buf[i] = b
			return i + 1
		}
		buf[i] = b | 0x80
	}
}

func setPkgPath(n name, pkgpath string) {
	if n.Bytes == nil || *n.DataChecked(0, "name flag pkgPath")&(1<<2) == 0 {
		return
	}
	i, l := n.ReadVarint(1)
	off := 1 + i + l
	if n.HasTag() {
		i2, l2 := n.ReadVarint(off)
		off += i2 + l2
	}
	v := resolveReflectName(newName(pkgpath, "", false))
	copy((*[4]byte)(unsafe.Pointer(n.DataChecked(off, "name offset pkgPath")))[:], (*[4]byte)(unsafe.Pointer(&v))[:])
}

func newNameEx(n, tag string, exported bool, pkgpath bool) name {
	if len(n) >= 1<<29 {
		panic("reflect.nameFrom: name too long: " + n[:1024] + "...")
	}
	if len(tag) >= 1<<29 {
		panic("reflect.nameFrom: tag too long: " + tag[:1024] + "...")
	}
	var nameLen [10]byte
	var tagLen [10]byte
	nameLenLen := writeVarint(nameLen[:], len(n))
	tagLenLen := writeVarint(tagLen[:], len(tag))

	var bits byte
	l := 1 + nameLenLen + len(n)
	if exported {
		bits |= 1 << 0
	}
	if len(tag) > 0 {
		l += tagLenLen + len(tag)
		bits |= 1 << 1
	}
	if !exported && pkgpath {
		bits |= 1 << 2
		l += 4
	}

	b := make([]byte, l)
	b[0] = bits
	copy(b[1:], nameLen[:nameLenLen])
	copy(b[1+nameLenLen:], n)
	if len(tag) > 0 {
		tb := b[1+nameLenLen+len(n):]
		copy(tb, tagLen[:tagLenLen])
		copy(tb[tagLenLen:], tag)
	}

	return name{Bytes: &b[0]}
}

func newName(n, tag string, exported bool) name {
	return newNameEx(n, tag, exported, false)
}
