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

type unixTime int64

//NodeID is a id for a node.
type NodeID [32]byte

//TxID is a id for a tx.
type TxID [32]byte

//TxSetID is a id for a txset.
type TxSetID [32]byte

//LedgerID is a id for a ledger.
type LedgerID [32]byte

//ValidationID is a id for a ledger.
type ValidationID [32]byte

//ProposalID is a id for a Proposal.
type ProposalID [32]byte

//Seq is a sequence no.
type Seq uint64

// TxT is a single transaction
type TxT interface {
	ID() TxID
}

//The ValidationAdaptor template implements a set of helper functions that
//plug the consensus algorithm into a specific application.  It also identifies
//the types that play important roles in Consensus (transactions, ledgers, ...).
//Normally you should use Peer and PeerInterface instead.
type ValidationAdaptor interface {
	// Attempt to acquire a specific ledger.
	AcquireLedger(LedgerID) (*Ledger, error)

	// Handle a newly stale validation, this should do minimal work since
	// it is called by Validations while it may be iterating Validations
	// under lock
	OnStale(*Validation)

	// Flush the remaining validations (typically done on shutdown)
	Flush(remaining map[NodeID]*Validation)

	// Return the current network time (used to determine staleness)
	Now() time.Time
}

//The Adaptor template implements a set of helper functions that
//plug the consensus algorithm into a specific application.  It also identifies
//the types that play important roles in Consensus (transactions, ledgers, ...).
//Normally you should use Peer and PeerInterface instead.
type Adaptor interface {
	//-----------------------------------------------------------------------
	//
	// Attempt to acquire a specific ledger.
	AcquireLedger(LedgerID) (*Ledger, error)

	// Acquire the transaction set associated with a proposed position.
	AcquireTxSet(TxSetID) (TxSet, error)

	// Whether any transactions are in the open ledger
	HasOpenTransactions() bool

	// Number of proposers that have validated the given ledger
	ProposersValidated(LedgerID) uint

	// Number of proposers that have validated a ledger descended from the
	// given ledger; if prevLedger.id() != prevLedgerID, use prevLedgerID
	// for the determination
	ProposersFinished(*Ledger, LedgerID) uint

	// Return the ID of the last closed (and validated) ledger that the
	// application thinks consensus should use as the prior ledger.
	GetPrevLedger(LedgerID, *Ledger, Mode) LedgerID

	// Called whenever consensus operating mode changes
	OnModeChange(Mode, Mode)

	// Called when ledger closes
	OnClose(*Ledger, time.Time, Mode) *Result

	// Called when ledger is accepted by consensus
	OnAccept(*Result, *Ledger, time.Duration, *CloseTimes, Mode)

	// Propose the position to peers.
	Propose(*Proposal)

	// Share a received peer proposal with other peer's.
	SharePosition(*Proposal)

	// Share a disputed transaction with peers
	ShareTx(TxT)

	// Share given transaction set with peers
	ShareTxset(TxSet)

	//ShouldAccept returns false if we should wait accept
	ShouldAccept(*Result) bool

	//UpdateOurProposal updates our proposal from otherss proposals
	UpdateOurProposal(props map[NodeID]*Proposal, ourSet, newSet TxSet) TxSet
}

type clock interface {
	Now() time.Time
}
