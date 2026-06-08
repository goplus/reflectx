package reflectx

// structField is an alias to abi.StructField (defined in type.go)

func structFieldOffset(f *structField) uintptr {
	return f.Offset
}

func structFieldEmbedded(f *structField) bool {
	return f.Name.IsEmbedded()
}

func setEmbedded(f *structField) {
	(*f.Name.Bytes) |= 1 << 3
}
