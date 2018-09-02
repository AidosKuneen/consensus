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
	"errors"
	"log"
	"time"
)

type adaptorT struct {
	staleData *staleData
}

//-----------------------------------------------------------------------
//
// Attempt to acquire a specific ledger.
func (a *adaptorT) AcquireLedger(id LedgerID) (ledger, error) {
	l, ok := ledgers[id]
	if !ok {
		log.Println("not found", id)
		return nil, errors.New("not found")
	}
	log.Println("found")
	return l, nil
}

// Acquire the transaction set associated with a proposed position.
func (a *adaptorT) AcquireTxSet(TxSetID) txSet {
	return nil
}

// Whether any transactions are in the open ledger
func (a *adaptorT) HasOpenTransactions() bool {
	return false
}

// Number of proposers that have validated the given ledger
func (a *adaptorT) ProposersValidated(LedgerID) uint {
	return 0
}

// Number of proposers that have validated a ledger descended from the
// given ledger; if prevLedger.id() != prevLedgerID, use prevLedgerID
// for the determination
func (a *adaptorT) ProposersFinished(ledger, LedgerID) uint {
	return 0
}

// Return the ID of the last closed (and validated) ledger that the
// application thinks consensus should use as the prior ledger.
func (a *adaptorT) GetPrevLedger(LedgerID, ledger, consensusMode) LedgerID {
	return zeroID
}

// Called whenever consensus operating mode changes
func (a *adaptorT) OnModeChange(consensusMode, consensusMode) {
}

// Called when ledger closes
func (a *adaptorT) OnClose(ledger, time.Time, consensusMode) *consensusResult {
	return nil
}

// Called when ledger is accepted by consensus
func (a *adaptorT) OnAccept(*consensusResult, ledger, time.Duration, *consensusCloseTimes, consensusMode) {

}

// Called when ledger was forcibly accepted by consensus via the simulate
// function.
func (a *adaptorT) OnForceAccept(*consensusResult, ledger, time.Duration, *consensusCloseTimes, consensusMode) {

}

// Propose the position to peers.
func (a *adaptorT) Propose(consensusProposal) {

}

// Share a received peer proposal with other peer's.
func (a *adaptorT) SharePosition(peerPosition) {

}

// Share a disputed transaction with peers
func (a *adaptorT) ShareTx(txT) {

}

// Share given transaction set with peers
func (a *adaptorT) ShareTxset(txSet) {

}

// Handle a newly stale validation, this should do minimal work since
// it is called by Validations while it may be iterating Validations
// under lock
func (a *adaptorT) OnStale(v validation) {
	a.staleData.stale = append(a.staleData.stale, v)
}

// Flush the remaining validations (typically done on shutdown)
func (a *adaptorT) Flush(remaining map[NodeID]validation) {
	a.staleData.flushed = remaining
}

// Return the current network time (used to determine staleness)
func (a *adaptorT) Now() time.Time {
	return time.Now()
}
