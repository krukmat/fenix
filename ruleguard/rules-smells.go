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
}
