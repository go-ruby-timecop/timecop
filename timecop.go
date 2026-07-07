// Copyright (c) the go-ruby-timecop/timecop authors
//
// SPDX-License-Identifier: BSD-3-Clause

package timecop

import "time"

// std is the process-wide default Clock, the analogue of timecop's Timecop
// singleton. The package-level functions below operate on it so that a host or a
// program can use the Timecop module surface directly, while [Clock] remains
// available for isolated instances (e.g. per-thread in a host).
var std = New()

// Default returns the process-wide default [Clock] backing the package-level
// functions.
func Default() *Clock { return std }

// SetNow installs now as the real-clock seam of the default clock. Passing nil
// restores the real system clock ([time.Now]). Tests use this to make the
// default clock deterministic.
func SetNow(now func() time.Time) { std.Now = now }

// Now reports the default clock's current mock time — the analogue of Ruby's
// mocked Time.now.
func Now() time.Time { return std.Current() }

// Freeze freezes the default clock at t. Mirrors Timecop.freeze(t).
func Freeze(t time.Time) time.Time { return std.Freeze(t) }

// Travel travels the default clock to t. Mirrors Timecop.travel(t).
func Travel(t time.Time) time.Time { return std.Travel(t) }

// Scale scales the default clock by factor from t. Mirrors Timecop.scale(factor, t).
func Scale(factor float64, t time.Time) time.Time { return std.Scale(factor, t) }

// WithFrozen runs fn with the default clock frozen at t. Mirrors Timecop.freeze(t) { }.
func WithFrozen(t time.Time, fn func()) { std.WithFrozen(t, fn) }

// WithTravel runs fn with the default clock travelled to t. Mirrors Timecop.travel(t) { }.
func WithTravel(t time.Time, fn func()) { std.WithTravel(t, fn) }

// WithScale runs fn with the default clock scaled by factor from t. Mirrors
// Timecop.scale(factor, t) { }.
func WithScale(factor float64, t time.Time, fn func()) { std.WithScale(factor, t, fn) }

// Return unwinds the default clock to real time. Mirrors Timecop.return.
func Return() { std.Return() }

// WithReturn runs fn with the default clock reverted to real time, then restores
// it. Mirrors Timecop.return { }.
func WithReturn(fn func()) { std.WithReturn(fn) }

// SetBaseline sets the default clock's baseline. Mirrors Timecop's `baseline=`.
func SetBaseline(t time.Time) { std.SetBaseline(t) }

// Baseline returns the default clock's baseline. Mirrors Timecop.baseline.
func Baseline() (time.Time, bool) { return std.Baseline() }

// ReturnToBaseline unwinds the default clock to its baseline. Mirrors
// Timecop.return_to_baseline.
func ReturnToBaseline() time.Time { return std.ReturnToBaseline() }

// Frozen reports whether the default clock is frozen. Mirrors Timecop.frozen?.
func Frozen() bool { return std.Frozen() }

// Travelled reports whether the default clock is travelled. Mirrors Timecop.travelled?.
func Travelled() bool { return std.Travelled() }

// Scaled reports whether the default clock is scaled. Mirrors Timecop.scaled?.
func Scaled() bool { return std.Scaled() }

// Mocked reports whether the default clock is mocking time.
func Mocked() bool { return std.Mocked() }
