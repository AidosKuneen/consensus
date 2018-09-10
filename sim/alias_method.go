// Copyright (c) 2018 Aidos Developer

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// This is a rewrite of https://github.com/ripple/rippled/src/test/csf
// covered by:
//------------------------------------------------------------------------------
/*
   This file is part of rippled: https://github.com/ripple/rippled
   Copyright (c) 2012-2017 Ripple Labs Inc.

   Permission to use, copy, modify, and/or distribute this software for any
   purpose  with  or without fee is hereby granted, provided that the above
   copyright notice and this permission notice appear in all copies.

   THE  SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
   WITH  REGARD  TO  THIS  SOFTWARE  INCLUDING  ALL  IMPLIED  WARRANTIES  OF
   MERCHANTABILITY  AND  FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
   ANY  SPECIAL ,  DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
   WHATSOEVER  RESULTING  FROM  LOSS  OF USE, DATA OR PROFITS, WHETHER IN AN
   ACTION  OF  CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
   OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/
//==============================================================================

//
// Copyright (c) 2016 Tomochika Hara

// MIT License

// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:

// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package sim

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type aliasMethod struct {
	rand *rand.Rand
}

func newAliasMethod() *aliasMethod {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &aliasMethod{rand: r}
}

func (a *aliasMethod) random(w []float64) int {
	table, err := newAliasTable(w)
	if err != nil {
		panic(err)
	}
	u := a.rand.Float64()
	n := a.rand.Intn(table.Len)

	if u <= table.Prob[n] {
		return int(n)
	}
	return table.Alias[n]
}

type aliasTable struct {
	Len   int
	Prob  []float64
	Alias []int
}

func newAliasTable(weights []float64) (*aliasTable, error) {
	n := len(weights)

	sum := 0.0
	for _, value := range weights {
		sum += value
	}
	if sum == 0 {
		return nil, errors.New("sum of weights is 0")
	}

	prob := make([]float64, n)
	for i, w := range weights {
		prob[i] = float64(w) * float64(n) / float64(sum)
	}

	h := 0
	l := n - 1
	hl := make([]int, n)
	for i, p := range prob {
		if p < 1 {
			hl[l] = i
			l--
		}
		if p > 1 {
			hl[h] = i
			h++
		}
	}

	a := make([]int, n)
	for h != 0 && l != n-1 {
		j := hl[l+1]
		k := hl[h-1]

		if 1 < prob[j] {
			panic(fmt.Sprintf("MUST: %f <= 1", prob[j]))
		}
		if prob[k] < 1 {
			panic(fmt.Sprintf("MUST: 1 <= %f", prob[k]))
		}

		a[j] = k
		prob[k] -= (1 - prob[j]) // - residual weight
		l++
		if prob[k] < 1 {
			hl[l] = k
			l--
			h--
		}
	}

	return &aliasTable{Len: n, Prob: prob, Alias: a}, nil
}
