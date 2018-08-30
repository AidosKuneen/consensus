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

/** Represents a proposed position taken during a round of consensus.

  During consensus, peers seek agreement on a set of transactions to
  apply to the prior ledger to generate the next ledger.  Each peer takes a
  position on whether to include or exclude potential transactions.
  The position on the set of transactions is proposed to its peers as an
  instance of the ConsensusProposal class.

  An instance of ConsensusProposal can be either our own proposal or one of
  our peer's.

  As consensus proceeds, peers may change their position on the transaction,
  or choose to abstain. Each successive proposal includes a strictly
  monotonically increasing number (or, if a peer is choosing to abstain,
  the special value `seqLeave`).

  Refer to @ref Consensus for requirements of the template arguments.

  @tparam NodeID_t Type used to uniquely identify nodes/peers
  @tparam LedgerID_t Type used to uniquely identify ledgers
  @tparam Position_t Type used to represent the position taken on transactions
                     under consideration during this round of consensus
*/

const (
	//< Sequence value when a peer initially joins consensus
	seqJoin = 0
	//< Sequence number when  a peer wants to bow out and leave consensus
	seqLeave = 0xffffffff
)

type consensusProposal struct {
	//! Unique identifier of prior ledger this proposal is based on
	previousLedger LedgerID

	//! Unique identifier of the position this proposal is taking
	position TxSetID

	//! The ledger close time this position is taking
	closeTime time.Time

	// !The time this position was last updated
	seenTime time.Time

	//! The sequence number of these positions taken by this node
	proposeSeq Seq

	//! The identifier of the node taking this position
	nodeID NodeID
}

/** Constructor

  @param prevLedger The previous ledger this proposal is building on.
  @param seq The sequence number of this proposal.
  @param position The position taken on transactions in this round.
  @param closeTime Position of when this ledger closed.
  @param now Time when the proposal was taken.
  @param nodeID ID of node/peer taking this position.
*/
func newConsensusProposal(prev LedgerID, seq Seq, positon TxSetID, closeTime, now time.Time, nodeID NodeID) *consensusProposal {
	return &consensusProposal{
		previousLedger: prev,
		position:       positon,
		closeTime:      closeTime,
		seenTime:       now,
		proposeSeq:     seq,
		nodeID:         nodeID,
	}
}

/** Whether this is the first position taken during the current
  consensus round.
*/
func (p *consensusProposal) isInitial() bool {
	return p.proposeSeq == seqJoin
}

//! Get whether this node left the consensus process

func (p *consensusProposal) isBowOut() bool {
	return p.proposeSeq == seqLeave
}

//! Get whether this position is stale relative to the provided cutoff

func (p *consensusProposal) isStale(cutoff time.Time) bool {
	return p.seenTime.Before(cutoff)
}

/** Update the position during the consensus process. This will increment
  the proposal's sequence number if it has not already bowed out.

  @param newPosition The new position taken.
  @param newCloseTime The new close time.
  @param now the time The new position was taken
*/
func (p *consensusProposal) changePosition(
	newPosition TxSetID, newCloseTime time.Time, now time.Time) {
	p.position = newPosition
	p.closeTime = newCloseTime
	p.seenTime = now
	if p.proposeSeq != seqLeave {
		p.proposeSeq++
	}
}

/** Leave consensus
  Update position to indicate the node left consensus.
  @param now Time when this node left consensus.
*/
func (p *consensusProposal) bowOut(now time.Time) {
	p.seenTime = now
	p.proposeSeq = seqLeave
}
func (p *consensusProposal) equals(p2 *consensusProposal) bool {
	return p.nodeID == p2.nodeID && p.proposeSeq == p2.proposeSeq &&
		p.previousLedger == p2.previousLedger && p.position == p2.position &&
		p.closeTime == p2.closeTime && p.seenTime == p2.seenTime
}
