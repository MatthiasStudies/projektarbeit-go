Hier werden zwei Go-Dateien mit dem Go-Typechecker überprüft, um festzustellen ob der Typechecker Zuweisbarkeitsprüfungen cached oder nicht.

- In der `stress_multi.gotest` werden 5000 unterschiedliche Structs dem gleichen Interface zugewiesen. Der Typechecker muss also bei jeder Zuweisung überprüfen, ob das Struct alle Methode des Interfaces implementiert.
- In der `stress_single.gotest` wird nur ein einziges Struct dem Interface zugewiesen, aber 5000 Mal hintereinander. Der Typechecker muss also theoretisch nur einmal überprüfen, ob das Struct alle Methoden des Interfaces implementiert und sein Ergebnis zwischenspeichern (cachen).

Um den Typcheker zu timen, wird ein kleines Skript `time_it.go` verwendet. Dabei wird 500 Mal der Typechecker auf beiden Dateien ausgeführt und die durchschnittliche Laufzeit berechnet.
```sh
go run time_it.go stress_single.gotest stress_multi.gotest
```

Das Ergebnis:
```
File: .\stress_multi.gotest
Average time over 500 iterations: 62.67855719999996 ms
Minimum time: 51.304700000000004 ms
Maximum time: 107.08370000000001 ms

File: .\stress_single.gotest
Average time over 500 iterations: 60.855014200000014 ms
Minimum time: 48.899699999999996 ms
Maximum time: 118.6888 ms
```

Das Ergebnis zeigt, die Laufzeiten sind nahezu identisch. Dies deutet darauf hin, das der Go Typechecker die Zuweisbarkeitsprüfung nicht cached, sondern bei jeder Zuweisung erneut durchführt.

Dies geht auch aus dem `go/types` Sourcecode hervor, welcher keine Cachestruktur für Zuweisbarkeitsprüfungen enthält. Siehe [`go/types/assign.go`](https://github.com/golang/go/blob/cead111a772c2852c870fb140029d89152da4d14/src/go/types/assignments.go#L24).