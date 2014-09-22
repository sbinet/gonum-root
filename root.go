// Copyright 2014 The Gonum Authors. All rights reserved.
// Use of this code is governed by a BSD-style
// license that can be found in the LICENSE file

package root

import (
	"errors"
	"math"
	"sync"
)

var (
	ErrMaxEval = errors.New("root: maximum function evaluations reached")
	ErrNaN     = errors.New("root: NaN function value")
	ErrNoRoot  = errors.New("root: no root could be found")
)

const noRootIter = 100 // number of iterations before there must be a function error

type bound struct {
	loc   float64
	value float64
}

// Settings represents ways to customize the root finding method. UseMinValue
// allows the user to specify the function value at the minimum bound provided by
// the MinValue field (and likewise for max). MaxEvals puts a cap on the number
// of function values allowed. If such a cap is reached, ErrMaxEval will be returned.
// If MaxEvals <= 0, no such cap is placed. Finally, concurrent and MaxWorkers
// allows the function to be multiple times concurrently if deemed appropriate
// by the root finding method.
type Settings struct {
	UseMinValue bool    // Have the minimum value
	MinValue    float64 // Value of the objective function at the minimum
	UseMaxValue bool    // Have the maximum value
	MaxValue    float64 // Value of the objective function at the maximum
	MaxEvals    int     // maximum number of function evaluations
	Concurrent  bool    // Enable concurrency (when appropriate)
	MaxWorkers  int     // Concurrency level?
}

// Find finds an x for which abs(f(x)) < tol. Min and max provide lower and upper
// bounds for the value of f.
//
// Min may be -inf and/or max may be +inf. If either is infinity, a search will be
// done to bound the value of the zero. The search assumes that going "sufficiently far"
// in the direction toward the infinite bound will find a point of opposite sign.
//
// Find will return ErrNan if the function evaluates to NaN, and will return
// ErrNoRoot if no root is found. See the documentation of the settings structure
// for additional behavior.
func Find(f func(float64) float64, min, max, tol float64, settings *Settings) (float64, error) {

	// getBoundVals deals with all of the inf cases and does some error checking
	// it ensures the bounds are finite and well posed
	minBound, maxBound, nFunEvals, err := getBoundVals(f, min, max, tol, settings)

	if err != nil {
		return minBoundValue(minBound, maxBound), err
	}
	if math.Abs(minBound.value) < tol {
		return minBound.loc, nil
	}
	if math.Abs(maxBound.value) < tol {
		return maxBound.loc, nil
	}

	// Choose a root finding method:
	// TODO: Here's how it should be:
	// GoldenRule if serial and no bound
	// Parallel bisection if concurrent
	// Fibbonacci if MaxEvals is "small"

	var maxEvals int
	if settings == nil {
		maxEvals = 0
	} else {
		maxEvals = settings.MaxEvals
	}

	// For now, just do a bisection
	return bisection(f, minBound, maxBound, tol, nFunEvals, maxEvals)
}

// minBoundValue returns the location of the bound whose location is closer to zero
func minBoundValue(a, b bound) float64 {
	absA := math.Abs(a.value)
	absB := math.Abs(b.value)
	return math.Min(absA, absB)
}

// getBoundVals takes in the user-specified minimum and maximum, and returns
// finite bounds. At return of this function, minBounds and maxBounds have finite
// locations, have the respective function values, minBounds.loc < maxBounds.loc, and
// the two values will be of opposite sign. It will also return the number of
// times the function is evaluated and any error encountered. If an error is encountered,
// one or more of the bounds may not meet the conditions listed above.
func getBoundVals(f func(float64) float64, min, max float64, tol float64, settings *Settings) (
	minBounds, maxBounds bound, nFunEvals int, err error) {

	if min >= max {
		panic("minimum less than maximum")
	}

	// Cases:
	// If the value is known, use it, unless the bound is infinite, which is an error
	// If the value is not known, and the bound is finite, compute it.
	// If one of the bounds is infinite, search in that direction to find a crossover
	// point
	// If both of the bounds are finite, evaluate at 0 and 1 to find the appropriate direction

	minInf := math.IsInf(min, -1)
	maxInf := math.IsInf(max, 1)
	if settings != nil && settings.UseMinValue {
		if minInf {
			// Maybe this should panic? It's programmer error
			err = errors.New("Using minimum value when minimum bound is infinity")
			return
		}
		if math.Abs(settings.MaxValue) < tol {
			return
		}
	}
	if settings != nil && settings.UseMaxValue {
		if maxInf {
			// Maybe this should panic? It's programmer error
			err = errors.New("Using maximum value when maximum bound is infinity")
			return
		}
		if math.Abs(settings.MaxValue) < tol {
			return
		}
	}

	if minInf || maxInf {
		return getBoundsInfinite(f, min, max, settings)
	} else {
		minVal, maxVal, nFunEvals := getBoundsFinite(f, min, max, settings)
		return bound{loc: min, value: minVal},
			bound{loc: max, value: maxVal},
			nFunEvals, nil
	}
	panic("unreachable")
}

// getBoundsFinite sets the bounds for when both bounds are finite
func getBoundsFinite(f func(float64) float64, min, max float64, settings *Settings) (
	minVal, maxVal float64, nFunEvals int) {

	// Have fixed locations for both bounds, so see
	var wg *sync.WaitGroup
	if settings != nil && settings.Concurrent {
		wg = &sync.WaitGroup{}
	}

	if settings != nil && settings.UseMinValue {
		minVal = settings.MinValue
	} else {
		if settings != nil && settings.Concurrent {
			wg.Add(1)
			go func() {
				defer wg.Done()
				minVal = f(min)
			}()
		} else {
			minVal = f(min)
		}
		nFunEvals++
	}

	if settings != nil && settings.UseMaxValue {
		maxVal = settings.MaxValue
	} else {
		if settings != nil && settings.Concurrent {
			wg.Add(1)
			go func() {
				defer wg.Done()
				maxVal = f(max)
			}()
		} else {
			maxVal = f(max)
		}
		nFunEvals++
	}

	if settings != nil && settings.Concurrent {
		wg.Wait()
	}
	return minVal, maxVal, nFunEvals
}

// getBoundsInfinite finds finite bounds when one or more of the user specified
// bounds are infinite.
func getBoundsInfinite(f func(float64) float64, min, max float64, settings *Settings) (
	minBound, maxBound bound, nFunEvals int, err error) {

	minInf := math.IsInf(min, -1)
	maxInf := math.IsInf(max, 1)

	if !minInf {
		minBound = bound{loc: min}
		if settings != nil && settings.UseMinValue {
			minBound.value = settings.MinValue
		} else {
			minBound.value = f(min)
		}
	}

	if !maxInf {
		maxBound = bound{loc: max}
		if settings != nil && settings.UseMaxValue {
			maxBound.value = settings.MaxValue
		} else {
			maxBound.value = f(max)
		}
	}

	// If both are infinity, use a local search to find the direction to search
	if minInf && maxInf {
		var (
			wg   *sync.WaitGroup
			zero float64
			one  float64
		)
		if settings != nil && settings.Concurrent {
			wg = &sync.WaitGroup{}
			wg.Add(2)
			go func() {
				defer wg.Done()
				zero = f(0)
			}()
			go func() {
				defer wg.Done()
				one = f(1)
			}()
			wg.Wait()
		} else {
			zero = f(0)
			one = f(1)
		}
		nFunEvals += 2

		signsOpposite := (zero > 0 && one < 0) || (zero < 0 && one > 0)
		if signsOpposite {
			minBound.loc = 0
			maxBound.loc = 1
			minBound.value = zero
			maxBound.value = one
			return minBound, maxBound, nFunEvals, nil
		} else {
			if zero > 0 && one > 0 {
				if zero < one {
					// Gradient points negative, so zero is the maximum, minimum
					// stays -inf
					maxBound.loc = 0
					maxBound.value = zero
					maxInf = false
				} else {
					// Gradient points positive, so one is the minimum, and maximum
					// stays +inf
					minBound.loc = 1
					minBound.value = one
					minInf = false
				}
			} else {
				if zero > one {
					// Gradient points negative, so zero is the maximum, minimum
					// stays -inf
					maxBound.loc = 0
					maxBound.value = zero
					maxInf = false
				} else {
					// Gradient points positive, so one is the minimum, and maximum
					// stays +inf
					minBound.loc = 1
					minBound.value = one
					minInf = false
				}
			}
		}
	}

	// Now we have exactly one infinite value, so do a search to find a point
	// of opposite sign.

	var maxEvals int
	if settings != nil {
		maxEvals = settings.MaxEvals
	}

	if maxInf {
		return infSearch(f, minBound, 1, nFunEvals, maxEvals)
	}

	if minInf {
		return infSearch(f, maxBound, -1, nFunEvals, maxEvals)
	}
	panic("unreachable")
}

// infSearch searches in a direction until the bound is found. Dir should be
// -1 or 1. The search is performed by stepping 1 away from the starting point,
// testing if has opposite sign, and doubling the step otherwise.
func infSearch(f func(float64) float64, start bound, dir float64, nFunEvals, maxFunEvals int) (
	minBound, maxBound bound, newFunEvals int, err error) {

	step := 1.0
	newStart := start
	for i := 0; i < noRootIter; i++ {
		if maxFunEvals > 0 && nFunEvals > maxFunEvals {
			break
		}
		newLoc := start.loc + step*dir
		newValue := f(newLoc)
		nFunEvals++

		if math.IsNaN(newValue) {
			if dir == 1 {
				return newStart, bound{loc: math.NaN()}, nFunEvals, ErrNaN
			}
			return bound{loc: math.NaN()}, newStart, nFunEvals, ErrNaN
		}

		if newValue == 0 || !sameSign(newStart.value, newValue) {
			// We have found and/or bound the zero
			if dir == 1 {
				return newStart, bound{loc: newLoc, value: newValue}, nFunEvals, nil
			}
			return bound{loc: newLoc, value: newValue}, newStart, nFunEvals, nil
		}
		// The new point had the same sign, so update the minimum value to there
		newStart.loc = newLoc
		newStart.value = newValue

		// Double the step to keep trying
		step *= 2
	}
	if dir == 1 {
		return newStart, bound{loc: math.NaN()}, nFunEvals, ErrMaxEval
	}
	return bound{loc: math.NaN()}, newStart, nFunEvals, ErrMaxEval
}

func sameSign(a, b float64) bool {
	return (a > 0 && b > 0) || (a < 0 && b < 0)
}
