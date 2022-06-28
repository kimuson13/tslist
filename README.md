# tslist
tslist shows the list about types satisfied by type sets
## Caution
tslist is intended to be used for Go1.18 or higher
## Install
tslist:
```
$ go install github.com/kimuson13/tslist/cmd/tslist@latest
```
tslist.InterfaceVisitor:
```
$ go get github.com/kimuson13/tslist
```
## Situation to use
When you want to check what the type sets actually fulfill.
For exampel, Interger in `golang.org/x/exp/constraints`
```
type Integer interface {
    Signed | Unsigned
}
```
If you want to understand what `Integer` actually fulfill, you need to check both `Signed` and `Unsigned` too.  
However use `tslist`, all types in one command.
## How to use
### tslist
You should input package you want to check the type sets actually fulfill like that.
```
$ tslist golang.org/x/exp/constrains
```
### tslist.InterfaceVisitor
You should input *ast.InterfaceType to tslist.InterfaceVisitor, so you got type sets and method list.
This API help to make it easy to perform static analysis of `*ast.InterfaceType`
The return value type is `VisitorResult`. This struct has 4 fields below.
| field name | type | about |
| ---- | ---- | ---- | 
|  Pos  |  token.Pos(int)  | interface declaration position |
|  Name  |  string  | interface name(e.g. Stringer) |
| TypeSets | []TypeValue | type sets |
| Methods | []Method | method list |

TypeValue:
| field name | type | about |
| ---- | ---- | ---- |
| Name | string | identifier name. not type name |
| Type | types.Type | type's info |

Method:
| field name | type | about |
| ---- | ---- | ---- |
| Name | string | method's name |
| Args | []TypeValue | method's arguments |
| Outputs | []TypeValue | method's return values |

**In golang.org/x/exp/typeparams implemented same APIs.**  
So, after Go future updates, typeparms will be in stadard Go statistic check package such as types.When that actually happens this API(tslist.InterfaceVisitor) will become duplicated
## Demo(tslist)
### golang.org/x/exp/constraints
```
$ tslist golang.org/x/exp/constraints
/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:12:13
Signed
type set: [~int64 ~int ~int8 ~int16 ~int32]
method list:

/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:19:15: 
Unsigned
type set: [~uintptr ~uint ~uint8 ~uint16 ~uint32 ~uint64]
method list:

/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:26:14: 
Integer
type set: [~uint32 ~uint64 ~int ~int16 ~int32 ~uint ~uint8 ~int8 ~int64 ~uint16 ~uintptr]
method list:

/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:33:12: 
Float
type set: [~float32 ~float64]
method list:

/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:40:14: 
Complex
type set: [~complex64 ~complex128]
method list:

/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:48:14: 
Ordered
type set: [~int32 ~uintptr ~string ~uint8 ~int64 ~float64 ~uint64 ~uint32 ~int8 ~int16 ~uint16 ~int ~uint ~float32]
method list:

/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:12:13: 
Signed
type set: [~int ~int8 ~int16 ~int32 ~int64]
method list:

/[your file path]/golang.org/x/exp@[some versions]constraints/constraints.go:19:15: 
Unsigned
type set: [~uint ~uint8 ~uint16 ~uint32 ~uint64 ~uintptr]
method list:
```
### To fmt package
```
$ tslist fmt
/usr/local/go/src/fmt/print.go:38:12: 
State
type set: [any]
method list:
Write(b []byte) (n int, err error) 
Width() (wid int, ok bool) 
Precision() (prec int, ok bool) 
Flag(c int) bool

/usr/local/go/src/fmt/print.go:53:16: 
Formatter
type set: [any]
method list:
Format(f fmt.State, verb rune) 

/usr/local/go/src/fmt/print.go:62:15: 
Stringer
type set: [any]
method list:
String() string

/usr/local/go/src/fmt/print.go:70:17: 
GoStringer
type set: [any]
method list:
GoString() string

/usr/local/go/src/fmt/scan.go:21:16: 
ScanState
type set: [any]
method list:
ReadRune() (r rune, size int, err error) 
UnreadRune() error
SkipSpace() 
Token(skipSpace bool, f func(rune) bool) (token []byte, err error) 
Width() (wid int, ok bool) 
Read(buf []byte) (n int, err error) 

/usr/local/go/src/fmt/scan.go:55:14: 
Scanner
type set: [any]
method list:
Scan(state fmt.ScanState, verb rune) error
```