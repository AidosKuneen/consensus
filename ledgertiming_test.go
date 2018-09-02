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

// This is a rewrite of https://github.com/ripple/rippled/src/test/consensus
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

package consensus

import (
	"testing"
	"time"
)

// helper to iteratively call into getNextLedgerTimeResolution
func run(previousAgree bool, rounds Seq) (uint32, uint32, uint32) {
	var decrease, equal, increase uint32
	closeResolution := ledgerDefaultTimeResolution
	nextCloseResolution := closeResolution
	var round Seq
	for {
		round++
		nextCloseResolution = getNextLedgerTimeResolution(closeResolution, previousAgree, round)
		switch {
		case nextCloseResolution < closeResolution:
			decrease++
		case nextCloseResolution > closeResolution:
			increase++
		default:
			equal++
		}
		nextCloseResolution, closeResolution = closeResolution, nextCloseResolution
		if round >= rounds {
			break
		}
	}
	return increase, decrease, equal
}

func TestGetNextLedgerTimeResolution(t *testing.T) {
	// If we never agree on close time, only can increase resolution
	// until hit the max
	increase, decrease, equal := run(false, 10)
	if increase != 3 {
		t.Error("invalid increase", increase)
	}
	if decrease != 0 {
		t.Error("invalid decrease", decrease)
	}
	if equal != 7 {
		t.Error("invalid equal", equal)
	}

	// If we always agree on close time, only can decrease resolution
	// until hit the min
	increase, decrease, equal = run(false, 100)
	if increase != 3 {
		t.Error("invalid increase")
	}
	if decrease != 0 {
		t.Error("invalid decrease")
	}
	if equal != 97 {
		t.Error("invalid equal")
	}
}

func TestRoundCloseTime(t *testing.T) {
	// A closeTime equal to the epoch is not modified
	var def time.Time
	if def != roundCloseTime(def, 30*time.Second) {
		t.Error("invalid roundclosetime")
	}
	// Otherwise, the closeTime is rounded to the nearest
	if !roundCloseTime(time.Unix(29, 0), 60*time.Second).Equal(time.Unix(0, 0)) {
		t.Error("invalid roundclosetime")
	}
	if !roundCloseTime(time.Unix(30, 0), 1*time.Second).Equal(time.Unix(30, 0)) {
		t.Error("invalid roundclosetime")
	}
	if !roundCloseTime(time.Unix(31, 0), 60*time.Second).Equal(time.Unix(60, 0)) {
		t.Error("invalid roundclosetime")
	}
	if !roundCloseTime(time.Unix(30, 0), 60*time.Second).Equal(time.Unix(60, 0)) {
		t.Error("invalid roundclosetime")
	}
	if !roundCloseTime(time.Unix(59, 0), 60*time.Second).Equal(time.Unix(60, 0)) {
		t.Error("invalid roundclosetime")
	}
	if !roundCloseTime(time.Unix(60, 0), 60*time.Second).Equal(time.Unix(60, 0)) {
		t.Error("invalid roundclosetime")
	}
	if !roundCloseTime(time.Unix(61, 0), 60*time.Second).Equal(time.Unix(60, 0)) {
		t.Error("invalid roundclosetime")
	}
}

func TestEffCloseTime(t *testing.T) {
	close := effCloseTime(time.Unix(10, 0), 30*time.Second, time.Unix(0, 0))
	if !close.Equal(time.Unix(1, 0)) {
		t.Error("invalid effclosetime", close.Unix())
	}
	close = effCloseTime(time.Unix(16, 0), 30*time.Second, time.Unix(0, 0))
	if !close.Equal(time.Unix(30, 0)) {
		t.Error("invalid effclosetime")
	}
	close = effCloseTime(time.Unix(16, 0), 30*time.Second, time.Unix(30, 0))
	if !close.Equal(time.Unix(31, 0)) {
		t.Error("invalid effclosetime")
	}
	close = effCloseTime(time.Unix(16, 0), 30*time.Second, time.Unix(60, 0))
	if !close.Equal(time.Unix(61, 0)) {
		t.Error("invalid effclosetime")
	}
	close = effCloseTime(time.Unix(31, 0), 30*time.Second, time.Unix(0, 0))
	if !close.Equal(time.Unix(30, 0)) {
		t.Error("invalid effclosetime")
	}
}
