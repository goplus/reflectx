//go:build go1.17 && goexperiment.regabireflect
// +build go1.17,goexperiment.regabireflect

package reflectx

import (
	"reflect"
	"unsafe"
)

const capacity = 256

type provider struct {
}

//go:linkname callReflect reflect.callReflect
func callReflect(ctxt unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer)

//go:linkname moveMakeFuncArgPtrs reflect.moveMakeFuncArgPtrs
func moveMakeFuncArgPtrs(ctx unsafe.Pointer, r unsafe.Pointer)

var infos []*MethodInfo
var funcs []reflect.Value
var fnptr []unsafe.Pointer

//go:nosplit
func i_x(index int, c unsafe.Pointer, frame unsafe.Pointer, retValid *bool, r unsafe.Pointer) {
	moveMakeFuncArgPtrs(fnptr[index], r)
	callReflect(fnptr[index], unsafe.Pointer(uintptr(frame)+ptrSize), retValid, r)
}

const ptrSize = (32 << (^uint(0) >> 63)) / 8

func spillArgs()
func unspillArgs()

func (p *provider) Push(info *MethodInfo) (ifn unsafe.Pointer) {
	fn := icall_fn[len(infos)]
	infos = append(infos, info)

	ftyp := info.Func.Type()
	toPtr := (!info.Pointer && !info.OnePtr) || info.Indirect
	if toPtr {
		numIn := ftyp.NumIn()
		numOut := ftyp.NumOut()
		in := make([]reflect.Type, numIn, numIn)
		out := make([]reflect.Type, numOut, numOut)
		in[0] = reflect.PtrTo(info.Type)
		for i := 1; i < numIn; i++ {
			in[i] = ftyp.In(i)
		}
		for i := 0; i < numOut; i++ {
			out[i] = ftyp.Out(i)
		}
		ftyp = reflect.FuncOf(in, out, ftyp.IsVariadic())
	}
	v := reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
		if toPtr {
			args[0] = args[0].Elem()
		}
		if info.Variadic {
			return info.Func.CallSlice(args)
		}
		return info.Func.Call(args)
	})
	funcs = append(funcs, v)
	fnptr = append(fnptr, tovalue(&v).ptr)

	return unsafe.Pointer(reflect.ValueOf(fn).Pointer())
}

func (p *provider) Len() int {
	return len(infos)
}

func (p *provider) Cap() int {
	return capacity
}

func (p *provider) Clear() {
	infos = nil
}

var (
	mp provider
)

func init() {
	AddMethodProvider(&mp)
}

type unsafeptr = unsafe.Pointer

func f0()
func f1()
func f2()
func f3()
func f4()
func f5()
func f6()
func f7()
func f8()
func f9()
func f10()
func f11()
func f12()
func f13()
func f14()
func f15()
func f16()
func f17()
func f18()
func f19()
func f20()
func f21()
func f22()
func f23()
func f24()
func f25()
func f26()
func f27()
func f28()
func f29()
func f30()
func f31()
func f32()
func f33()
func f34()
func f35()
func f36()
func f37()
func f38()
func f39()
func f40()
func f41()
func f42()
func f43()
func f44()
func f45()
func f46()
func f47()
func f48()
func f49()
func f50()
func f51()
func f52()
func f53()
func f54()
func f55()
func f56()
func f57()
func f58()
func f59()
func f60()
func f61()
func f62()
func f63()
func f64()
func f65()
func f66()
func f67()
func f68()
func f69()
func f70()
func f71()
func f72()
func f73()
func f74()
func f75()
func f76()
func f77()
func f78()
func f79()
func f80()
func f81()
func f82()
func f83()
func f84()
func f85()
func f86()
func f87()
func f88()
func f89()
func f90()
func f91()
func f92()
func f93()
func f94()
func f95()
func f96()
func f97()
func f98()
func f99()
func f100()
func f101()
func f102()
func f103()
func f104()
func f105()
func f106()
func f107()
func f108()
func f109()
func f110()
func f111()
func f112()
func f113()
func f114()
func f115()
func f116()
func f117()
func f118()
func f119()
func f120()
func f121()
func f122()
func f123()
func f124()
func f125()
func f126()
func f127()
func f128()
func f129()
func f130()
func f131()
func f132()
func f133()
func f134()
func f135()
func f136()
func f137()
func f138()
func f139()
func f140()
func f141()
func f142()
func f143()
func f144()
func f145()
func f146()
func f147()
func f148()
func f149()
func f150()
func f151()
func f152()
func f153()
func f154()
func f155()
func f156()
func f157()
func f158()
func f159()
func f160()
func f161()
func f162()
func f163()
func f164()
func f165()
func f166()
func f167()
func f168()
func f169()
func f170()
func f171()
func f172()
func f173()
func f174()
func f175()
func f176()
func f177()
func f178()
func f179()
func f180()
func f181()
func f182()
func f183()
func f184()
func f185()
func f186()
func f187()
func f188()
func f189()
func f190()
func f191()
func f192()
func f193()
func f194()
func f195()
func f196()
func f197()
func f198()
func f199()
func f200()
func f201()
func f202()
func f203()
func f204()
func f205()
func f206()
func f207()
func f208()
func f209()
func f210()
func f211()
func f212()
func f213()
func f214()
func f215()
func f216()
func f217()
func f218()
func f219()
func f220()
func f221()
func f222()
func f223()
func f224()
func f225()
func f226()
func f227()
func f228()
func f229()
func f230()
func f231()
func f232()
func f233()
func f234()
func f235()
func f236()
func f237()
func f238()
func f239()
func f240()
func f241()
func f242()
func f243()
func f244()
func f245()
func f246()
func f247()
func f248()
func f249()
func f250()
func f251()
func f252()
func f253()
func f254()
func f255()

var (
	icall_fn = []func(){f0, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, f15, f16, f17, f18, f19, f20, f21, f22, f23, f24, f25, f26, f27, f28, f29, f30, f31, f32, f33, f34, f35, f36, f37, f38, f39, f40, f41, f42, f43, f44, f45, f46, f47, f48, f49, f50, f51, f52, f53, f54, f55, f56, f57, f58, f59, f60, f61, f62, f63, f64, f65, f66, f67, f68, f69, f70, f71, f72, f73, f74, f75, f76, f77, f78, f79, f80, f81, f82, f83, f84, f85, f86, f87, f88, f89, f90, f91, f92, f93, f94, f95, f96, f97, f98, f99, f100, f101, f102, f103, f104, f105, f106, f107, f108, f109, f110, f111, f112, f113, f114, f115, f116, f117, f118, f119, f120, f121, f122, f123, f124, f125, f126, f127, f128, f129, f130, f131, f132, f133, f134, f135, f136, f137, f138, f139, f140, f141, f142, f143, f144, f145, f146, f147, f148, f149, f150, f151, f152, f153, f154, f155, f156, f157, f158, f159, f160, f161, f162, f163, f164, f165, f166, f167, f168, f169, f170, f171, f172, f173, f174, f175, f176, f177, f178, f179, f180, f181, f182, f183, f184, f185, f186, f187, f188, f189, f190, f191, f192, f193, f194, f195, f196, f197, f198, f199, f200, f201, f202, f203, f204, f205, f206, f207, f208, f209, f210, f211, f212, f213, f214, f215, f216, f217, f218, f219, f220, f221, f222, f223, f224, f225, f226, f227, f228, f229, f230, f231, f232, f233, f234, f235, f236, f237, f238, f239, f240, f241, f242, f243, f244, f245, f246, f247, f248, f249, f250, f251, f252, f253, f254, f255}
)

func x0(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(0, c, f, v, r)
}
func x1(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(1, c, f, v, r)
}
func x2(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(2, c, f, v, r)
}
func x3(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(3, c, f, v, r)
}
func x4(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(4, c, f, v, r)
}
func x5(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(5, c, f, v, r)
}
func x6(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(6, c, f, v, r)
}
func x7(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(7, c, f, v, r)
}
func x8(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(8, c, f, v, r)
}
func x9(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(9, c, f, v, r)
}
func x10(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(10, c, f, v, r)
}
func x11(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(11, c, f, v, r)
}
func x12(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(12, c, f, v, r)
}
func x13(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(13, c, f, v, r)
}
func x14(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(14, c, f, v, r)
}
func x15(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(15, c, f, v, r)
}
func x16(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(16, c, f, v, r)
}
func x17(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(17, c, f, v, r)
}
func x18(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(18, c, f, v, r)
}
func x19(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(19, c, f, v, r)
}
func x20(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(20, c, f, v, r)
}
func x21(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(21, c, f, v, r)
}
func x22(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(22, c, f, v, r)
}
func x23(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(23, c, f, v, r)
}
func x24(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(24, c, f, v, r)
}
func x25(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(25, c, f, v, r)
}
func x26(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(26, c, f, v, r)
}
func x27(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(27, c, f, v, r)
}
func x28(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(28, c, f, v, r)
}
func x29(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(29, c, f, v, r)
}
func x30(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(30, c, f, v, r)
}
func x31(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(31, c, f, v, r)
}
func x32(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(32, c, f, v, r)
}
func x33(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(33, c, f, v, r)
}
func x34(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(34, c, f, v, r)
}
func x35(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(35, c, f, v, r)
}
func x36(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(36, c, f, v, r)
}
func x37(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(37, c, f, v, r)
}
func x38(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(38, c, f, v, r)
}
func x39(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(39, c, f, v, r)
}
func x40(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(40, c, f, v, r)
}
func x41(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(41, c, f, v, r)
}
func x42(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(42, c, f, v, r)
}
func x43(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(43, c, f, v, r)
}
func x44(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(44, c, f, v, r)
}
func x45(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(45, c, f, v, r)
}
func x46(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(46, c, f, v, r)
}
func x47(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(47, c, f, v, r)
}
func x48(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(48, c, f, v, r)
}
func x49(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(49, c, f, v, r)
}
func x50(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(50, c, f, v, r)
}
func x51(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(51, c, f, v, r)
}
func x52(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(52, c, f, v, r)
}
func x53(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(53, c, f, v, r)
}
func x54(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(54, c, f, v, r)
}
func x55(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(55, c, f, v, r)
}
func x56(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(56, c, f, v, r)
}
func x57(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(57, c, f, v, r)
}
func x58(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(58, c, f, v, r)
}
func x59(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(59, c, f, v, r)
}
func x60(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(60, c, f, v, r)
}
func x61(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(61, c, f, v, r)
}
func x62(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(62, c, f, v, r)
}
func x63(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(63, c, f, v, r)
}
func x64(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(64, c, f, v, r)
}
func x65(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(65, c, f, v, r)
}
func x66(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(66, c, f, v, r)
}
func x67(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(67, c, f, v, r)
}
func x68(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(68, c, f, v, r)
}
func x69(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(69, c, f, v, r)
}
func x70(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(70, c, f, v, r)
}
func x71(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(71, c, f, v, r)
}
func x72(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(72, c, f, v, r)
}
func x73(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(73, c, f, v, r)
}
func x74(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(74, c, f, v, r)
}
func x75(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(75, c, f, v, r)
}
func x76(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(76, c, f, v, r)
}
func x77(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(77, c, f, v, r)
}
func x78(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(78, c, f, v, r)
}
func x79(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(79, c, f, v, r)
}
func x80(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(80, c, f, v, r)
}
func x81(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(81, c, f, v, r)
}
func x82(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(82, c, f, v, r)
}
func x83(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(83, c, f, v, r)
}
func x84(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(84, c, f, v, r)
}
func x85(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(85, c, f, v, r)
}
func x86(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(86, c, f, v, r)
}
func x87(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(87, c, f, v, r)
}
func x88(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(88, c, f, v, r)
}
func x89(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(89, c, f, v, r)
}
func x90(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(90, c, f, v, r)
}
func x91(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(91, c, f, v, r)
}
func x92(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(92, c, f, v, r)
}
func x93(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(93, c, f, v, r)
}
func x94(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(94, c, f, v, r)
}
func x95(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(95, c, f, v, r)
}
func x96(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(96, c, f, v, r)
}
func x97(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(97, c, f, v, r)
}
func x98(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(98, c, f, v, r)
}
func x99(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(99, c, f, v, r)
}
func x100(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(100, c, f, v, r)
}
func x101(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(101, c, f, v, r)
}
func x102(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(102, c, f, v, r)
}
func x103(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(103, c, f, v, r)
}
func x104(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(104, c, f, v, r)
}
func x105(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(105, c, f, v, r)
}
func x106(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(106, c, f, v, r)
}
func x107(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(107, c, f, v, r)
}
func x108(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(108, c, f, v, r)
}
func x109(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(109, c, f, v, r)
}
func x110(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(110, c, f, v, r)
}
func x111(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(111, c, f, v, r)
}
func x112(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(112, c, f, v, r)
}
func x113(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(113, c, f, v, r)
}
func x114(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(114, c, f, v, r)
}
func x115(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(115, c, f, v, r)
}
func x116(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(116, c, f, v, r)
}
func x117(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(117, c, f, v, r)
}
func x118(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(118, c, f, v, r)
}
func x119(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(119, c, f, v, r)
}
func x120(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(120, c, f, v, r)
}
func x121(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(121, c, f, v, r)
}
func x122(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(122, c, f, v, r)
}
func x123(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(123, c, f, v, r)
}
func x124(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(124, c, f, v, r)
}
func x125(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(125, c, f, v, r)
}
func x126(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(126, c, f, v, r)
}
func x127(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(127, c, f, v, r)
}
func x128(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(128, c, f, v, r)
}
func x129(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(129, c, f, v, r)
}
func x130(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(130, c, f, v, r)
}
func x131(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(131, c, f, v, r)
}
func x132(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(132, c, f, v, r)
}
func x133(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(133, c, f, v, r)
}
func x134(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(134, c, f, v, r)
}
func x135(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(135, c, f, v, r)
}
func x136(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(136, c, f, v, r)
}
func x137(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(137, c, f, v, r)
}
func x138(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(138, c, f, v, r)
}
func x139(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(139, c, f, v, r)
}
func x140(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(140, c, f, v, r)
}
func x141(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(141, c, f, v, r)
}
func x142(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(142, c, f, v, r)
}
func x143(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(143, c, f, v, r)
}
func x144(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(144, c, f, v, r)
}
func x145(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(145, c, f, v, r)
}
func x146(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(146, c, f, v, r)
}
func x147(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(147, c, f, v, r)
}
func x148(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(148, c, f, v, r)
}
func x149(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(149, c, f, v, r)
}
func x150(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(150, c, f, v, r)
}
func x151(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(151, c, f, v, r)
}
func x152(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(152, c, f, v, r)
}
func x153(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(153, c, f, v, r)
}
func x154(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(154, c, f, v, r)
}
func x155(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(155, c, f, v, r)
}
func x156(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(156, c, f, v, r)
}
func x157(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(157, c, f, v, r)
}
func x158(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(158, c, f, v, r)
}
func x159(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(159, c, f, v, r)
}
func x160(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(160, c, f, v, r)
}
func x161(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(161, c, f, v, r)
}
func x162(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(162, c, f, v, r)
}
func x163(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(163, c, f, v, r)
}
func x164(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(164, c, f, v, r)
}
func x165(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(165, c, f, v, r)
}
func x166(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(166, c, f, v, r)
}
func x167(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(167, c, f, v, r)
}
func x168(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(168, c, f, v, r)
}
func x169(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(169, c, f, v, r)
}
func x170(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(170, c, f, v, r)
}
func x171(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(171, c, f, v, r)
}
func x172(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(172, c, f, v, r)
}
func x173(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(173, c, f, v, r)
}
func x174(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(174, c, f, v, r)
}
func x175(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(175, c, f, v, r)
}
func x176(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(176, c, f, v, r)
}
func x177(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(177, c, f, v, r)
}
func x178(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(178, c, f, v, r)
}
func x179(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(179, c, f, v, r)
}
func x180(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(180, c, f, v, r)
}
func x181(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(181, c, f, v, r)
}
func x182(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(182, c, f, v, r)
}
func x183(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(183, c, f, v, r)
}
func x184(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(184, c, f, v, r)
}
func x185(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(185, c, f, v, r)
}
func x186(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(186, c, f, v, r)
}
func x187(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(187, c, f, v, r)
}
func x188(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(188, c, f, v, r)
}
func x189(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(189, c, f, v, r)
}
func x190(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(190, c, f, v, r)
}
func x191(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(191, c, f, v, r)
}
func x192(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(192, c, f, v, r)
}
func x193(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(193, c, f, v, r)
}
func x194(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(194, c, f, v, r)
}
func x195(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(195, c, f, v, r)
}
func x196(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(196, c, f, v, r)
}
func x197(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(197, c, f, v, r)
}
func x198(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(198, c, f, v, r)
}
func x199(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(199, c, f, v, r)
}
func x200(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(200, c, f, v, r)
}
func x201(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(201, c, f, v, r)
}
func x202(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(202, c, f, v, r)
}
func x203(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(203, c, f, v, r)
}
func x204(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(204, c, f, v, r)
}
func x205(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(205, c, f, v, r)
}
func x206(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(206, c, f, v, r)
}
func x207(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(207, c, f, v, r)
}
func x208(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(208, c, f, v, r)
}
func x209(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(209, c, f, v, r)
}
func x210(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(210, c, f, v, r)
}
func x211(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(211, c, f, v, r)
}
func x212(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(212, c, f, v, r)
}
func x213(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(213, c, f, v, r)
}
func x214(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(214, c, f, v, r)
}
func x215(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(215, c, f, v, r)
}
func x216(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(216, c, f, v, r)
}
func x217(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(217, c, f, v, r)
}
func x218(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(218, c, f, v, r)
}
func x219(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(219, c, f, v, r)
}
func x220(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(220, c, f, v, r)
}
func x221(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(221, c, f, v, r)
}
func x222(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(222, c, f, v, r)
}
func x223(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(223, c, f, v, r)
}
func x224(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(224, c, f, v, r)
}
func x225(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(225, c, f, v, r)
}
func x226(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(226, c, f, v, r)
}
func x227(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(227, c, f, v, r)
}
func x228(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(228, c, f, v, r)
}
func x229(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(229, c, f, v, r)
}
func x230(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(230, c, f, v, r)
}
func x231(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(231, c, f, v, r)
}
func x232(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(232, c, f, v, r)
}
func x233(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(233, c, f, v, r)
}
func x234(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(234, c, f, v, r)
}
func x235(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(235, c, f, v, r)
}
func x236(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(236, c, f, v, r)
}
func x237(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(237, c, f, v, r)
}
func x238(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(238, c, f, v, r)
}
func x239(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(239, c, f, v, r)
}
func x240(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(240, c, f, v, r)
}
func x241(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(241, c, f, v, r)
}
func x242(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(242, c, f, v, r)
}
func x243(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(243, c, f, v, r)
}
func x244(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(244, c, f, v, r)
}
func x245(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(245, c, f, v, r)
}
func x246(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(246, c, f, v, r)
}
func x247(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(247, c, f, v, r)
}
func x248(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(248, c, f, v, r)
}
func x249(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(249, c, f, v, r)
}
func x250(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(250, c, f, v, r)
}
func x251(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(251, c, f, v, r)
}
func x252(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(252, c, f, v, r)
}
func x253(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(253, c, f, v, r)
}
func x254(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(254, c, f, v, r)
}
func x255(c unsafeptr, f unsafeptr, v *bool, r unsafeptr) {
	i_x(255, c, f, v, r)
}
