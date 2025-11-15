package main

type MyInt int

func isEven(n MyInt) bool {
	return n%2 == 0
}

type MyStruct struct {
	Field1 string
	Field2 int
}

func main() {
	x := MyInt(42)
	x = MyInt(43)
	s := MyStruct{Field1: "hello", Field2: 10}
	// inspect: MyStruct, 1, s, s.Field1
	_ = x
	_ = s
}
