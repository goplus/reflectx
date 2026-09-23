# iface_patch

可选：给 Go **1.25.x**、**1.26.x** 或 **1.27.x** 源码打补丁，使
`-tags goplus.ifacefuncval` 能用 MakeFunc funcval 替代 icall stub。
reflectx 把未打 tag 的 `*makeFuncImpl` 交给 `addReflectOff`（GC 可扫），
并设置 `tflagIfaceFuncval`（`1<<6`）。打过补丁的 runtime 在 `itabInit` /
`methodReceiver` 把 `itab.Fun` 写成 `makeFuncImpl*|1`。

支持的 `GOARCH`：**wasm**、**arm64**、**amd64**、**386**（该后端对应的任意 `GOOS`）。

tag 必须 **显式指定**（不是 ToolTag）。不加 tag 时，打过补丁的编译器生成的
接口调用序列与未修改的 Go 相同。

reflectx **不会**检查编译器会不会 unwrap tagged ifn。官方 gc 带上这个 tag
仍能编译；接口方法调用会在运行时崩溃（`uninitialized element` / 非法 PC）。

## 打补丁并重编

安装：

```
go install github.com/goplus/reflectx/cmd/iface_patch@latest
```

`iface_patch` 只改源码。必须在该源码树里用 **`make.bash`** 重编
（只 `go install cmd/asm cmd/compile cmd/go` 不够）：

```shell
iface_patch /path/to/go
iface_patch -check /path/to/go
cd /path/to/go/src && ./make.bash   # Windows: make.bat
export GOROOT=/path/to/go
export PATH="$GOROOT/bin:$PATH"
go version   # 来自 $GOROOT
```

对已打过补丁的树再跑一遍是空操作。支持的版本是 `_data/` 下的目录名
（`iface_patch -h` 会列出）。

`go clean -cache` 可以清构建缓存，不是必须步骤。

不要把这份 `GOROOT` 和 `PATH` 里另一个 `go` 混用。

## 跑测试

### Native（linux/darwin 的 amd64、arm64 或 386）

```shell
export GOROOT=/path/to/go
export PATH="$GOROOT/bin:$PATH"
go test -tags goplus.ifacefuncval -v .
```

`make.bash` 装进 GOROOT 的标准库 **没有** `-ifacefuncval`。带 tag 的
`go test` 会把 `fmt`/`runtime` 重新编进 `GOCACHE`，它们的接口调用才会 unwrap。

### wasip1

本机跑不了 wasip1 二进制。先安装 [Wasmtime](https://wasmtime.dev/)，并加上
`-exec wasmtime`：

```shell
export GOROOT=/path/to/go
export PATH="$GOROOT/bin:$PATH"
GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -tags goplus.ifacefuncval -v .
```

下面这样 **不会**真正跑测试：

```shell
GOOS=wasip1 GOARCH=wasm go test -tags goplus.ifacefuncval .
```

### GOFLAGS

```shell
export GOFLAGS='-tags=goplus.ifacefuncval'
go test -v .                                          # native
GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -v .    # wasip1
```

若 `GOFLAGS` 里已有 `-tags`，需要合并：

```shell
export GOFLAGS='-tags=netgo,goplus.ifacefuncval'
```

关掉扩展：去掉这个 tag。`GOFLAGS` 不能做 tag 减法
（`-tags=-goplus.ifacefuncval` 无效）。

只设 `GOFLAGS=-gcflags=-ifacefuncval` 不够：那只会 unwrap 调用。
没有 build tag 时，reflectx 仍走 icall。

## 补丁改了什么

unwrap **仅在** `objabi.EnableIfaceFuncval` 为真时执行（`-ifacefuncval`）。
无 tag 的编译结果必须与官方 gc 一致。

| 位置 | 改动 |
|---|---|
| `cmd/internal/objabi/ifacefuncval.go` | `EnableIfaceFuncval`（共用，不是 wasm 专用） |
| `cmd/compile`、`cmd/asm` | wasm/arm64/amd64/386 上接受 `-ifacefuncval` |
| `cmd/go` | `GOARCH` 为 wasm/arm64/amd64/386 且带 tag 时，把 `-ifacefuncval` 加进 `forcedGcflags`/`forcedAsmflags` |
| `cmd/internal/obj/wasm` | 间接 `CALL` 时 unwrap 打标 PC |
| `cmd/compile/internal/arm64` | `CALLinter` / `CALLtailinter` 前 unwrap（CTXT=R26） |
| `cmd/compile/internal/amd64` | `CALLinter` / `CALLtailinter` 前 unwrap（CTXT=DX，经 R12 调用） |
| `cmd/compile/internal/x86` | `CALLinter` / `CALLtailinter` 前 unwrap（CTXT=DX，32 位） |
| `runtime/asm_arm64.s` | `CALLFN` 里 `BL (R20)` 前 unwrap |
| `runtime/asm_amd64.s` | `CALLFN` 里 `CALL R12` 前 unwrap |
| `runtime/asm_386.s` | `CALLFN` 里 `CALL AX` 前 unwrap |
| `runtime/iface.go` | `itabInit` 在类型 tflag bit 6（`tflagIfaceFuncval`）置位时给 `Fun` 打 tag |
| `runtime/iface_funcval.go` | `itabFuncval` 辅助函数 |
| `reflect/value.go` | `methodReceiver` 在 tflag bit 6 置位时给 Ifn 打 tag |

打标的 `itab.Fun` 是 `makeFuncImpl* | 1`。代码 PC 和堆指针都是偶数，bit0 空闲。
unwrap 把 CTXT 设成去掉标记的指针，并调用第一字（`makeFuncStub`）。

## `_data/`

按 `VERSION` 匹配目录名：

1. 若存在精确目录 `_data/go1.N.x/`，用它
2. 否则用 `_data/go1.N/`（`go1.25.14` → `go1.25`）

当前系列目录及对应源码树：

| 目录 | 来源 |
|---|---|
| `go1.25` | go1.25.14 |
| `go1.26` | go1.26.8 |
| `go1.27` | go1.27.1 |

Go 片段带 `//go:build ignore`。CALLFN 的 old/new 汇编是普通 `.s`，按原文精确替换。

| 文件 | 作用 |
|---|---|
| `objabi_ifacefuncval.go` | 写入 `src/cmd/internal/objabi/ifacefuncval.go` |
| `compile.go` / `asm.go` | 设置 `objabi.EnableIfaceFuncval` |
| `gc.go` / `gc_flags.go` | `ifaceFuncvalEnabled` + forced flags |
| `wasmobj.go` / `unwrap.go` | wasm 间接调用 unwrap |
| `arm64_ssa.go`、`arm64_callinter.go`、`arm64_calltailinter.go` | arm64 编译器 |
| `amd64_ssa.go`、`amd64_callinter.go`、`amd64_calltailinter.go` | amd64 编译器 |
| `x86_ssa.go`、`x86_callinter.go`、`x86_calltailinter.go` | 386 编译器 |
| `iface_funcval.go` | 写入 `src/runtime/iface_funcval.go` |
| `iface_fun0_*.txt` / `iface_ifn_*.txt` / `iface_ifn_ptr_*.txt` / `iface_fun0store_*.txt` | `runtime/iface.go` 的 `itabInit` |
| `reflect_method_*.txt` | `reflect/value.go` 的 `methodReceiver` |
| `arm64_callfn_{old,new}.s` | `runtime/asm_arm64.s` 的 `CALLFN` |
| `amd64_callfn_{old,new}.s` | `runtime/asm_amd64.s` 的 `CALLFN` |
| `386_callfn_{old,new}.s` | `runtime/asm_386.s` 的 `CALLFN` |

要支持新的系列：把 `_data/go1.27/` 复制为 `_data/go1.28/`。
某个补丁版本若分叉，复制为 `_data/go1.27.2/`（精确匹配优先）。
改片段直到 `iface_patch` 能匹配那棵源码树。
