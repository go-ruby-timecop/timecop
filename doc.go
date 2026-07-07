// Copyright (c) the go-ruby-timecop/timecop authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package timecop is a pure-Go (CGO-free) reimplementation of the deterministic
// core of Ruby's [timecop] gem — a controllable clock that lets code "freeze",
// "travel" through, and "scale" the passage of time, with block-scoped nesting
// and a return-to-baseline mechanism. It reproduces the gem's TimeStackItem
// mock-time formulas and its stack semantics exactly, without any Ruby runtime.
//
// It is the timecop engine for
// [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a
// standalone, reusable module: a host (rbgo) binds Ruby's Timecop module methods
// (Timecop.freeze/travel/scale/return/return_to_baseline plus their block forms)
// to a [Clock] and routes Time.now / Date.today / DateTime.now through
// [Clock.Current].
//
// # The real-clock seam
//
// timecop mocks time relative to the real system clock: a travelled or scaled
// clock keeps advancing as wall-clock time passes. To stay deterministic and
// testable, the "real now" is an injectable seam — the [Clock.Now] field. When
// it is nil the clock reads [time.Now]; a test (or a host) sets it to a
// controlled function so no reading of the wall clock ever leaks in. This
// mirrors the gem's Time.now_without_mock_time.
//
// # Mock-time formulas
//
// A [Clock] holds a stack of mock-time frames; [Clock.Current] reports the time
// according to the top frame (or the real clock when the stack is empty). Each
// frame captures the real "now" at the moment it was pushed (timeWas) and a
// target time, and reports:
//
//   - freeze: the fixed target — time does not move.
//   - travel: realNow + (target - timeWas) — offset that keeps advancing at 1×.
//   - scale:  target + (realNow - timeWas) * factor — advances at factor×.
//
// # Flow
//
//	c := timecop.New()
//	c.Freeze(t)                 // Time.now == t until Return
//	c.Travel(t)                 // jump to t, then keep ticking
//	c.Scale(4, t)               // from t, time runs 4× real
//	c.WithFrozen(t, func() {    // block scope: pushed, then popped
//		// ... time frozen at t here ...
//	})                          // popped even if fn panics
//	c.Return()                  // unwind everything, back to real time
//
// # Baseline
//
// [Clock.SetBaseline] records a baseline time (pushing a travel frame at it);
// [Clock.ReturnToBaseline] unwinds every nested frame back down to that
// baseline, mirroring Timecop.return_to_baseline.
//
// [timecop]: https://github.com/travisjeffery/timecop
package timecop
