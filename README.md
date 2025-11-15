# Projektarbeit: Go-Generics und der Typechecker (wip)

## Grundlagen und Einschränkungen von Go-Generics

### Einführung
- Seit Go 1.18 unterstützt Go Generics, die es ermöglichen, wiederverwendbare und typsichere Datenstrukturen und Algorithmen zu erstellen.
- Go-Generics werden mit Typparametern definiert und können in Funktionen, Structs, Interfaces, Typdefinitionen und Methoden verwendet werden.
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
- Zur Compile-Zeit verwendet Go die Monomorphisierung, um typspezifische Versionen generischer Funktionen/Typen für jede eindeutig verwendete Kombination von Typargumenten zu erzeugen. Dadurch entsteht ke, da für jede Kombination von Typargumenten eine eigene Version der Funktion/des Typs erstellt wird.

### Einschränkungen von Go-Generics
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

## Der Go Typechecker

### Einführung
- Der Go Typechecker ist ein wesentlicher Bestandteil des Go-Compilers, der sicherstellt, dass der Code den Typregeln der Sprache entspricht.
- Go stellt im package `go/types` genau diese Funktionalität bereit, die es ermöglicht, Go-Code zu analysieren und zu überprüfen.
- Kernaufgaben des Typecheckers:
	- **Identifier Resolution**: Verknüpft Bezeichner (Variablen-, Funktions- und Typnamen) mit ihren Deklarationen. Zum Beispiel: `fmt.Println` -> welches `Println`?
	- **Type Deduction**: Bestimmt die Typen von Ausdrücken basierend auf ihren Operanden und Kontext und stellt sicher, dass die Typen kompatibel sind. Z.B.: `a + b` -> welcher Typ?
	- **Constant Evaluation**: Berechnet die Werte von Konstanten zur Compile-Zeit.
	> Achtung: Diese 3 Aufgaben sind stark miteinander verknüpft und müssen daher zusammen durchgeführt werden. Zum Beispiel kann der Typ eines Ausdrucks von einer Konstanten abhängen.
- Zentrale Datenstrukturen des Typecheckers:
	- **Objekte (`types.Object`)**: Repräsentieren deklarierte Entitäten wie Variablen, Funktionen, Typen und Pakete.
	- **Typen (`types.Type`)**: Repräsentieren die verschiedenen Typen in Go, einschließlich primitiver Typen, zusammengesetzter Typen (Structs, Slices, Maps) und generischer Typen mit Typparametern.
	- **Scopes (`types.Scope`)**: Repräsentieren ein mapping von Bezeichnern zu Objekten in einem bestimmten Gültigkeitsbereich (z.B. Paket-, Funktions- oder Blockebene).

### Bausteine des Typecheckers
Jedes deklarierte Element in Go wird im Typechecker durch ein `types.Object` repräsentiert. Dieses wird verwendet, um Informationen über das deklarierte Element zu speichern und darauf zuzugreifen. Beispielsweise kann dadurch im Fall von Fehler- oder Code-Analyse-Tools auf Metadaten und präzise Positionen von Deklarationen im Quellcode zugegriffen werden. Das `types.Object`-Interface setzt u.A. folgende Methoden voraus (Auswahl):
- `Name() string`: Gibt den Namen des Objekts zurück (z.b. den Variablennamen).
- `Exported() bool`: Gibt zurück, ob das Objekt exportiert ist (d.h. ob es mit einem Großbuchstaben beginnt).
- `Type() Type`: Gibt den Typ des Objekts zurück (z.B. den Typ einer Variable oder die signatur einer Funktion).
- `Pos() token.Pos`: Gibt die Position der Deklaration des Objekts im Quellcode zurück.

- `Parent() *Scope`: Gibt den Gültigkeitsbereich zurück, in dem das Objekt deklariert ist (z.B. Paket- oder Funktionsscope).
- `Pkg() *Package`: Gibt das Paket zurück, zu dem das Objekt gehört. `nil` für Objekte im `universe`-Scope (vordefinierte Typen und Funktionen).
- `Id() string`: Gibt eine eindeutige Kennung für das Objekt zurück. Zwei IDs sind genau dann verschieden, wenn dieses unterschiedliche Namen habend, oder in unterschiedlichen Paketen deklariert und nicht exportiert sind (_[Uniqueness of identifiers](https://go.dev/ref/spec#Uniqueness_of_identifiers)_). Für _nicht_ exportierte Objekte wird daher die Paketkennung in die ID einbezogen, um Kollisionen zu vermeiden.

#### Wichtige `types.Object`-Implementierungen
- `*types.Var`: Repräsentiert eine Variable (lokal, global oder Feld in einem Struct).
- `*types.Func`: Repräsentiert eine Funktion oder Methode.
- `*types.TypeName`: Repräsentiert einen benutzerdefinierten Typ (z.B. `type MyInt int`, `type MyStruct struct {...}`).
- `*types.Const`: Repräsentiert eine Konstante (z.B. `const Pi = 3.14`).
- `*types.PkgName`: Repräsentiert den Import eines Pakets in einem anderen Paket (z.B. das `fmt` in `import "fmt"`).

Objekte sind kanonisch, d.h. es gibt genau ein `types.Object` für jede deklarierte Entität im Quellcode. Dies ermöglicht eine konsistente und effiziente Verwaltung von Typinformationen während der Typechecking-Phase.


### Organisation von Objekten
Der Go Typechecker verwendet Scopes (`types.Scope`), um Objekte zu organisieren und den Gültigkeitsbereich von Bezeichnern zu verwalten. Jeder Scope wird durch einen lexikalischen Block im Quellcode beschrieben (z.B. ein Paket, eine Funktion oder ein Codeblock), in dem Bezeichner deklariert und verwendet werden können. 

#### Scope Hierarchie
Scopes sind hierarchisch organisiert, wobei jeder Scope einen übergeordneten Scope hat. 

1. **`universe`-Scope (root)**: Der globale `types.Universe`-Scope enthält vordefinierte Typen und Funktionen (z.B. `int`, `string`, `println`). Dieser sollte niemals verändert werden.
2. **`package`-Scope**: Jeder Paket-Scope enthält alle Objekte, die in einem bestimmten Paket deklariert sind. Jedes Paket hat seinen eigenen Scope, und hat den `universe`-Scope als übergeordnet.
3. **Datei-Scope**: Jede Quellcodedatei (`*ast.File`) hat ihren eigenen Scope, der den enstprechenden Paket-Scope als übergeordneten Scope hat.
4. **Blocklevel-Scopes**: Jede Kontrollanweisung oder Funktion hat ihren eigenen Scope, der den übergeordneten Scope (z.B. Datei-Scope ) als übergeordneten Scope hat. Geschachtelte Blöcke (z.B. Schleifen, `if`-Anweisungen) haben ebenfalls eigene Scopes, die den Scope der umgebenden Funktion oder des Blocks untergeordnet sind.

#### Namensauflösung
Um ein Objekt anhand seines Namens zu finden, stellt das `types.Scope`-Struct zwei zentrale Methoden bereit:
- `Lookup(name string) *Object`: Sucht im aktuellen Scope nach einem Objekt mit dem angegebenen Namen. Wenn das Objekt nicht gefunden wird, wird `nil` zurückgegeben.
- `LookupParent(name string, pos token.Pos) (*Scope, Object)`: Sucht rekursiv in dem aktuellen Scope und den übergeordnet Scopes nach einem Objekt mit dem angegebenen Namen. Der `pos`-Parameter verweist dabei auf die Position im Quellcode, an welcher nach dem Objekt gesucht werden soll. Das ist nötig, um sicherzustellen, dass ein Objekt nur gefunden wird, wenn es zum Zeitpunkt der Suche bereits deklariert wurde (Lexikalische Sichtbarkeit). Z.B. kann dadurch eine Variable im gleichen Scope nicht vor ihrer Deklaration gefunden werden.

### Typen
Jedes Objekt (`types.Object`) des Go Typecheckers hat einen zugehörigen Typ (`types.Type`), der den Datentyp der deklarierten Entität beschreibt. Der `types.Type`-Interface ist die zentrale Abstraktion für alle Typen in Go, einschließlich primitiver Typen (z.B. `int`, `string`), zusammengesetzter Typen (z.B. Structs, Slices, Maps) und generischer Typen mit Typparametern.
Das `types.Type`-Interface definiert nur wenige Methoden, da Typen sehr unterschiedlich sein können. Die primäre Methode ist:
- `Underlying() Type`: Gibt den zugrunde liegenden Typ zurück. Dies ist besonders nützlich für benutzerdefinierte Typen, um den Basisdatentyp zu ermitteln. Für primitive Typen gibt diese Methode den Typ selbst zurück. Zugrunde liegende Typen sind niemals benannte Typen oder Aliase.

#### Wichtige `types.Type`s
- `*types.Basic`: Repräsentiert primitive Typen wie `int`, `string`, `bool`.
- `*types.Struct`: Repräsentiert Struct-Typen mit Feldern.
- `*types.Interface`: Repräsentiert Interface-Typen mit Methoden.
- `*types.Signature`: Repräsentiert Funktions- und Methodensignaturen.
- `*types.Named`: Repräsentiert benannte Typen, die durch `type`-Deklarationen definiert sind.

> Achtung: Nach der Go-Spezifikation sind primitive Typen wie `int` und `string` ebenfalls benannte Typen, da sie durch `type`-Deklarationen definiert sind. Im Go Typechecker werden diese jedoch als `*types.Basic` repräsentiert, um ihre spezielle Rolle als primitive Typen zu verdeutlichen.

#### Komaptibilität von Typen
Um zu überprüfen, ob zwei Typen miteinander kompatibel sind, unterscheided Go zwischen drei Beziehungen von Typen. Für jede dieser Beziehungen stellt der `go/types`-Package entsprechende Funktionen bereit

##### Zuweisbarkeit
Zuweisbarkeit regelt, welche Paare von Typen in Zuweisungen (darunter zählen auch Funktionsaufrufe mit Parametern, Map-Zugriff, etc.) verwendet werden können. Für zwei Typen `T` und `V` ist `V` zuweisbar zu `T`, wenn eines der folgenden Kriterien erfüllt ist (Auswahl):
- `V` und `T` sind identisch.
- `V` und `T` haben den gleichen zugrunde liegenden Typ und mindestens einer von `T` oder `V` ist kein benannter Typ.

	> Achtung: Benannte Typen bezieht sich hier auf die durch Definition der Go-Spezifikation, nicht auf die vom Typecheker verwendeten `*types.Named`. Daher sind `int`, `string`, etc. auch benannte Typen.

	```go
	type MyInt int
	var a int
	var b MyInt

	a = b // Nicht erlaubt: auf beiden Seiten sind benannte Typen


	type MySlice []int
	var c []int
	var d MySlice

	c = d // Erlaubt: zugrunde liegender Typ ist gleich ([]int) und c ist kein benannter Typ
	```

- Weitere spezielle Regeln für bestimmte Typen (z.B. Schnittstellen, Funktionen, etc.).
Um zu überprüfen, ob zwei Typen zueinander zuweisbar sind, stellt das `go/types`-Package die Funktion `types.AssignableTo(V, T Type) bool` bereit.

##### Vergleichbarkeit
Regelt, ob ein Type mit `==` oder `!=` verglichen werden kann. Primitive Typen und Pointer sind beispielsweise immer vergleichbar, während Structs und Arrays nur unter bestimmten Bedingungen vergleichbar sind. Damit ein Struct vergleichbar ist, müssen alle seine Felder vergleichbar sein. Arrays sind vergleichbar, wenn ihr Elementtyp vergleichbar ist. Slices, Maps und Funktionen sind niemals vergleichbar.

Um zu überprüfen, ob ein Typ vergleichbar ist, stellt das `go/types`-Package die Funktion `types.Comparable(T Type) bool` bereit.

##### Umwandlungsfähigkeit
Regelt, ob ein Wert von einem Type in einen anderen Type umgewandelt werden kann. Umwandlungen können sowohl explizit (z.B. `T(v)`) als auch implizit (z.B. bei Funktionsaufrufen) erfolgen. Ein Wert `x` kann dann in einen Typ `T` umgewandelt werden, wenn eines der folgenden Kriterien erfüllt ist (Auswahl):
- `x` ist zuweisbar zu `T`.
- `x` ist ein `string` und `T` ist ein `[]byte` oder `[]rune` (undumgekehrt).
- `x` und `T` sind beide numerische Typen (z.B. `int`, `float64`, etc.).
- Weitere spezielle Regeln für bestimmte Typen (z.B. Typeparameter, Pointer, etc.).

Um zu überprüfen, ob ein Typ in einen anderen Typ umgewandelt werden kann, stellt das `go/types`-Package die Funktion `types.ConvertibleTo(V, T Type) bool` bereit.

### Verbindung von AST und Typechecker
Mit dem `types.Type` und `types.Object` fehlt dem Typechecker noch die Verbindung zum Quellcode. Diese Verbindung wird durch das `types.Info`-Struct hergestellt, das während des Typecheckings mit Informationen über die Typen und Objekte im Quellcode gefüllt wird. Das `types.Info`-Struct enthält mehrere Maps, die verschiedene Aspekte des Quellcodes abbilden. Die wichtigsten davon sind:
- `Types map[ast.Expr]types.TypeAndValue`: Verknüpft jeden AST-Ausdruck (`ast.Expr`) mit seinem Typ und Wert (nur für Kosntanten).
- `Defs map[*ast.Ident]types.Object`: Verknüpft jede Identifier-Deklaration (`*ast.Ident`) mit dem entsprechenden Objekt (`types.Object`).
- `Uses map[*ast.Ident]types.Object`: Verknüpft jede Identifier-Verwendung (`*ast.Ident`) mit dem entsprechenden Objekt (`types.Object`).
- `Scopes map[ast.Node]*types.Scope`: Verknüpft jeden AST-Knoten, der einen Gültigkeitsbereich definiert (z.B. Funktionen, Blöcke), mit dem entsprechenden Scope (`types.Scope`).

Mit diesen Feldern ist das `types.Info`-Struct die zentrale Verbindung zwischen dem abstrakten Syntaxbaum (AST) und den Typinformationen, die vom Typechecker generiert werden. Dadurch können Tools und Anwendungen detaillierte Analysen des Go-Codes durchführen, indem sie sowohl die Struktur des Codes als auch die zugehörigen Typinformationen berücksichtigen.

## Quellen und weiterführende Literatur
- [Tutorial: Getting started with generics](https://go.dev/doc/tutorial/generics)
- [`go/types`: The Go Type Checker](https://github.com/golang/example/tree/7f05d217867b2af52b0a28c6d1c91df97e1b5b39/gotypes)
- [Updating tools to support type parameters](https://github.com/golang/exp/tree/a4bb9ffd2546b4ac9773d60f1e9a6ff4ba82ad23/typeparams/example)
- [`go/types` package](https://cs.opensource.google/go/go/+/refs/tags/go1.25.4:src/go/types/;drc=34b70684ba2fc8c5cba900e9abdfb874c1bd8c0e)
- [The Go Programming Language Specification](https://go.dev/ref/spec)
- [Go Standard Library Documentation (`go/types`)](https://pkg.go.dev/go/types@go1.25.4)

## Angaben zur Nutzung von KI
Einige Inhalte dieses Dokuments wurden mit Unterstützung von KI-Technologien recherchiert und verfasst. Dabei kamen insbesondere Sprachmodelle wie Google Gemini und Copilot zum Einsatz. Diese Technologien halfen dabei, Informationen zu strukturieren, Codebeispiele zu generieren und komplexe Konzepte verständlich darzustellen. Trotz sorgfältiger Überprüfung durch den Autor können Fehler oder Ungenauigkeiten nicht vollständig ausgeschlossen werden. Der Autor übernimmt die volle Verantwortung für den Inhalt dieses Dokuments.
