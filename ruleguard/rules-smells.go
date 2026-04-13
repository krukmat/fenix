package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

func smells(m dsl.Matcher) {
	// 1) Dos "guard if" seguidos con el mismo return => combinables con ||
	//    Ej:
	//      if a { return err }
	//      if b { return err }
	//    => if a || b { return err }
	m.Match(`if $c1 { return $ret }; if $c2 { return $ret }`).
		Report(`two consecutive guards return the same value; consider merging conditions with ||`).
		Suggest(`if $c1 || $c2 { return $ret }`)

	// Variante típica con continue (dentro de loops)
	m.Match(`if $c1 { continue }; if $c2 { continue }`).
		Report(`two consecutive continues; consider merging conditions with ||`).
		Suggest(`if $c1 || $c2 { continue }`)

	// 2) For anidados: no siempre es "malo", pero es un smell útil para refactor/extract
	m.Match(`for $*_ { for $*_ { $*_ } }`).
		Report(`nested for-loop; consider extracting inner loop logic or reducing algorithmic complexity`)

	// 3) time.Now() duplicado en argumentos de la misma llamada
	//    Dos evaluaciones de time.Now() en la misma call producen timestamps distintos.
	//    Capturamos dos posiciones variadic para cubrir cualquier combinación de args.
	//    Ej: f(a, time.Now(), b, time.Now()) => now := time.Now(); f(a, now, b, now)
	m.Match(`$f($*_, time.Now(), $*_, time.Now(), $*_)`).
		Report(`time.Now() called twice in the same argument list; capture it in a variable to guarantee a single consistent timestamp`)

	m.Match(`$f($*_, time.Now().UTC(), $*_, time.Now().UTC(), $*_)`).
		Report(`time.Now().UTC() called twice in the same argument list; capture it in a variable to guarantee a single consistent timestamp`)
}
