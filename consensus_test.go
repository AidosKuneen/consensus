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

func TestShouldCloseLedger(t *testing.T) {
	if !shouldCloseLedger(true, 10, 10, 10, -10*time.Second,
		10*time.Second, 1*time.Second, 1*time.Second) {
		t.Error()
	}
	if !shouldCloseLedger(true, 10, 10, 10, 100*time.Hour,
		10*time.Second, 1*time.Second, 1*time.Second) {
		t.Error()
	}
	if !shouldCloseLedger(true, 10, 10, 10, 10*time.Second,
		100*time.Hour, 1*time.Second, 1*time.Second) {
		t.Error()
	}

	if !shouldCloseLedger(true, 10, 3, 5, 10*time.Second,
		10*time.Second, 10*time.Second, 10*time.Second) {
		t.Error()
	}

	if shouldCloseLedger(false, 10, 0, 0, 1*time.Second,
		1*time.Second, 1*time.Second, 10*time.Second) {
		t.Error()
	}
	if !shouldCloseLedger(false, 10, 0, 0, 1*time.Second,
		10*time.Second, 1*time.Second, 10*time.Second) {
		t.Error()
	}

	if shouldCloseLedger(true, 10, 0, 0, 10*time.Second,
		10*time.Second, 1*time.Second, 10*time.Second) {
		t.Error()
	}

	if shouldCloseLedger(true, 10, 0, 0, 10*time.Second,
		10*time.Second, 3*time.Second, 10*time.Second) {
		t.Error()
	}
	if !shouldCloseLedger(true, 10, 0, 0, 10*time.Second,
		10*time.Second, 10*time.Second, 10*time.Second) {
		t.Error()
	}
}
func TestCheckConsensus(t *testing.T) {
	if checkConsensus(10, 2, 2, 0, 3*time.Second,
		2*time.Second, true) != StateNo {
		t.Error()
	}
	if checkConsensus(10, 2, 2, 0, 3*time.Second,
		4*time.Second, true) != StateNo {
		t.Error()
	}
	if checkConsensus(10, 2, 2, 0, 3*time.Second,
		10*time.Second, true) != StateYes {
		t.Error()
	}
	if checkConsensus(10, 2, 1, 0, 3*time.Second,
		10*time.Second, true) != StateNo {
		t.Error()
	}
	if checkConsensus(10, 2, 1, 8, 3*time.Second,
		10*time.Second, true) != StateMovedOn {
		t.Error()
	}
	if checkConsensus(0, 0, 0, 0, 3*time.Second,
		10*time.Second, true) != StateYes {
		t.Error()
	}
}
