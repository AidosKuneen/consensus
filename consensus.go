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

/** Determines whether the current ledger should close at this time.

  This function should be called when a ledger is open and there is no close
  in progress, or when a transaction is received and no close is in progress.

  @param anyTransactions indicates whether any transactions have been received
  @param prevProposers proposers in the last closing
  @param proposersClosed proposers who have currently closed this ledger
  @param proposersValidated proposers who have validated the last closed
                            ledger
  @param prevRoundTime time for the previous ledger to reach consensus
  @param timeSincePrevClose  time since the previous ledger's (possibly rounded)
                      close time
  @param openTime     duration this ledger has been open
  @param idleInterval the network's desired idle interval
  @param parms        Consensus constant parameters
  @param j            journal for logging
*/
func shouldCloseLedger(
	anyTransactions bool,
	prevProposers uint,
	proposersClosed uint,
	proposersValidated uint,
	prevRoundTime time.Duration,
	timeSincePrevClose time.Duration, // Time since last ledger's close time
	openTime time.Duration, // Time waiting to close this ledger
	idleInterval time.Duration) bool {

	if (prevRoundTime < -1*time.Second) || (prevRoundTime > 10*time.Minute) ||
		(timeSincePrevClose > 10*time.Minute) {
		// These are unexpected cases, we just close the ledger
		log.Println("shouldCloseLedger Trans=",
			anyTransactions,
			" Prop: ", prevProposers, "/", proposersClosed,
			" Secs: ", timeSincePrevClose,
			" (last: ", prevRoundTime, ")")
		return true
	}

	if (proposersClosed + proposersValidated) > (prevProposers / 2) {
		// If more than half of the network has closed, we close
		log.Println("Others have closed")
		return true
	}

	if !anyTransactions {
		// Only close at the end of the idle interval
		return timeSincePrevClose >= idleInterval // normal idle
	}

	// Preserve minimum ledger open time
	if openTime < ledgerMinClose {
		log.Println("Must wait minimum time before closing")
		return false
	}

	// Don't let this ledger close more than twice as fast as the previous
	// ledger reached consensus so that slower validators can slow down
	// the network
	if openTime < (prevRoundTime / 2) {
		log.Println("Ledger has not been open long enough")
		return false
	}

	// Close the ledger
	return true
}

/** Determine whether the network reached consensus and whether we joined.

  @param prevProposers proposers in the last closing (not including us)
  @param currentProposers proposers in this closing so far (not including us)
  @param currentAgree proposers who agree with us
  @param currentFinished proposers who have validated a ledger after this one
  @param previousAgreeTime how long, in milliseconds, it took to agree on the
                           last ledger
  @param currentAgreeTime how long, in milliseconds, we've been trying to
                          agree
  @param parms            Consensus constant parameters
  @param proposing        whether we should count ourselves
  @param j                journal for logging
*/

func checkConsensusReached(
	agreeing uint,
	total uint,
	countSelf bool,
	minConsensusPct int) bool {
	// If we are alone, we have a consensus
	if total == 0 {
		return true
	}
	if countSelf {
		agreeing++
		total++
	}

	currentPercentage := (agreeing * 100) / total

	return currentPercentage >= uint(minConsensusPct)
}

func checkConsensus(
	prevProposers uint,
	currentProposers uint,
	currentAgree uint,
	currentFinished uint,
	previousAgreeTime time.Duration,
	currentAgreeTime time.Duration,
	proposing bool) consensusState {
	log.Println("checkConsensus: prop=", currentProposers, "/",
		prevProposers, " agree=", currentAgree,
		" validated=", currentFinished,
		" time=", currentAgreeTime, "/",
		previousAgreeTime)

	if currentAgreeTime <= ledgerMinConsensus {
		return no
	}

	if currentProposers < (prevProposers * 3 / 4) {
		// Less than 3/4 of the last ledger's proposers are present; don't
		// rush: we may need more time.
		if currentAgreeTime < (previousAgreeTime + ledgerMinConsensus) {
			log.Println("too fast, not enough proposers")
			return no
		}
	}

	// Have we, together with the nodes on our UNL list, reached the threshold
	// to declare consensus?
	if checkConsensusReached(
		currentAgree, currentProposers, proposing, minConsensusPCT) {
		log.Println("normal consensus")
		return yes
	}

	// Have sufficient nodes on our UNL list moved on and reached the threshold
	// to declare consensus?
	if checkConsensusReached(
		currentFinished, currentProposers, false, minConsensusPCT) {
		log.Println("We see no consensus, but 80% of nodes have moved on")
		return movedOn
	}

	// no consensus yet
	log.Println("no consensus")
	return no
}
