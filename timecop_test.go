// Copyright (c) the go-ruby-timecop/timecop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package timecop

import (
	"testing"
	"time"
)

// resetStd returns the default clock to a pristine, deterministic state and
// registers cleanup so the shared singleton never leaks across tests.
func resetStd(t *testing.T, f *fake) {
	t.Helper()
	std.Return()
	SetNow(f.now)
	t.Cleanup(func() {
		std.Return()
		SetNow(nil)
	})
}

func TestDefaultClockSurface(t *testing.T) {
	f := newFake(ref)
	resetStd(t, f)

	if Default() != std {
		t.Fatal("Default() must return the package singleton")
	}
	if Mocked() {
		t.Fatal("fresh default clock is mocked")
	}

	mustEqual(t, Freeze(other), other)
	if !Frozen() {
		t.Fatal("expected Frozen()")
	}
	mustEqual(t, Now(), other)
	Return()

	mustEqual(t, Travel(other), other)
	if !Travelled() {
		t.Fatal("expected Travelled()")
	}
	f.advance(time.Minute)
	mustEqual(t, Now(), other.Add(time.Minute))
	Return()

	f.t = ref
	mustEqual(t, Scale(3, other), other)
	if !Scaled() {
		t.Fatal("expected Scaled()")
	}
	f.advance(time.Second)
	mustEqual(t, Now(), other.Add(3*time.Second))
	Return()
}

func TestDefaultClockBlocksAndBaseline(t *testing.T) {
	f := newFake(ref)
	resetStd(t, f)

	WithFrozen(other, func() {
		mustEqual(t, Now(), other)
	})
	WithTravel(other, func() {
		if !Travelled() {
			t.Fatal("expected Travelled in block")
		}
	})
	WithScale(2, other, func() {
		if !Scaled() {
			t.Fatal("expected Scaled in block")
		}
	})
	if Mocked() {
		t.Fatal("blocks did not pop")
	}

	Freeze(other)
	WithReturn(func() {
		mustEqual(t, Now(), ref)
	})
	mustEqual(t, Now(), other)
	Return()

	SetBaseline(other)
	if b, ok := Baseline(); !ok || !b.Equal(other) {
		t.Fatalf("Baseline() = %v %v", b, ok)
	}
	Freeze(ref)
	mustEqual(t, ReturnToBaseline(), other)
}
