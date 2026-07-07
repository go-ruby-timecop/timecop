<p align="center"><img src="https://go-ruby-timecop.github.io/logo.png" alt="go-ruby-timecop/timecop" width="720"></p>

# timecop — go-ruby-timecop

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-timecop.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic core of Ruby's
[`timecop`](https://github.com/travisjeffery/timecop) gem** — a controllable
clock that lets code *freeze*, *travel* through, and *scale* the passage of time,
with block-scoped nesting and a return-to-baseline mechanism. It reproduces the
gem's `TimeStackItem` mock-time formulas and its stack semantics exactly —
**without any Ruby runtime**.

It is the timecop engine for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module: a host (rbgo) binds Ruby's `Timecop` module methods to a
`Clock` and routes `Time.now` / `Date.today` / `DateTime.now` through
`Clock.Current`.

> **The real-clock seam.** timecop mocks time *relative to the real system
> clock*: a travelled or scaled clock keeps advancing as wall-clock time passes.
> To stay deterministic and testable, the "real now" is an **injectable seam** —
> the `Clock.Now` field. When it is nil the clock reads `time.Now`; a test (or a
> host) sets it to a controlled function so no reading of the wall clock ever
> leaks in. This mirrors the gem's `Time.now_without_mock_time`.

## Mock-time formulas

A `Clock` holds a stack of mock-time frames; `Clock.Current` reports the time
according to the top frame (or the real clock when the stack is empty). Each
frame captures the real "now" at the moment it was pushed (`timeWas`) and a
target time, and reports — exactly as timecop's `TimeStackItem#time`:

| mode     | formula                                     | behaviour                       |
| -------- | ------------------------------------------- | ------------------------------- |
| `freeze` | `target`                                    | time is pinned; it does not move |
| `travel` | `realNow + (target − timeWas)`              | jump, then advance at 1× real    |
| `scale`  | `target + (realNow − timeWas) × factor`     | jump, then advance at *factor*×  |

## Features

Faithful port of the `timecop` gem's engine:

- **Clock** — `New()` / `NewWith(now)`; `Current()` reports the mocked now.
- **freeze / travel / scale** — `Freeze(t)`, `Travel(t)`, `Scale(factor, t)`
  push a frame and return the resulting mocked time.
- **Block forms** — `WithFrozen(t, fn)`, `WithTravel(t, fn)`,
  `WithScale(factor, t, fn)` push a frame for the duration of `fn` and pop it
  afterwards — **even if `fn` panics** (timecop's `ensure`-scoped block).
- **Nesting** — an arbitrarily deep stack of frames; the top frame wins, mirroring
  `Timecop.freeze { Timecop.travel { … } }`.
- **return** — `Return()` unwinds all frames and clears the baseline;
  `WithReturn(fn)` reverts to real time for `fn` then restores (panic-safe).
- **baseline** — `SetBaseline(t)` records a baseline (pushing a travel frame at
  it); `ReturnToBaseline()` unwinds every nested frame back down to the baseline;
  `Baseline()` reads it.
- **predicates** — `Frozen()`, `Travelled()`, `Scaled()`, `Mocked()`, `Depth()`,
  `ScalingFactor()`, mirroring `Timecop.frozen?/travelled?/scaled?`.
- **Package surface** — package-level `Freeze/Travel/Scale/Return/…` operate on a
  process-wide default `Clock` (the analogue of timecop's singleton);
  `SetNow(func)` injects the seam for deterministic tests.

CGO-free, dependency-free (stdlib only), **100% test coverage**, `gofmt` +
`go vet` clean, and green across the six 64-bit Go targets (amd64, arm64,
riscv64, loong64, ppc64le, **s390x** — big-endian) plus `js/wasm` and
`wasip1/wasm`.

## Install

```sh
go get github.com/go-ruby-timecop/timecop
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/go-ruby-timecop/timecop"
)

func main() {
	c := timecop.New()

	t := time.Date(2008, 10, 5, 0, 0, 0, 0, time.UTC)

	c.Freeze(t)                     // Time.now == t, pinned
	fmt.Println(c.Current())        // 2008-10-05 00:00:00 +0000 UTC
	c.Return()

	c.Travel(t)                     // jump to t, then keep ticking at 1×
	c.Scale(4, t)                   // ... nested: now runs 4× real time

	c.WithFrozen(t, func() {        // block scope: pushed, then popped
		// ... time frozen at t inside here ...
	})                              // popped even if fn panics

	c.Return()                      // unwind everything, back to real time
}
```

### Injecting the real-clock seam (tests / hosts)

```go
now := time.Date(2008, 10, 5, 12, 0, 0, 0, time.UTC)
c := timecop.NewWith(func() time.Time { return now })

c.Travel(time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC))
now = now.Add(90 * time.Minute)          // advance the *real* clock
fmt.Println(c.Current())                 // 1999-01-01 01:30:00 (travelled + advanced)
```

## Value model

| gem                                   | this package                             |
| ------------------------------------- | ---------------------------------------- |
| `Timecop.freeze(t)`                   | `Clock.Freeze(t)` / `timecop.Freeze(t)`  |
| `Timecop.travel(t)`                   | `Clock.Travel(t)` / `timecop.Travel(t)`  |
| `Timecop.scale(factor, t)`            | `Clock.Scale(factor, t)`                 |
| `Timecop.freeze(t) { … }`             | `Clock.WithFrozen(t, fn)`                |
| `Timecop.travel(t) { … }`             | `Clock.WithTravel(t, fn)`                |
| `Timecop.scale(factor, t) { … }`      | `Clock.WithScale(factor, t, fn)`         |
| `Timecop.return`                      | `Clock.Return()`                         |
| `Timecop.return { … }`                | `Clock.WithReturn(fn)`                   |
| `Timecop.baseline = t`                | `Clock.SetBaseline(t)`                   |
| `Timecop.return_to_baseline`          | `Clock.ReturnToBaseline()`               |
| mocked `Time.now`                     | `Clock.Current()` / `timecop.Now()`      |
| `Timecop.frozen?/travelled?/scaled?`  | `Clock.Frozen()/Travelled()/Scaled()`    |
| `Time.now_without_mock_time`          | `Clock.Now` (injectable real-clock seam) |

## Tests & coverage

The suite is fully deterministic: every test injects a fake real-clock seam and
advances it explicitly, so no wall-clock reading leaks into an assertion. Every
branch — nested frames, block pop-on-panic, scale math, baseline unwinding — is
covered, holding coverage at **100%** across every OS and arch lane.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-timecop/timecop authors.
