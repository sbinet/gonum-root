// Copyright 2014 The Gonum Authors. All rights reserved.
// Use of this code is governed by a BSD-style
// license that can be found in the LICENSE file

package root

import "math"

// unexported bisection assumes error checking has all been done, and that the values
// are correct at the bounds. nFunEvals is how many have already been done, and
// maxFunEvals is how many can be done
func bisection(f func(float64) float64, minBound, maxBound bound, tol float64, nFunEvals, maxFunEvals int) (float64, error) {
	for i := 0; i < noRootIter; i++ {
		mid := (minBound.loc + maxBound.loc) / 2
		midValue := f(mid)
		nFunEvals++
		if math.IsNaN(midValue) {
			return minBoundValue(minBound, maxBound), ErrNaN
		}

		if math.Abs(midValue) < tol {
			return mid, nil
		}

		if maxFunEvals > 0 && nFunEvals > maxFunEvals {
			return minBoundValue(minBound, maxBound), ErrMaxEval
		}

		// Did not find the minimum or error, update the bounds
		if sameSign(minBound.value, midValue) {
			minBound.loc = mid
			minBound.value = midValue
		} else {
			maxBound.loc = mid
			maxBound.value = midValue
		}
	}
	return minBoundValue(minBound, maxBound), ErrMaxEval
}
