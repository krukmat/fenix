package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// Production-grade performance and safety rules.
// These block patterns that are acceptable in scripts but not in production services.

func performance(m dsl.Matcher) {
	// 1) panic in production code — use error returns instead.
	//    panic unwinds the stack and kills the goroutine; in an HTTP server this
	//    crashes the request (or the process if not recovered). Always return error.
	m.Match(`panic($x)`).
		Report(`panic in production code; return an error instead of panicking`)

	// 2) Silent error drop — both return values discarded.
	//    Silently dropping errors hides failures and makes debugging impossible
	//    in production. Handle or explicitly log every error.
	m.Match(`_, _ = $f($*_)`).
		Report(`silent error drop: both return values discarded from $f; handle the error explicitly`)

	// 3) fmt.Sprintf for simple two-value string concat — allocates a tmp buffer.
	//    Use + operator or strings.Builder for hot-path string construction.
	m.Match(`fmt.Sprintf("%s%s", $a, $b)`).
		Report(`fmt.Sprintf for simple concat; prefer $a + $b or strings.Builder to avoid unnecessary allocation`).
		Suggest(`$a + $b`)

	m.Match(`fmt.Sprintf("%s%s%s", $a, $b, $c)`).
		Report(`fmt.Sprintf for simple concat; prefer strings.Builder to avoid unnecessary allocation`)

	// 4) strings.Replace without limit when ReplaceAll is intended.
	//    strings.Replace(s, old, new, -1) allocates the same as ReplaceAll but
	//    is less expressive; use strings.ReplaceAll to signal intent.
	m.Match(`strings.Replace($s, $old, $new, -1)`).
		Report(`use strings.ReplaceAll($s, $old, $new) instead of strings.Replace with -1`).
		Suggest(`strings.ReplaceAll($s, $old, $new)`)

	// 5) bytes.Equal([]byte(s1), []byte(s2)) — double allocation converting strings.
	//    Use strings package or compare directly.
	m.Match(`bytes.Equal([]byte($s1), []byte($s2))`).
		Report(`bytes.Equal on converted strings allocates twice; use $s1 == $s2 or strings.EqualFold for case-insensitive comparison`)

	// 6) append in a loop assigning back to same slice — correct but call it out
	//    if the slice grows unboundedly without a pre-allocated cap.
	//    Detects: for ... { result = append(result, ...) } where result was declared
	//    without make(..., 0, n). This is a warn-level smell, not a hard block.
	m.Match(`for $*_ { $result = append($result, $*_) }`).
		Report(`append inside loop without pre-allocated capacity; consider make([]T, 0, estimatedLen) before the loop to avoid repeated reallocations`)

	// 7) http.Get / http.Post with no timeout — uses DefaultClient which has no timeout.
	//    In production every outbound HTTP call must use a client with explicit timeout.
	m.Match(`http.Get($url)`).
		Report(`http.Get uses DefaultClient with no timeout; use an *http.Client with Timeout set for production calls`)

	m.Match(`http.Post($url, $ct, $body)`).
		Report(`http.Post uses DefaultClient with no timeout; use an *http.Client with Timeout set for production calls`)

	// 8) log.Fatal / log.Panic in library/domain code — terminates the process.
	//    Only main() and test harnesses may call log.Fatal.
	m.Match(`log.Fatal($*_)`).
		Report(`log.Fatal calls os.Exit and cannot be deferred or recovered; return an error instead`)

	m.Match(`log.Fatalf($*_)`).
		Report(`log.Fatalf calls os.Exit and cannot be deferred or recovered; return an error instead`)

	m.Match(`log.Panic($*_)`).
		Report(`log.Panic panics after logging; return an error instead`)

	m.Match(`log.Panicf($*_)`).
		Report(`log.Panicf panics after logging; return an error instead`)

	// 9) time.Sleep in non-test code — usually hides a missing synchronisation primitive.
	//    Use context.WithTimeout, ticker, or channel signalling instead.
	m.Match(`time.Sleep($d)`).
		Report(`time.Sleep in production code; use context.WithTimeout, time.After, or a sync primitive instead`)

	// 10) Returning a nil error as a named variable without assigning it — footgun.
	//     func foo() (err error) { return } when err was never set returns nil silently.
	//     This catches the most common form: explicit `return err` where err is zero.
	m.Match(`var $err error; return $err`).
		Report(`returning zero-value error variable; make the success path explicit with return nil`)
}
