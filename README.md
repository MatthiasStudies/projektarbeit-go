## Go-Generics

- Go-Generics werden mit Typparametern definiert und können in Funktionen, Structs, Interfaces und Methoden verwendet werden.
- Typparameter werden in eckigen Klammern `[]` nach dem Funktions- oder Typnamen angegeben.
- Generische Typen müssen mit Typbeschränkungen (Type Constraints) eingeschränkt werden.
	- **Einschränkung durch Interface**: Definiere ein Interface, das der Typparameter implementieren muss.
		- Beispiel:
		```go
		type Stringer interface {
			String() string
		}

		func PrintString[T Stringer](value T) {
			fmt.Println(value.String())
		}
		```

		> Die spezielle Einschränkung `comparable` erlaubt jeden Typ, der Vergleichsoperatoren (`==`, `!=`) unterstützt. Das ist nützlich, wenn der generische Typ als Schlüssel in Maps verwendet wird oder bei Gleichheitsprüfungen.
		- Beispiel:
			```go
			func AreEqual[T comparable](a, b T) bool {
				return a == b
			}
			```
	- **Einschränkung durch Typ**: Definiere einen (Basis-)Typ, auf dem der Typparameter basieren muss.
		- Beispiel:
		```go
		func PrintValue[T ~int](value T) {
			fmt.Println(value)
		}

		type MyInt int

		PrintValue(MyInt(42)) // Gültig, der zugrunde liegende Typ von MyInt ist int
		PrintValue(100)       // Gültig, int ist erlaubt
		```
		> Der Operator `~` erlaubt dem Typparameter, jeden Typ zu akzeptieren, dessen zugrunde liegender Typ dem angegebenen Basistyp entspricht.

	- **Einschränkung durch mehrere Typen (Type Sets)**: Definiere eine Menge von Typen, die ein Typparameter akzeptieren kann.
		- Beispiel: 
		```go
		func PrintType[T int | string](value T) {
			fmt.Println(value)
		}

		// oder

		func SumNumbers[T interface {int | float64}](a, b T) T {
			return a + b
		}

		PrintType(42)        // Gültig, int ist erlaubt
		PrintType("Hello")   // Gültig, string ist erlaubt
		```
		> Die experimentelle Go-Bibliothek `golang.org/x/exp/constraints` stellt einige vordefinierte Typmengen bereit, wie `constraints.Ordered` für Typen, die Ordnungsoperatoren (`<`, `>`, etc.) unterstützen, oder `constraints.Integer` für Ganzzahltypen.

- Go kann Typparameter beim Aufruf generischer Funktionen ableiten, daher müssen sie oft nicht explizit angegeben werden.
	- Beispiel:
	```go
	func PrintValue[T any](value T) {
		fmt.Println(value)
	}		
	PrintValue(42)          // Typparameter T wird als int abgeleitet
	PrintValue("Hello")     // Typparameter T wird als string abgeleitet
	```
- Zur Compile-Zeit verwendet Go die Monomorphisierung, um typspezifische Versionen generischer Funktionen/Typen für jede eindeutig verwendete Kombination von Typargumenten zu erzeugen. Dadurch entsteht kein Laufzeit-Overhead bei der Verwendung von Generics. Allerdings führt dies zu größeren Binärdateien, wenn viele unterschiedliche Typargumente verwendet werden.

## Einschränkungen von Go-Generics
- Methoden können keine eigenen Typparameter haben; nur der Typ, auf dem sie definiert sind (Receiver), kann Typparameter besitzen.
	```go
	func (t T) MethodName[U any](param U) { 
		// Das ist nicht erlaubt 
	}
	```
- Methoden können die Typparameter ihres Receivers nicht weiter einschränken.
	```go
	type Container[T any] struct {
		value T
	}

	func (c Container[T comparable]) IsEqual(other Container[T]) bool { 
		// Das ist nicht erlaubt 
		return c.value == other.value
	}
	```
- Die Einschränkung `comparable` kann nicht für benutzerdefinierte Typen implementiert werden.
- Eingeschränkte Type Assertions, selbst wenn `any` als Constraint verwendet wird.
	```go
	func ProcessValue[T any](value T) {
		str,ok := value.(string) // Das ist nicht erlaubt
	}

	// Workaround:
	func ProcessValue[T any](value T) {
		str, ok := any(value).(string) // Das ist erlaubt
	}
	```
- Auf Methoden oder Felder von Struct-Constraints kann nicht direkt über den Typparameter zugegriffen werden.
	```go
	type Box struct {
		value int
	}

	func (b Box) GetValue() int {
		return b.value
	}

	func ProcessBox[T Box](box T) {
		val := box.GetValue() // Das ist nicht erlaubt
		val := box.value // Das ist nicht erlaubt
	}

	// Workaround:
	func ProcessBox[T Box](box T) {
		val := Box(box).GetValue() // Das ist erlaubt
	}
	```
- Mehrere Interfaces können nicht als Typmengen verwendet werden.
	```go
	type Reader interface {
		Read(p []byte) (n int, err error)
	}

	type Writer interface {
		Write(p []byte) (n int, err error)
	}

	func ReadWrite[T Reader | Writer](rw T) { 
		// Das ist nicht erlaubt 
	}
	```

