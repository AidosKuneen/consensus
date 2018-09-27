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
	"crypto/sha256"
	"encoding/binary"
	"time"
)

const (
	//< Sequence value when a peer initially joins consensus
	seqJoin Seq = 0
	//< Sequence number when  a peer wants to bow out and leave consensus
	seqLeave Seq = 0xffffffff
)

//Proposal represents a proposed position taken during a round of consensus.
// During consensus, peers seek agreement on a set of transactions to
// apply to the prior ledger to generate the next ledger.  Each peer takes a
// position on whether to include or exclude potential transactions.
// The position on the set of transactions is proposed to its peers as an
// instance of the ConsensusProposal class.
// An instance of ConsensusProposal can be either our own proposal or one of
// our peer's.
// As consensus proceeds, peers may change their position on the transaction,
// or choose to abstain. Each successive proposal includes a strictly
// monotonically increasing number (or, if a peer is choosing to abstain,
// the special value `seqLeave`).
// Refer to @ref Consensus for requirements of the template arguments.
//   @tparam NodeID_t Type used to uniquely identify nodes/peers
//   @tparam LedgerID_t Type used to uniquely identify ledgers
//   @tparam Position_t Type used to represent the position taken on transactions under consideration during this round of consensus
type Proposal struct {
	//! Unique identifier of prior ledger this proposal is based on
	PreviousLedger LedgerID

	//! Unique identifier of the Position this proposal is taking
	Position TxSetID

	//! The ledger close time this position is taking
	CloseTime time.Time

	// !The time this position was last updated
	Time time.Time

	//! The sequence number of these positions taken by this node
	ProposeSeq Seq

	//! The identifier of the node taking this position
	NodeID NodeID
}

//Clone clones the proposal
func (p *Proposal) Clone() *Proposal {
	p2 := *p
	return &p2
}

/** Whether this is the first position taken during the current
  consensus round.
*/
func (p *Proposal) isInitial() bool {
	return p.ProposeSeq == seqJoin
}

//! Get whether this node left the consensus process

func (p *Proposal) isBowOut() bool {
	return p.ProposeSeq == seqLeave
}

//! Get whether this position is stale relative to the provided cutoff

func (p *Proposal) isStale(cutoff time.Time) bool {
	return p.Time.Before(cutoff) || p.Time.Equal(cutoff)
}

/** Update the position during the consensus process. This will increment
  the proposal's sequence number if it has not already bowed out.

  @param newPosition The new position taken.
  @param newCloseTime The new close time.
  @param now the time The new position was taken
*/
func (p *Proposal) changePosition(
	newPosition TxSetID, newCloseTime time.Time, now time.Time) {
	p.Position = newPosition
	p.CloseTime = newCloseTime
	p.Time = now
	if p.ProposeSeq != seqLeave {
		p.ProposeSeq++
	}
}

/** Leave consensus
  Update position to indicate the node left consensus.
  @param now Time when this node left consensus.
*/
func (p *Proposal) bowOut(now time.Time) {
	p.Time = now
	p.ProposeSeq = seqLeave
}

//ID returns the ID of Proposal v.
func (p *Proposal) ID() ProposalID {
	return sha256.Sum256(p.bytes())
}

func (p *Proposal) bytes() []byte {
	bs := make([]byte, 32+32+4+4+8+32)
	copy(bs, p.PreviousLedger[:])
	copy(bs[32:], p.Position[:])
	binary.LittleEndian.PutUint32(bs[32+32:], uint32(p.CloseTime.Unix()))
	binary.LittleEndian.PutUint32(bs[32+32+4:], uint32(p.Time.Unix()))
	binary.LittleEndian.PutUint64(bs[32+32+4+4:], uint64(p.ProposeSeq))
	copy(bs[32+4+4+8:], p.NodeID[:])
	return bs
}
