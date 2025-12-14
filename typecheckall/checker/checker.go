package checker

type Checker interface {
	AddFile(filename string, content string) error
	Check() ([]Error, error)
}
