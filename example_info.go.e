package main

var m = make(map[string]int)

func main() {
	v, ok := m["hello, "+"world"]
	print(rune(v), ok)
}
