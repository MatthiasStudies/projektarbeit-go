package main

import "testmodule/submodule"

func main() {
	submodule.Method()
	Method()
}

func M() {
	var s string
	s = 123
}
