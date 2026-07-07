// Copyright (c) the go-ruby-timecop/timecop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package timecop

import "time"

// Mode is the kind of mock-time frame on a [Clock]'s stack — the Go analogue of
// timecop's TimeStackItem mock_type (:freeze, :travel, :scale).
type Mode int

const (
	// ModeFreeze pins time at a fixed instant; it does not advance.
	ModeFreeze Mode = iota
	// ModeTravel jumps time to an instant, after which it keeps advancing at
	// real (1×) rate.
	ModeTravel
	// ModeScale jumps time to an instant, after which it advances at a
	// configurable factor of the real rate.
	ModeScale
)

// String returns the timecop mock_type name of the mode: "freeze", "travel", or
// "scale".
func (m Mode) String() string {
	switch m {
	case ModeFreeze:
		return "freeze"
	case ModeTravel:
		return "travel"
	case ModeScale:
		return "scale"
	default:
		return "unknown"
	}
}

// frame is one mock-time entry on the stack, mirroring timecop's TimeStackItem.
// It captures the real "now" at the moment it was pushed (timeWas) together with
// the target time and, for ModeScale, the scaling factor.
type frame struct {
	mode    Mode
	target  time.Time // @time: the requested instant
	timeWas time.Time // Time.now_without_mock_time at push
	scale   float64   // scaling factor (ModeScale only)
}

// at reports the mock time for this frame given the current real now, applying
// timecop's TimeStackItem#time formula for the frame's mode.
func (f frame) at(realNow time.Time) time.Time {
	switch f.mode {
	case ModeScale:
		// target + (realNow - timeWas) * factor
		elapsed := realNow.Sub(f.timeWas)
		return f.target.Add(time.Duration(float64(elapsed) * f.scale))
	case ModeTravel:
		// realNow + (target - timeWas) — an offset that keeps advancing.
		return realNow.Add(f.target.Sub(f.timeWas))
	default: // ModeFreeze
		return f.target
	}
}

// Clock is a controllable clock: the pure-Go engine behind Ruby's Timecop
// module. It maintains a stack of mock-time frames (freeze/travel/scale) plus an
// optional baseline, and reports the current mock time through [Clock.Current].
//
// A Clock is not safe for concurrent use; guard it externally if shared. This
// matches the gem, whose per-thread stacks the host manages.
type Clock struct {
	// Now is the real-clock seam. When nil the clock reads [time.Now]; set it
	// to a deterministic function in tests or to a host-provided clock. It is
	// the analogue of timecop's Time.now_without_mock_time.
	Now func() time.Time

	stack       []frame
	baseline    time.Time
	hasBaseline bool
}

// New returns a Clock backed by the real system clock ([time.Now]).
func New() *Clock { return &Clock{} }

// NewWith returns a Clock whose real-clock seam is now. A nil now falls back to
// [time.Now] at read time.
func NewWith(now func() time.Time) *Clock { return &Clock{Now: now} }

// realNow reads the injected seam, or [time.Now] when none is set.
func (c *Clock) realNow() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// newFrame builds a frame, capturing the real now as its timeWas base.
func (c *Clock) newFrame(mode Mode, target time.Time, scale float64) frame {
	return frame{mode: mode, target: target, timeWas: c.realNow(), scale: scale}
}

// Current reports the current mock time: the top stack frame's time, or the real
// now when the stack is empty. It is the analogue of Ruby's mocked Time.now.
func (c *Clock) Current() time.Time {
	if len(c.stack) == 0 {
		return c.realNow()
	}
	return c.stack[len(c.stack)-1].at(c.realNow())
}

// Freeze pushes a freeze frame pinning time at t and returns the resulting
// current mock time (which equals t). Mirrors Timecop.freeze(t) without a block.
func (c *Clock) Freeze(t time.Time) time.Time {
	c.stack = append(c.stack, c.newFrame(ModeFreeze, t, 0))
	return c.Current()
}

// Travel pushes a travel frame jumping time to t; time then keeps advancing at
// the real rate. Returns the resulting current mock time (≈ t). Mirrors
// Timecop.travel(t) without a block.
func (c *Clock) Travel(t time.Time) time.Time {
	c.stack = append(c.stack, c.newFrame(ModeTravel, t, 0))
	return c.Current()
}

// Scale pushes a scale frame: time jumps to t and then advances at factor× the
// real rate. Returns the resulting current mock time (≈ t). Mirrors
// Timecop.scale(factor, t) without a block.
func (c *Clock) Scale(factor float64, t time.Time) time.Time {
	c.stack = append(c.stack, c.newFrame(ModeScale, t, factor))
	return c.Current()
}

// withPush pushes f, runs fn, then restores the pre-push stack — even if fn
// panics (the panic still propagates). This is timecop's block form: push,
// yield, restore backup in an ensure block.
func (c *Clock) withPush(f frame, fn func()) {
	backup := c.stack
	c.stack = append(append([]frame(nil), backup...), f)
	defer func() { c.stack = backup }()
	fn()
}

// WithFrozen runs fn with time frozen at t, then pops the frame (even on panic).
// Mirrors Timecop.freeze(t) { ... }.
func (c *Clock) WithFrozen(t time.Time, fn func()) {
	c.withPush(c.newFrame(ModeFreeze, t, 0), fn)
}

// WithTravel runs fn with time travelled to t, then pops the frame (even on
// panic). Mirrors Timecop.travel(t) { ... }.
func (c *Clock) WithTravel(t time.Time, fn func()) {
	c.withPush(c.newFrame(ModeTravel, t, 0), fn)
}

// WithScale runs fn with time scaled by factor from t, then pops the frame (even
// on panic). Mirrors Timecop.scale(factor, t) { ... }.
func (c *Clock) WithScale(factor float64, t time.Time, fn func()) {
	c.withPush(c.newFrame(ModeScale, t, factor), fn)
}

// Return unwinds all mock-time frames and clears any baseline, reverting to the
// real clock. Mirrors Timecop.return / Timecop.unfreeze without a block.
func (c *Clock) Return() {
	c.stack = nil
	c.baseline = time.Time{}
	c.hasBaseline = false
}

// WithReturn reverts to the real clock for the duration of fn, then restores the
// previous stack and baseline — even if fn panics. Mirrors Timecop.return { ... }.
func (c *Clock) WithReturn(fn func()) {
	backupStack := c.stack
	backupBaseline := c.baseline
	backupHas := c.hasBaseline
	c.stack = nil
	c.baseline = time.Time{}
	c.hasBaseline = false
	defer func() {
		c.stack = backupStack
		c.baseline = backupBaseline
		c.hasBaseline = backupHas
	}()
	fn()
}

// SetBaseline records t as the baseline and pushes a travel frame at it, so the
// clock travels to the baseline. [Clock.ReturnToBaseline] later unwinds back to
// this frame. Mirrors Timecop's `baseline=`.
func (c *Clock) SetBaseline(t time.Time) {
	c.baseline = t
	c.hasBaseline = true
	c.stack = append(c.stack, c.newFrame(ModeTravel, t, 0))
}

// Baseline returns the current baseline and whether one is set.
func (c *Clock) Baseline() (time.Time, bool) {
	return c.baseline, c.hasBaseline
}

// ReturnToBaseline unwinds every nested frame back down to the baseline frame
// (keeping only the bottom-most frame), or reverts entirely to the real clock
// when no baseline is set. It returns the resulting current mock time. Mirrors
// Timecop.return_to_baseline.
func (c *Clock) ReturnToBaseline() time.Time {
	if c.hasBaseline {
		// Invariant: a set baseline always leaves at least the baseline frame
		// on the stack, so index 0 is safe. Equivalent to `[stack.shift]`.
		c.stack = c.stack[:1:1]
	} else {
		c.Return()
	}
	return c.Current()
}

// Mocked reports whether any mock-time frame is active (Time.now is faked).
func (c *Clock) Mocked() bool { return len(c.stack) > 0 }

// Depth reports the number of active mock-time frames on the stack.
func (c *Clock) Depth() int { return len(c.stack) }

// top returns the top frame's mode and true, or false when the stack is empty.
func (c *Clock) top() (Mode, bool) {
	if len(c.stack) == 0 {
		return 0, false
	}
	return c.stack[len(c.stack)-1].mode, true
}

// Frozen reports whether the top frame is a freeze. Mirrors Timecop.frozen?.
func (c *Clock) Frozen() bool { m, ok := c.top(); return ok && m == ModeFreeze }

// Travelled reports whether the top frame is a travel. Mirrors Timecop.travelled?.
func (c *Clock) Travelled() bool { m, ok := c.top(); return ok && m == ModeTravel }

// Scaled reports whether the top frame is a scale. Mirrors Timecop.scaled?.
func (c *Clock) Scaled() bool { m, ok := c.top(); return ok && m == ModeScale }

// ScalingFactor returns the top frame's scaling factor and true when the top
// frame is a scale, or 0 and false otherwise. Mirrors Timecop.scaling_factor.
func (c *Clock) ScalingFactor() (float64, bool) {
	if len(c.stack) == 0 {
		return 0, false
	}
	top := c.stack[len(c.stack)-1]
	if top.mode != ModeScale {
		return 0, false
	}
	return top.scale, true
}
