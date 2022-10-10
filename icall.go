//go:build (!go1.17 || (go1.17 && !go1.18 && !goexperiment.regabireflect) || (go1.18 && !go1.19 && !goexperiment.regabireflect && !amd64) || (go1.19 && !goexperiment.regabiargs && !amd64 && !arm64 && !ppc64 && !ppc64le)) && (!js || (js && wasm))
// +build !go1.17 go1.17,!go1.18,!goexperiment.regabireflect go1.18,!go1.19,!goexperiment.regabireflect,!amd64 go1.19,!goexperiment.regabiargs,!amd64,!arm64,!ppc64,!ppc64le
// +build !js js,wasm

package reflectx

import (
	"reflect"
	"unsafe"
)

const capacity = 256

type provider struct {
	infos []*MethodInfo
}

func (p *provider) Push(info *MethodInfo) (ifn unsafe.Pointer) {
	fn := icall_array[len(p.infos)]
	p.infos = append(p.infos, info)
	return unsafe.Pointer(reflect.ValueOf(fn).Pointer())
}

func (p *provider) Len() int {
	return len(p.infos)
}

func (p *provider) Cap() int {
	return len(icall_array)
}

func (p *provider) Clear() {
	p.infos = nil
}

var (
	mp provider
)

func init() {
	AddMethodProvider(&mp)
}

func i_x(index int, ptr unsafe.Pointer, p unsafe.Pointer) {
	info := mp.infos[index]
	var receiver reflect.Value
	if !info.Pointer && info.OnePtr {
		receiver = reflect.NewAt(info.Type, unsafe.Pointer(&ptr)).Elem() //.Elem().Field(0)
	} else {
		receiver = reflect.NewAt(info.Type, ptr)
		if !info.Pointer || info.Indirect {
			receiver = receiver.Elem()
		}
	}
	in := []reflect.Value{receiver}
	if inCount := info.Func.Type().NumIn(); inCount > 1 {
		sz := info.InTyp.Size()
		buf := make([]byte, sz, sz)
		if sz > info.InSize {
			sz = info.InSize
		}
		for i := uintptr(0); i < sz; i++ {
			buf[i] = *(*byte)(add(p, i, ""))
		}
		var inArgs reflect.Value
		if sz == 0 {
			inArgs = reflect.New(info.InTyp).Elem()
		} else {
			inArgs = reflect.NewAt(info.InTyp, unsafe.Pointer(&buf[0])).Elem()
		}
		for i := 1; i < inCount; i++ {
			in = append(in, inArgs.Field(i-1))
		}
	}
	var r []reflect.Value
	if info.Variadic {
		r = info.Func.CallSlice(in)
	} else {
		r = info.Func.Call(in)
	}
	if info.OutTyp.NumField() > 0 {
		out := reflect.New(info.OutTyp).Elem()
		for i, v := range r {
			out.Field(i).Set(v)
		}
		po := unsafe.Pointer(out.UnsafeAddr())
		for i := uintptr(0); i < info.OutSize; i++ {
			*(*byte)(add(p, info.InSize+i, "")) = *(*byte)(add(po, uintptr(i), ""))
		}
	}
}

type unsafeptr = unsafe.Pointer

var icall_array = []interface{}{
	func(p, a unsafeptr) { i_x(0, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(1, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(2, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(3, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(4, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(5, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(6, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(7, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(8, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(9, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(10, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(11, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(12, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(13, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(14, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(15, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(16, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(17, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(18, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(19, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(20, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(21, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(22, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(23, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(24, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(25, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(26, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(27, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(28, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(29, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(30, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(31, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(32, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(33, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(34, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(35, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(36, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(37, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(38, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(39, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(40, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(41, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(42, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(43, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(44, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(45, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(46, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(47, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(48, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(49, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(50, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(51, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(52, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(53, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(54, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(55, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(56, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(57, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(58, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(59, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(60, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(61, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(62, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(63, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(64, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(65, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(66, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(67, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(68, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(69, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(70, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(71, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(72, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(73, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(74, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(75, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(76, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(77, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(78, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(79, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(80, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(81, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(82, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(83, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(84, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(85, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(86, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(87, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(88, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(89, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(90, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(91, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(92, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(93, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(94, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(95, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(96, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(97, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(98, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(99, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(100, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(101, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(102, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(103, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(104, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(105, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(106, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(107, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(108, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(109, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(110, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(111, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(112, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(113, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(114, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(115, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(116, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(117, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(118, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(119, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(120, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(121, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(122, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(123, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(124, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(125, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(126, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(127, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(128, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(129, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(130, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(131, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(132, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(133, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(134, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(135, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(136, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(137, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(138, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(139, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(140, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(141, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(142, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(143, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(144, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(145, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(146, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(147, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(148, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(149, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(150, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(151, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(152, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(153, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(154, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(155, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(156, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(157, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(158, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(159, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(160, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(161, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(162, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(163, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(164, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(165, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(166, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(167, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(168, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(169, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(170, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(171, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(172, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(173, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(174, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(175, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(176, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(177, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(178, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(179, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(180, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(181, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(182, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(183, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(184, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(185, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(186, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(187, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(188, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(189, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(190, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(191, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(192, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(193, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(194, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(195, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(196, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(197, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(198, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(199, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(200, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(201, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(202, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(203, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(204, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(205, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(206, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(207, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(208, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(209, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(210, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(211, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(212, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(213, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(214, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(215, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(216, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(217, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(218, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(219, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(220, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(221, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(222, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(223, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(224, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(225, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(226, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(227, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(228, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(229, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(230, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(231, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(232, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(233, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(234, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(235, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(236, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(237, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(238, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(239, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(240, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(241, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(242, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(243, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(244, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(245, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(246, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(247, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(248, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(249, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(250, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(251, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(252, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(253, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(254, p, unsafeptr(&a)) },
	func(p, a unsafeptr) { i_x(255, p, unsafeptr(&a)) },
}
