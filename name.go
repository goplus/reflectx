//go:build !go1.17 && (!js || (js && wasm))
// +build !go1.17
// +build !js js,wasm

package reflectx

import "unsafe"

// name is an encoded type name with optional extra data.
//
// The first byte is a bit field containing:
//
//	1<<0 the name is exported
//	1<<1 tag data follows the name
//	1<<2 pkgPath nameOff follows the name and tag
//
// The next two bytes are the data length:
//
//	 l := uint16(data[1])<<8 | uint16(data[2])
//
// Bytes [3:3+l] are the string data.
//
// If tag data follows then bytes 3+l and 3+l+1 are the tag length,
// with the data following.
//
// If the import path follows, then 4 bytes at the end of
// the data form a nameOff. The import path is only set for concrete
// methods that are defined in a different package than their type.
//
// If a name starts with "*", then the exported bit represents
// whether the pointed to type is exported.
type name struct {
	bytes *byte
}

func (n name) data(off int, whySafe string) *byte {
	return (*byte)(add(unsafe.Pointer(n.bytes), uintptr(off), whySafe))
}

func (n name) isExported() bool {
	return (*n.bytes)&(1<<0) != 0
}

func (n name) nameLen() int {
	return int(uint16(*n.data(1, "name len field"))<<8 | uint16(*n.data(2, "name len field")))
}

func (n name) tagLen() int {
	if *n.data(0, "name flag field")&(1<<1) == 0 {
		return 0
	}
	off := 3 + n.nameLen()
	return int(uint16(*n.data(off, "name taglen field"))<<8 | uint16(*n.data(off+1, "name taglen field")))
}

func (n name) name() (s string) {
	if n.bytes == nil {
		return
	}
	b := (*[4]byte)(unsafe.Pointer(n.bytes))

	hdr := (*stringHeader)(unsafe.Pointer(&s))
	hdr.Data = unsafe.Pointer(&b[3])
	hdr.Len = int(b[1])<<8 | int(b[2])
	return s
}

func (n name) tag() (s string) {
	tl := n.tagLen()
	if tl == 0 {
		return ""
	}
	nl := n.nameLen()
	hdr := (*stringHeader)(unsafe.Pointer(&s))
	hdr.Data = unsafe.Pointer(n.data(3+nl+2, "non-empty string"))
	hdr.Len = tl
	return s
}

func (n name) pkgPath() string {
	if n.bytes == nil || *n.data(0, "name flag field")&(1<<2) == 0 {
		return ""
	}
	off := 3 + n.nameLen()
	if tl := n.tagLen(); tl > 0 {
		off += 2 + tl
	}
	var nameOff int32

	// copy((*[4]byte)(unsafe.Pointer(n.data(off, "name offset field")))[:], (*[4]byte)(unsafe.Pointer(&nameOff))[:])

	// Note that this field may not be aligned in memory,
	// so we cannot use a direct int32 assignment here.
	copy((*[4]byte)(unsafe.Pointer(&nameOff))[:], (*[4]byte)(unsafe.Pointer(n.data(off, "name offset field")))[:])
	pkgPathName := name{(*byte)(resolveTypeOff(unsafe.Pointer(n.bytes), nameOff))}
	return pkgPathName.name()
}

func (n name) setPkgPath(pkgpath nameOff) bool {
	if n.bytes == nil || *n.data(0, "name flag field")&(1<<2) == 0 {
		return false
	}
	off := 3 + n.nameLen()
	if tl := n.tagLen(); tl > 0 {
		off += 2 + tl
	}
	copy((*[4]byte)(unsafe.Pointer(n.data(off, "name offset field")))[:], (*[4]byte)(unsafe.Pointer(&pkgpath))[:])
	return true
}

func newNameEx(n, tag string, exported bool, pkgpath bool) name {
	if len(n) > 1<<16-1 {
		panic("reflect.nameFrom: name too long: " + n)
	}
	if len(tag) > 1<<16-1 {
		panic("reflect.nameFrom: tag too long: " + tag)
	}

	var bits byte
	l := 1 + 2 + len(n)
	if exported {
		bits |= 1 << 0
	}
	if len(tag) > 0 {
		l += 2 + len(tag)
		bits |= 1 << 1
	}
	if !exported && pkgpath {
		bits |= 1 << 2
		l += 4
	}

	b := make([]byte, l)
	b[0] = bits
	b[1] = uint8(len(n) >> 8)
	b[2] = uint8(len(n))
	copy(b[3:], n)
	if len(tag) > 0 {
		tb := b[3+len(n):]
		tb[0] = uint8(len(tag) >> 8)
		tb[1] = uint8(len(tag))
		copy(tb[2:], tag)
	}
	return name{bytes: &b[0]}
}
