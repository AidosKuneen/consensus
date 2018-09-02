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

var zeroID [32]byte

type unixTime int64

//NodeID is a id for a node.
type NodeID [32]byte

//TxID is a id for a tx.
type TxID [32]byte

//TxSetID is a id for a txset.
type TxSetID [32]byte

//LedgerID is a id for a ledger.
type LedgerID [32]byte

//Seq is a sequence no.
type Seq uint64

type seqLedgerID [8 + 32]byte

// A single transaction
type txT interface {
	ID() TxID
}

// A set of transactions
type txSet interface {
	Exists(TxID) bool
	// Return value should have semantics like Tx const *
	Find(TxID) txT
	ID() TxSetID

	// Return set of transactions that are not common to this set or other
	// boolean indicates which set it was in
	// If true I have the tx, otherwiwse o has it.
	Compare(o txSet) map[TxID]bool
	Insert(txT) bool
	Erase(TxID) bool
}

// Agreed upon state that consensus transactions will modify
type ledger interface {
	// Unique identifier of ledgerr
	ID() LedgerID
	Seq() Seq
	CloseTimeResolution() time.Duration
	CloseAgree() bool
	CloseTime() time.Time
	ParentCloseTime() time.Time
	IndexOf(Seq) LedgerID
}

// Wraps a peer's ConsensusProposal
type peerPosition interface {
	Proposal() *consensusProposal
}

type adaptor interface {
	//-----------------------------------------------------------------------
	//
	// Attempt to acquire a specific ledger.
	AcquireLedger(LedgerID) (ledger, error)

	// Acquire the transaction set associated with a proposed position.
	AcquireTxSet(TxSetID) txSet

	// Whether any transactions are in the open ledger
	HasOpenTransactions() bool

	// Number of proposers that have validated the given ledger
	ProposersValidated(LedgerID) uint

	// Number of proposers that have validated a ledger descended from the
	// given ledger; if prevLedger.id() != prevLedgerID, use prevLedgerID
	// for the determination
	ProposersFinished(ledger, LedgerID) uint

	// Return the ID of the last closed (and validated) ledger that the
	// application thinks consensus should use as the prior ledger.
	GetPrevLedger(LedgerID, ledger, consensusMode) LedgerID

	// Called whenever consensus operating mode changes
	OnModeChange(consensusMode, consensusMode)

	// Called when ledger closes
	OnClose(ledger, time.Time, consensusMode) *consensusResult

	// Called when ledger is accepted by consensus
	OnAccept(*consensusResult, ledger, time.Duration, *consensusCloseTimes, consensusMode)

	// Called when ledger was forcibly accepted by consensus via the simulate
	// function.
	OnForceAccept(*consensusResult, ledger, time.Duration, *consensusCloseTimes, consensusMode)

	// Propose the position to peers.
	Propose(consensusProposal)

	// Share a received peer proposal with other peer's.
	SharePosition(peerPosition)

	// Share a disputed transaction with peers
	ShareTx(txT)

	// Share given transaction set with peers
	ShareTxset(txSet)

	// Handle a newly stale validation, this should do minimal work since
	// it is called by Validations while it may be iterating Validations
	// under lock
	OnStale(validation)

	// Flush the remaining validations (typically done on shutdown)
	Flush(remaining map[NodeID]validation)

	// Return the current network time (used to determine staleness)
	Now() time.Time
}

type validation interface {
	// Ledger ID associated with this validation
	LedgerID() LedgerID

	// Sequence number of validation's ledger (0 means no sequence number)
	Seq() Seq

	// When the validation was signed
	SignTime() time.Time

	// When the validation was first observed by this node
	SeenTime() time.Time

	// Whether the publishing node was Trusted at the time the validation
	// arrived
	Trusted() bool

	// Set the validation as trusted
	SetTrusted()

	// Set the validation as untrusted
	SetUntrusted()

	// Whether this is a Full or partial validation
	Full() bool

	LoadFee() uint32
}
