OS_NAME := $(shell uname)
ifeq ($(OS_NAME), Darwin)
OPEN := open
else
OPEN := xdg-open
endif

clean:
	@rm -rf build/

build: clean
	@GOOS=windows GOARCH=amd64 go build -o ./build/typecheckall.exe ./main.go
	@GOOS=linux GOARCH=amd64 go build -o ./build/typecheckall ./main.go

.PHONY: build \
		clean
