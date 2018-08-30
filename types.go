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
	"time"
)

/** Represents how a node currently participates in Consensus.

   A node participates in consensus in varying modes, depending on how
   the node was configured by its operator and how well it stays in sync
   with the network during consensus.

   @code
     proposing               observing
        \                       /
         \---> wrongLedger <---/
                    ^
                    |
                    |
                    v
               switchedLedger
  @endcode

  We enter the round proposing or observing. If we detect we are working
  on the wrong prior ledger, we go to wrongLedger and attempt to acquire
  the right one. Once we acquire the right one, we go to the switchedLedger
  mode.  It is possible we fall behind again and find there is a new better
  ledger, moving back and forth between wrongLedger and switchLedger as
  we attempt to catch up.
*/

type consensusMode byte

const (
	//! We are normal participant in consensus and propose our position
	proposing consensusMode = iota
	//! We are observing peer positions, but not proposing our position
	observing
	//! We have the wrong ledger and are attempting to acquire it
	wrongLedger
	//! We switched ledgers since we started this consensus round but are now
	//! running on what we believe is the correct ledger.  This mode is as
	//! if we entered the round observing, but is used to indicate we did
	//! have the wrongLedger at some point.
	switchedLedger
)

func (m consensusMode) String() string {
	switch m {
	case proposing:
		return "proposing"
	case observing:
		return "observing"
	case wrongLedger:
		return "wrongLedger"
	case switchedLedger:
		return "switchedLedger"
	default:
		return "unknown"
	}
}

type consensusPhase byte

/** Phases of consensus for a single ledger round.

   @code
         "close"             "accept"
    open ------- > establish ---------> accepted
      ^               |                    |
      |---------------|                    |
      ^                     "startRound"   |
      |------------------------------------|
  @endcode

  The typical transition goes from open to establish to accepted and
  then a call to startRound begins the process anew. However, if a wrong prior
  ledger is detected and recovered during the establish or accept phase,
  consensus will internally go back to open (see Consensus::handleWrongLedger).
*/
const (
	//! We haven't closed our ledger yet, but others might have
	open consensusPhase = iota

	//! Establishing consensus by exchanging proposals with our peers
	establish

	//! We have accepted a new last closed ledger and are waiting on a call
	//! to startRound to begin the next consensus round.  No changes
	//! to consensus phase occur while in this phase.
	accepted
)

func (p consensusPhase) String() string {
	switch p {
	case open:
		return "open"
	case establish:
		return "establish"
	case accepted:
		return "accepted"
	default:
		return "unknown"
	}
}

type consensusTimer struct {
	start time.Time
	dur   time.Duration
}

func (ct consensusTimer) read() time.Duration {
	return ct.dur
}

func (ct consensusTimer) tick(fixed time.Duration) {
	ct.dur += fixed
}

func (ct consensusTimer) reset(tp time.Time) {
	ct.start = tp
	ct.dur = 0
}

func (ct consensusTimer) tickTime(tp time.Time) {
	ct.dur = tp.Sub(ct.start)
}

/** Stores the set of initial close times

  The initial consensus proposal from each peer has that peer's view of
  when the ledger closed.  This object stores all those close times for
  analysis of clock drift between peers.
*/
type consensusCloseTimes struct {
	//! Close time estimates, keep ordered for predictable traverse
	peers map[unixTime]int

	//! Our close time estimate
	self time.Time
}

func newConsensusCloseTimes() *consensusCloseTimes {
	return &consensusCloseTimes{
		peers: make(map[unixTime]int),
	}
}

/** Whether we have or don't have a consensus */
type consensusState byte

const (
	no      consensusState = iota //!< We do not have consensus
	movedOn                       //!< The network has consensus without us
	yes                           //!< We have consensus along with the network
)

/** Encapsulates the result of consensus.

   Stores all relevant data for the outcome of consensus on a single
  ledger.

   @tparam Traits Traits class defining the concrete consensus types used
                  by the application.
*/
type consensusResult struct {
	//! The set of transactions consensus agrees go in the ledger
	txns txSet

	//! Our proposed position on transactions/close time
	position consensusProposal

	//! Transactions which are under dispute with our peers
	disputes map[TxID]*disputedTx

	// Set of TxSet ids we have already compared/created disputes
	compares map[TxSetID]txSet

	// Measures the duration of the establish phase for this consensus round
	roundTime consensusTimer

	// Indicates state in which consensus ended.  Once in the accept phase
	// will be either Yes or MovedOn
	state consensusState

	// The number of peers proposing during the round
	proposers uint
}

func newConsensusResult(txns txSet, pos consensusProposal) *consensusResult {
	if txns.ID() != pos.position {
		panic("invalid txSet and proposal")
	}
	return &consensusResult{
		txns:     txns,
		position: pos,
		state:    no,
		disputes: make(map[TxID]*disputedTx),
		compares: make(map[TxSetID]txSet),
	}

}
