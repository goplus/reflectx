package reflectx

// structField is an alias to abi.StructField (defined in type.go)

func setEmbedded(f *structField) {
	(*f.Name.Bytes) |= 1 << 3
}
