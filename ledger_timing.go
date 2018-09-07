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

// This is a rewrite of https://github.com/ripple/rippled/src/ripple/consensus
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
	"log"
	"time"
)

/**  Possible ledger close time resolutions.

  Values should not be duplicated.
  @see getNextLedgerTimeResolution
*/
var ledgerPossibleTimeResolutions = []time.Duration{
	10 * time.Second,
	20 * time.Second,
	30 * time.Second,
	60 * time.Second,
	90 * time.Second,
	120 * time.Second,
}

// LedgerDefaultTimeResolution is the initial resolution of ledger close time.
var LedgerDefaultTimeResolution = ledgerPossibleTimeResolutions[2]

const (
	//! How often we increase the close time resolution (in numbers of ledgers)
	increaseLedgerTimeResolutionEvery = 8

	//! How often we decrease the close time resolution (in numbers of ledgers)
	decreaseLedgerTimeResolutionEvery = 1
)

/** Calculates the close time resolution for the specified ledger.

  The Ripple protocol uses binning to represent time intervals using only one
  timestamp. This allows servers to derive a common time for the next ledger,
  without the need for perfectly synchronized clocks.
  The time resolution (i.e. the size of the intervals) is adjusted dynamically
  based on what happened in the last ledger, to try to avoid disagreements.

  @param previousResolution the resolution used for the prior ledger
  @param previousAgree whether consensus agreed on the close time of the prior
  ledger
  @param ledgerSeq the sequence number of the new ledger

  @pre previousResolution must be a valid bin
       from @ref ledgerPossibleTimeResolutions

  @tparam Rep Type representing number of ticks in std::chrono::duration
  @tparam Period An std::ratio representing tick period in
                 std::chrono::duration
  @tparam Seq Unsigned integer-like type corresponding to the ledger sequence
              number. It should be comparable to 0 and support modular
              division. Built-in and tagged_integers are supported.
*/

func getNextLedgerTimeResolution(
	previousResolution time.Duration,
	previousAgree bool,
	ledgerSeq Seq) time.Duration {

	if ledgerSeq == 0 {
		panic("invalid ledger")
	}

	// Find the current resolution:
	iter := 0
	for iter = 0; iter < len(ledgerPossibleTimeResolutions); iter++ {
		if ledgerPossibleTimeResolutions[iter] == previousResolution {
			break
		}
	}
	if iter == len(ledgerPossibleTimeResolutions) {
		log.Println(previousResolution)
		panic("invalid resolution")
	}

	// If we did not previously agree, we try to decrease the resolution to
	// improve the chance that we will agree now.
	if !previousAgree &&
		ledgerSeq%decreaseLedgerTimeResolutionEvery == 0 {
		if iter++; iter != len(ledgerPossibleTimeResolutions) {
			return ledgerPossibleTimeResolutions[iter]
		}
	}

	// If we previously agreed, we try to increase the resolution to determine
	// if we can continue to agree.
	if previousAgree &&
		ledgerSeq%increaseLedgerTimeResolutionEvery == 0 {
		if iter--; iter >= 0 {
			return ledgerPossibleTimeResolutions[iter]
		}
	}
	return previousResolution
}

/** Calculates the close time for a ledger, given a close time resolution.

  @param closeTime The time to be rouned.
  @param closeResolution The resolution
  @return @b closeTime rounded to the nearest multiple of @b closeResolution.
  Rounds up if @b closeTime is midway between multiples of @b closeResolution.
*/
func roundCloseTime(
	closeTime time.Time,
	closeResolution time.Duration) time.Time {
	return closeTime.Round(closeResolution)
}

//EffCloseTime calculate the effective ledger close time
//   After adjusting the ledger close time based on the current resolution, also
//   ensure it is sufficiently separated from the prior close time.
//   @param closeTime The raw ledger close time
//   @param resolution The current close time resolution
//   @param priorCloseTime The close time of the prior ledger
func EffCloseTime(
	closeTime time.Time,
	resolution time.Duration,
	priorCloseTime time.Time) time.Time {

	if closeTime.IsZero() {
		return closeTime
	}
	p1s := priorCloseTime.Add(time.Second)
	round := closeTime.Round(resolution)
	if p1s.After(round) {
		return p1s
	}
	return round
}
