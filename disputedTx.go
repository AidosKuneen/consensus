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

import "log"

/** A transaction discovered to be in dispute during conensus.

  During consensus, a @ref DisputedTx is created when a transaction
  is discovered to be disputed. The object persists only as long as
  the dispute.

  Undisputed transactions have no corresponding @ref DisputedTx object.

  Refer to @ref Consensus for details on the template type requirements.

  @tparam Tx_t The type for a transaction
  @tparam NodeID_t The type for a node identifier
*/

type mapT map[NodeID]bool

type disputedTx struct {
	yays    int  //< Number of yes votes
	nays    int  //< Number of no votes
	ourVote bool //< Our vote (true is yes)
	tx      txT  //< Transaction under dispute
	votes   mapT //< Map from NodeID to vote
}

func newDisputedTx(tr txT, ourVote bool, numPeers uint) *disputedTx {
	return &disputedTx{
		ourVote: ourVote,
		tx:      tr,
		votes:   make(mapT),
	}
}

func (dtx *disputedTx) setVote(peer NodeID, votesYes bool) {
	res, exist := dtx.votes[peer]
	dtx.votes[peer] = votesYes

	// new vote
	switch {
	case !exist:
		if votesYes {
			log.Println("Peer ", peer, " votes YES on ", dtx.tx.ID())
			dtx.yays++
		} else {
			log.Println("Peer ", peer, " votes NO on ", dtx.tx.ID())
			dtx.nays++
		}
	case votesYes && !res:
		// changes vote to yes
		log.Println("Peer ", peer, "now votes YES on ", dtx.tx.ID())
		dtx.nays--
		dtx.yays++
		// changes vote to no
	case !votesYes && res:
		log.Println("Peer ", peer, "now votes NO on ", dtx.tx.ID())
		dtx.nays++
		dtx.yays--
	}
}

// Remove a peer's vote on this disputed transasction
func (dtx *disputedTx) unVote(peer NodeID) {
	it, exist := dtx.votes[peer]

	if exist {
		if it {
			dtx.yays--
		} else {
			dtx.nays--
		}
		delete(dtx.votes, peer)
	}
}

func (dtx *disputedTx) updateVote(percentTime int, proposing bool) bool {
	if dtx.ourVote && dtx.nays == 0 {
		return false
	}
	if !dtx.ourVote && dtx.yays == 0 {
		return false
	}
	weight := 0
	newPosition := true
	if proposing { // give ourselves full weight
		// This is basically the percentage of nodes voting 'yes' (including us)
		vote := 0
		if dtx.ourVote {
			vote = 100
		}
		weight = (dtx.yays*100 + vote) / (dtx.nays + dtx.yays + 1)
		// To prevent avalanche stalls, we increase the needed weight slightly
		// over time.
		switch {
		case percentTime < avMidConsensusTime:
			newPosition = weight > avInitConsensusPCT
		case percentTime < avLateConsensusTime:
			newPosition = weight > avMidConsensusPCT
		case percentTime < avStuckConsensusTime:
			newPosition = weight > avLateConsensusPCT
		default:
			newPosition = weight > avStuckConsensusPCT
		}
	} else {
		// don't let us outweigh a proposing node, just recognize consensus
		weight = -1
		newPosition = dtx.yays > dtx.nays
	}

	if newPosition == dtx.ourVote {
		log.Println("No change (", dtx.ourVote, ") : weight ", weight, ", percent ", percentTime)
		return false
	}

	dtx.ourVote = newPosition
	log.Println("We now vote ", dtx.ourVote, " on ", dtx.tx.ID())
	return true
}
