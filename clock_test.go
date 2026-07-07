// Copyright (c) the go-ruby-timecop/timecop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package timecop

import (
	"testing"
	"time"
)

// fake is a deterministic real-clock seam: tests advance it explicitly so no
// wall-clock reading ever leaks into the assertions.
type fake struct{ t time.Time }

func (f *fake) now() time.Time          { return f.t }
func (f *fake) advance(d time.Duration) { f.t = f.t.Add(d) }
func newFake(t time.Time) *fake         { return &fake{t: t} }

var (
	ref   = time.Date(2008, 10, 5, 12, 0, 0, 0, time.UTC) // real "now"
	other = time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)   // a target instant
)

func mustEqual(t *testing.T, got, want time.Time) {
	t.Helper()
	if !got.Equal(want) {
		t.Fatalf("time = %v, want %v", got, want)
	}
}

func TestModeString(t *testing.T) {
	cases := map[Mode]string{
		ModeFreeze: "freeze",
		ModeTravel: "travel",
		ModeScale:  "scale",
		Mode(99):   "unknown",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Fatalf("Mode(%d).String() = %q, want %q", m, got, want)
		}
	}
}

func TestNewUsesRealClockSeam(t *testing.T) {
	// New() with no seam falls back to time.Now; Current on an empty stack must
	// return (approximately) the wall clock — exercising the nil-seam branch.
	c := New()
	before := time.Now()
	got := c.Current()
	after := time.Now()
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Fatalf("Current() = %v, not near wall clock [%v, %v]", got, before, after)
	}
	if c.Mocked() {
		t.Fatal("fresh clock reports Mocked() true")
	}
}

func TestFreezeIsFixed(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	got := c.Freeze(other)
	mustEqual(t, got, other)
	if !c.Frozen() || c.Travelled() || c.Scaled() {
		t.Fatal("expected Frozen() only")
	}
	// Real time advances; a frozen clock does not move.
	f.advance(time.Hour)
	mustEqual(t, c.Current(), other)
	if !c.Mocked() || c.Depth() != 1 {
		t.Fatalf("Mocked/Depth wrong: %v %d", c.Mocked(), c.Depth())
	}
}

func TestTravelAdvancesAtRealRate(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	got := c.Travel(other)
	mustEqual(t, got, other) // at push, current == target
	if !c.Travelled() {
		t.Fatal("expected Travelled()")
	}
	f.advance(90 * time.Minute)
	mustEqual(t, c.Current(), other.Add(90*time.Minute))
}

func TestScaleAdvancesAtFactor(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	got := c.Scale(4, other)
	mustEqual(t, got, other)
	if !c.Scaled() {
		t.Fatal("expected Scaled()")
	}
	if fac, ok := c.ScalingFactor(); !ok || fac != 4 {
		t.Fatalf("ScalingFactor() = %v %v, want 4 true", fac, ok)
	}
	f.advance(time.Second)
	mustEqual(t, c.Current(), other.Add(4*time.Second))
}

func TestNesting(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	c.Travel(other) // bottom frame
	f.advance(time.Minute)
	c.Freeze(other.Add(time.Hour)) // top frame frozen
	if c.Depth() != 2 {
		t.Fatalf("Depth = %d, want 2", c.Depth())
	}
	f.advance(time.Hour)
	mustEqual(t, c.Current(), other.Add(time.Hour)) // top freeze dominates
}

func TestWithFrozenPopsAfter(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	c.WithFrozen(other, func() {
		mustEqual(t, c.Current(), other)
		if c.Depth() != 1 {
			t.Fatalf("in-block Depth = %d, want 1", c.Depth())
		}
	})
	if c.Mocked() {
		t.Fatal("frame not popped after WithFrozen")
	}
	mustEqual(t, c.Current(), ref)
}

func TestWithTravelAndScaleBlocks(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	c.WithTravel(other, func() {
		if !c.Travelled() {
			t.Fatal("expected Travelled in block")
		}
	})
	c.WithScale(2, other, func() {
		if !c.Scaled() {
			t.Fatal("expected Scaled in block")
		}
	})
	if c.Mocked() {
		t.Fatal("frames not popped")
	}
}

func TestBlockRestoresOnPanic(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)
	c.Travel(other) // an outer frame that must survive the panic

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic to propagate")
			}
		}()
		c.WithFrozen(other.Add(time.Hour), func() {
			if c.Depth() != 2 {
				t.Fatalf("in-block Depth = %d, want 2", c.Depth())
			}
			panic("boom")
		})
	}()

	if c.Depth() != 1 {
		t.Fatalf("after panic Depth = %d, want 1 (outer frame kept)", c.Depth())
	}
	if !c.Travelled() {
		t.Fatal("outer travel frame lost after panic")
	}
}

func TestReturn(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)
	c.Freeze(other)
	c.SetBaseline(other)
	c.Return()
	if c.Mocked() {
		t.Fatal("Return did not clear stack")
	}
	if _, ok := c.Baseline(); ok {
		t.Fatal("Return did not clear baseline")
	}
	mustEqual(t, c.Current(), ref)
}

func TestWithReturnRevertsThenRestores(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)
	c.Freeze(other)

	c.WithReturn(func() {
		mustEqual(t, c.Current(), ref) // real time inside the block
		if c.Mocked() {
			t.Fatal("clock still mocked inside WithReturn")
		}
	})
	// Restored afterwards.
	mustEqual(t, c.Current(), other)
	if !c.Frozen() {
		t.Fatal("frozen frame not restored after WithReturn")
	}
}

func TestWithReturnRestoresOnPanic(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)
	c.SetBaseline(other)
	c.Freeze(other.Add(time.Hour))

	func() {
		defer func() { recover() }()
		c.WithReturn(func() { panic("boom") })
	}()

	if c.Depth() != 2 {
		t.Fatalf("Depth after panic = %d, want 2", c.Depth())
	}
	if _, ok := c.Baseline(); !ok {
		t.Fatal("baseline not restored after panicking WithReturn")
	}
}

func TestBaselineAndReturnToBaseline(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	c.SetBaseline(other)
	if b, ok := c.Baseline(); !ok || !b.Equal(other) {
		t.Fatalf("Baseline() = %v %v", b, ok)
	}
	if !c.Travelled() || c.Depth() != 1 {
		t.Fatal("SetBaseline should push a single travel frame")
	}
	// Baseline travels: advancing real time advances the clock.
	f.advance(time.Minute)
	mustEqual(t, c.Current(), other.Add(time.Minute))

	// Nest a freeze on top, then unwind back to baseline.
	c.Freeze(ref)
	if c.Depth() != 2 {
		t.Fatalf("Depth = %d, want 2", c.Depth())
	}
	got := c.ReturnToBaseline()
	if c.Depth() != 1 || !c.Travelled() {
		t.Fatalf("ReturnToBaseline should leave only the baseline travel frame; Depth=%d", c.Depth())
	}
	mustEqual(t, got, other.Add(time.Minute))
}

func TestReturnToBaselineWithoutBaseline(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)
	c.Freeze(other)

	got := c.ReturnToBaseline() // no baseline => full revert
	if c.Mocked() {
		t.Fatal("ReturnToBaseline without baseline should revert to real time")
	}
	mustEqual(t, got, ref)
}

func TestScalingFactorNonScaleTop(t *testing.T) {
	f := newFake(ref)
	c := NewWith(f.now)

	if _, ok := c.ScalingFactor(); ok {
		t.Fatal("empty stack should report no scaling factor")
	}
	c.Freeze(other)
	if _, ok := c.ScalingFactor(); ok {
		t.Fatal("frozen top should report no scaling factor")
	}
}

func TestPredicatesOnEmptyStack(t *testing.T) {
	c := NewWith(newFake(ref).now)
	if c.Frozen() || c.Travelled() || c.Scaled() {
		t.Fatal("empty stack must report all predicates false")
	}
}
