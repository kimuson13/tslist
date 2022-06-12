package a

type MyInt int

type MyMyInt MyInt

type i interface { // want "no type"
	MyInt
	MyMyInt
}

type i0 interface { // want "[a.MyInt int]"
	MyInt | MyMyInt
}

type i1 interface { // want "no type"
	int
	float64
}

type i2 interface { // want "[int string]"
	int | string
}

type i3 interface { // want "no type"
	f1()
	f2(val int, hoge string) string
	f3(val int) (string, bool)
	f4(val int) (res int)
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
