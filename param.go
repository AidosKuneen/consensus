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

import "time"

/** Consensus algorithm parameters

  Parameters which control the consensus algorithm.  This are not
  meant to be changed arbitrarily.
*/

var (
	// Validation and proposal durations are relative to NetClock times, so use
	// second resolution

	//! How long we consider a proposal fresh
	proposeFreshness = 20 * time.Second

	//! How often we force generating a new proposal to keep ours fresh
	proposeInterval = 12 * time.Second

	//-------------------------------------------------------------------------
	// Consensus durations are relative to the internal Consenus clock and use
	// millisecond resolution.

	//! The percentage threshold above which we can declare consensus.
	minConsensusPCT = 80

	//LedgerIdleInterval is the duration a ledger may remain idle before closing
	LedgerIdleInterval = 15 * time.Second

	//! The number of seconds we wait minimum to ensure participation
	ledgerMinConsensus = 1950 * time.Millisecond

	//! Minimum number of seconds to wait to ensure others have computed the LCL
	ledgerMinClose = 2 * time.Second

	//LedgerGranularity determines how often we check state or change positions
	LedgerGranularity = 1 * time.Second

	//LedgerPrevInterval defines interval between ledgers.
	LedgerPrevInterval = 10 * time.Minute

	/** The minimum amount of time to consider the previous round
	  to have taken.

	  The minimum amount of time to consider the previous round
	  to have taken. This ensures that there is an opportunity
	  for a round at each avalanche threshold even if the
	  previous consensus was very fast. This should be at least
	  twice the interval between proposals (0.7s) divided by
	  the interval between mid and late consensus ([85-50]/100).
	*/
	avMinConsensusTime = 5 * time.Second

	//------------------------------------------------------------------------------
	// Avalanche tuning
	// As a function of the percent this round's duration is of the prior round,
	// we increase the threshold for yes vots to add a tranasaction to our
	// position.

	//! Percentage of nodes on our UNL that must vote yes
	avInitConsensusPCT = 50

	//! Percentage of previous round duration before we advance
	avMidConsensusTime = 50

	//! Percentage of nodes that most vote yes after advancing
	avMidConsensusPCT = 65

	//! Percentage of previous round duration before we advance
	avLateConsensusTime = 85

	//! Percentage of nodes that most vote yes after advancing
	avLateConsensusPCT = 70

	//! Percentage of previous round duration before we are stuck
	avStuckConsensusTime = 200

	//! Percentage of nodes that must vote yes after we are stuck
	avStuckConsensusPCT = 95

	//! Percentage of nodes required to reach agreement on ledger close time
	avCTConsensusPCT = 75

	//--------------------------------------------------------------------------

	/** Whether to use roundCloseTime or effCloseTime for reaching close time
	  consensus.
	  This was added to migrate from effCloseTime to roundCloseTime on the
	  live network. The desired behavior (as given by the default value) is
	  to use roundCloseTime during consensus voting and then use effCloseTime
	  when accepting the consensus ledger.
	*/
	useRoundedCloseTime = true
)

//SetUseRoundedCloseTime is only for testing.
func SetUseRoundedCloseTime(y bool) {
	useRoundedCloseTime = y
}
