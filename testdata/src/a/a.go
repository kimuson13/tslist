package a

type MyInt int

type MyMyInt MyInt

type i interface { // want "i\ntype set: empty\nmethod list:"
	MyInt
	MyMyInt
}

type i0 interface { // want "\ni0\ntype set: \\[MyInt MyMyInt\\]\nmethod list:\n"
	MyInt | MyMyInt
}

type i1 interface { // want "i1\ntype set: empty\nmethod list:"
	int
	float64
}

type i2 interface { // want "\ni2\ntype set: \\[int string\\]\nmethod list:"
	int | string
}

type i3 interface { // want "\ni3\ntype set: empty\nmethod list:\nf()\nf(val int, hoge string) string\nf3(val int) (string, bool)\nf4(val int)  (res int)\nf5(val \[\]string, ptr *int)\nf6(val string)  (res string)\n"
	f1()
	f2(val int, hoge string) string
	f3(val int) (string, bool)
	f4(val int) (res int)
	f5(val []string, ptr *int)
	f6(val string) (res string)
}

type i4 interface{} // want "[any]"

type i5 interface { // want "[~int]"
	~int
}

type i6 interface { // want "[any]"
	int | any
}

type i7 interface { // want "[int]"
	any
	int
}

type i8 interface { // want "int"
	int
	f1()
}

type i9 interface { // want "[int string]"
	i2
}

type i10 interface { // want "[int]"
	~int
	int
}

type i11 interface { // want "[int]"
	int | string
	f1() int
	int
}

type i12 interface { // want "[int]"
	any
	any
	any
	int
}

type i13 interface { // want "[int]"
	any | string
	string | bool | any | ~float64
	int
}

type s1 struct {
	val int
}

type i14 interface { // want "[\\[\\]string a.s1]"
	s1 | []string
}

type i15 interface { // want "[*int]"
	*int
}

type i16 interface { // want "[int MyInt]"
	int | MyInt
}

type i17 interface { // want "empty"
	int
	MyInt
}

type i18 interface {
	int | i12
}
