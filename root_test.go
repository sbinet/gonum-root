// Copyright 2014 The Gonum Authors. All rights reserved.
// Use of this code is governed by a BSD-style
// license that can be found in the LICENSE file

package root

import (
	"math"
	"testing"
)

func TestFind(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Fun      func(x float64) float64
		Min      float64
		Max      float64
		Tol      float64
		Ans      float64
		Settings *Settings
	}{
		{
			Name: "EasyLinear",
			Fun:  func(x float64) float64 { return x - 7 },
			Min:  -3,
			Max:  10,
			Tol:  1e-14,
			Ans:  7,
		},
		{
			Name: "NegInfLinear",
			Fun:  func(x float64) float64 { return x - 7 },
			Min:  math.Inf(-1),
			Max:  9.5,
			Tol:  1e-14,
			Ans:  7,
		},
		{
			Name: "PosInfLinear",
			Fun:  func(x float64) float64 { return x - 7 },
			Min:  0.1,
			Max:  math.Inf(1),
			Tol:  1e-14,
			Ans:  7,
		},
		{
			Name: "BothInfLinear",
			Fun:  func(x float64) float64 { return x - 7 },
			Min:  math.Inf(-1),
			Max:  math.Inf(1),
			Tol:  1e-14,
			Ans:  7,
		},
	} {
		ans, err := Find(test.Fun, test.Min, test.Max, test.Tol, test.Settings)
		if err != nil {
			t.Errorf("Case %v: error in Find: %v", test.Name, err)
			continue
		}
		if math.Abs(ans-test.Ans) > test.Tol {
			t.Errorf("Case %v: tolerance not met. Want %v, Got %v", test.Name, test.Ans, ans)
		}
	}
}
