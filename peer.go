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

// This is partially from https://github.com/ripple/rippled/src/test/csf
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
	"context"
	"encoding/hex"
	"log"
	"math"
	"sync"
	"time"
)

//PeerInterface is the funcs for intererctint Peer struct.
//don't call funcs in peer struct in these callbacks or will cause deadlock.
type PeerInterface interface {
	// Attempt to acquire a specific ledger.
	AcquireLedger(LedgerID) (*Ledger, error)

	// Handle a newly stale validation, this should do minimal work since
	// it is called by Validations while it may be iterating Validations
	// under lock
	OnStale(*Validation)

	// Flush the remaining validations (typically done on shutdown)
	Flush(remaining map[NodeID]*Validation)

	// Acquire the transaction set associated with a proposed position.
	AcquireTxSet(TxSetID) ([]TxT, error)

	// Whether any transactions are in the open ledger
	HasOpenTransactions() bool

	// Called whenever consensus operating mode changes
	OnModeChange(Mode, Mode)

	// Called when ledger closes
	OnClose(*Ledger, time.Time, Mode) TxSet

	// Called when ledger is accepted by consensus
	OnAccept(*Ledger)

	// Propose the position to Peers.
	Propose(*Proposal)

	// Share a received Peer proposal with other Peer's.
	SharePosition(*Proposal)

	// Share a disputed transaction with Peers
	ShareTx(TxT)

	// Share given transaction set with Peers
	ShareTxset(TxSet)

	// Share my validation
	ShareValidaton(*Validation)

	ShouldAccept(*Result) bool
}

type realclock struct{}

func (c realclock) Now() time.Time {
	return time.Now()
}

type valAdaptor struct {
	p *Peer
}

func (a *valAdaptor) AcquireLedger(id LedgerID) (*Ledger, error) {
	//ValidationAdaptor
	return a.p.AcquireLedger(id)
}

// Handle a newly stale validation, this should do minimal work since
// it is called by Validations while it may be iterating Validations
// under lock
func (a *valAdaptor) OnStale(v *Validation) {
	//ValidationAdaptor
	a.p.adaptor.OnStale(v)
}

// Flush the remaining validations (typically done on shutdown)
func (a *valAdaptor) Flush(remaining map[NodeID]*Validation) {
	//ValidationAdaptor
	a.p.adaptor.Flush(remaining)
}

// Return the current network time (used to determine staleness)
func (a *valAdaptor) Now() time.Time {
	//ValidationAdaptor
	return time.Now()
}

//Peer is the API interface for consensus.
type Peer struct {
	sync.RWMutex
	//! Generic consensus
	consensus *Consensus

	//! Our unique ID
	id NodeID

	unl []NodeID

	//! The last ledger closed by this node
	lastClosedLedger *Ledger

	//! Validations from trusted nodes
	validations *Validations

	//! The most recent ledger that has been fully validated by the network from
	//! the perspective of this Peer
	fullyValidatedLedger *Ledger

	//-------------------------------------------------------------------------
	// Store most network messages; these could be purged if memory use ever
	// becomes problematic

	//! Map from Ledger::ID to vector of Positions with that ledger
	//! as the prior ledger
	PeerPositions map[LedgerID][]*Proposal

	//! Enforce invariants on validation sequence numbers
	seqEnforcer SeqEnforcer

	adaptor PeerInterface

	//! Whether to simulate running as validator or a tracking node
	runAsValidator bool

	lastValidation *Validation
	lastProposal   *Proposal
}

func param() {
	ledgerIdleInterval = 1 * time.Hour
	ledgerPrevInterval = 1 * time.Hour
}

//NewPeer returns a peer object.
func NewPeer(adaptor PeerInterface, i NodeID, unl []NodeID, runAsValidator bool, lastLedger *Ledger) *Peer {
	p := &Peer{
		adaptor:              adaptor,
		id:                   i,
		unl:                  unl,
		lastClosedLedger:     lastLedger,
		fullyValidatedLedger: Genesis,
		PeerPositions:        make(map[LedgerID][]*Proposal),
		runAsValidator:       runAsValidator,
	}
	param()
	c := &realclock{}
	p.consensus = NewConsensus(c, p)
	p.validations = NewValidations(&valAdaptor{
		p: p,
	}, c)
	return p
}

//AcquireLedger gets the ledger whose ID is ledgerID.
func (p *Peer) AcquireLedger(ledgerID LedgerID) (*Ledger, error) {
	//Adaptor
	return p.adaptor.AcquireLedger(ledgerID)
}

//ShouldAccept returns true if the result should be accepted.
func (p *Peer) ShouldAccept(result *Result) bool {
	//Adaptor
	return p.adaptor.ShouldAccept(result)
}

// AcquireTxSet Attempt to acquire the TxSet associated with the given ID
func (p *Peer) AcquireTxSet(setID TxSetID) (TxSet, error) {
	//Adaptor
	ts, err := p.adaptor.AcquireTxSet(setID)
	if err != nil {
		return nil, err
	}
	txset := make(TxSet)
	for _, t := range ts {
		txset[t.ID()] = t
	}
	return txset, nil
}

//HasOpenTransactions returns true if having txs that should be approved.
func (p *Peer) HasOpenTransactions() bool {
	//Adaptor
	return p.adaptor.HasOpenTransactions()
}

// ProposersValidated is the number of proposers that have validated the given ledger
func (p *Peer) ProposersValidated(prevLedger LedgerID) uint {
	//Adaptor
	//This funcs is used for determing if the node closes right now.
	//But it causes consensus too fast, so for now we prevent radically faster consensus.
	return 0
	// return p.validations.NumTrustedForLedger(prevLedger)
}

// ProposersFinished is the number of proposers that have validated a ledger descended from the
// given ledger; if prevLedger.id() != prevLedgerID, use prevLedgerID
// for the determination
func (p *Peer) ProposersFinished(prevLedger *Ledger, prevLedgerID LedgerID) uint {
	//Adaptor
	return uint(p.validations.GetNodesAfter(prevLedger, prevLedgerID))
}

// GetPrevLedger returns the ID of the last closed (and validated) ledger that the
// application thinks consensus should use as the prior ledger.
func (p *Peer) GetPrevLedger(ledgerID LedgerID, ledger *Ledger, mode Mode) LedgerID {
	//Adaptor
	// only do if we are past the genesis ledger
	if ledger.Seq == 0 {
		return ledgerID
	}

	netLgr := p.validations.GetPreferred2(ledger, p.earliestAllowedSeq())
	if netLgr != ledgerID {
		log.Println("Now we are having wrong latest ledger", hex.EncodeToString(ledgerID[:])[:4])
		log.Println("Most prefferd is", hex.EncodeToString(netLgr[:])[:4])
	}

	return netLgr
}

// Propose the position to peers.
func (p *Peer) Propose(pos *Proposal) {
	//Adaptor
	if pos.NodeID == p.id {
		p.lastProposal = pos
	}
	p.adaptor.Propose(pos)
}

// OnModeChange is Called whenever consensus operating mode changes
func (p *Peer) OnModeChange(from, to Mode) {
	//Adaptor
	p.adaptor.OnModeChange(from, to)
}

// SharePosition share a received peer proposal with other peer's.
func (p *Peer) SharePosition(pos *Proposal) {
	//Adaptor
	p.adaptor.SharePosition(pos)
}

// ShareTx shares a received peer proposal with other peer's.
func (p *Peer) ShareTx(tx TxT) {
	//Adaptor
	p.adaptor.ShareTx(tx)
}

// ShareTxset shares given transaction set with peers
func (p *Peer) ShareTxset(ts TxSet) {
	//Adaptor
	p.adaptor.ShareTxset(ts)
}

// PrevLedgerID gets the previous ledger ID.
// The previous ledger is the last ledger seen by the consensus code and should correspond to the most recent validated ledger seen by this peer.
//   @return ID of previous ledger
func (p *Peer) PrevLedgerID() LedgerID {
	p.RLock()
	defer p.RUnlock()
	return p.consensus.PrevLedgerID()
}

//AddProposal adds a proposal
func (p *Peer) AddProposal(prop *Proposal) {
	p.Lock()
	defer p.Unlock()
	p.addProposal(prop)
}

func (p *Peer) addProposal(prop *Proposal) {
	if prop.NodeID == p.id {
		return
	}
	ok := false
	for _, u := range p.unl {
		if u == prop.NodeID {
			ok = true
		}
	}
	if !ok {
		return
	}

	// TODO: This always suppresses relay of peer positions already seen
	// Should it allow forwarding if for a recent ledger ?
	dest := p.PeerPositions[prop.PreviousLedger]
	for _, d := range dest {
		if d.ID() == prop.ID() {
			return
		}
	}
	p.PeerPositions[prop.PreviousLedger] = append(p.PeerPositions[prop.PreviousLedger], prop)
	p.consensus.PeerProposal(time.Now(), prop)
}

func (p *Peer) earliestAllowedSeq() Seq {
	return p.fullyValidatedLedger.Seq
}

var resendValidationWaitTime = time.Minute

//Start starts the consensus.
func (p *Peer) Start(ctx context.Context) {
	p.startRound()
	p.validations.Expire()
	go func() {
		ctx2, cancel2 := context.WithCancel(ctx)
		defer cancel2()
		for {
			select {
			case <-ctx2.Done():
				return
			case <-time.After(LedgerGranularity):
				p.Lock()
				p.consensus.TimerEntry(time.Now())
				p.Unlock()
			}
		}
	}()
	go func() {
		ctx2, cancel2 := context.WithCancel(ctx)
		defer cancel2()
		for {
			select {
			case <-ctx2.Done():
				return
			case <-time.After(resendValidationWaitTime):
				p.RLock()
				if p.lastValidation != nil {
					p.adaptor.ShareValidaton(p.lastValidation)
				}
				if p.lastProposal != nil {
					p.adaptor.SharePosition(p.lastProposal)
				}
				p.RUnlock()
			}
		}
	}()
}

//AddValidation adds a trusted validation and return true if it is worth forwarding
func (p *Peer) AddValidation(v *Validation) bool {
	p.Lock()
	defer p.Unlock()
	return p.addValidation(v)
}

func (p *Peer) addValidation(v *Validation) bool {
	v.Trusted = true
	v.SeenTime = time.Now()
	res := p.validations.Add(v.NodeID, v)
	log.Println("validatoin status:", res)

	if res == VstatStale {
		return false
	}

	// Acquire will try to get from network if not already local
	if lgr, err := p.AcquireLedger(v.LedgerID); err == nil {
		p.checkFullyValidated(lgr)
	}
	return true
}

func (p *Peer) startRound() {
	// Between rounds, we take the majority ledger
	// In the future, consider taking Peer dominant ledger if no validations
	// yet
	bestLCL :=
		p.validations.GetPreferred2(p.lastClosedLedger, p.earliestAllowedSeq())
	if bestLCL == Genesis.ID() {
		bestLCL = p.lastClosedLedger.ID()
	}
	pid := p.lastClosedLedger.ID()
	log.Println("starting a round", "best ledger", hex.EncodeToString(bestLCL[:])[:4], "prevLedger", hex.EncodeToString(pid[:])[:4])
	// Not yet modeling dynamic UNL.
	nowUntrusted := make(map[NodeID]struct{})
	p.consensus.StartRound(
		time.Now(), bestLCL, p.lastClosedLedger, nowUntrusted, p.runAsValidator)
}

/** Check if a new ledger can be deemed fully validated */
func (p *Peer) checkFullyValidated(ledger *Ledger) {
	// Only consider ledgers newer than our last fully validated ledger
	if ledger.Seq <= p.fullyValidatedLedger.Seq {
		return
	}
	count := p.validations.NumTrustedForLedger(ledger.ID())
	quorum := int(math.Ceil(float64(len(p.unl)) * 0.8))
	if count >= uint(quorum) && ledger.IsAncestor(p.fullyValidatedLedger) {
		p.fullyValidatedLedger = ledger
	}
}

// OnClose is Called when ledger closes
func (p *Peer) OnClose(prevLedger *Ledger, closeTime time.Time, mode Mode) *Result {
	txns := p.adaptor.OnClose(prevLedger, closeTime, mode)
	id := txns.ID()
	pid := prevLedger.ID()
	log.Println("closing prevledger", hex.EncodeToString(pid[:])[:4], "txnsid", hex.EncodeToString(id[:])[:4], "time", closeTime)
	return NewResult(txns,
		&Proposal{
			PreviousLedger: prevLedger.ID(),
			Position:       txns.ID(),
			CloseTime:      closeTime,
			Time:           time.Now(),
			NodeID:         p.id,
			ProposeSeq:     0,
		})
}

//IndexOfFunc returns the IndexOf func for the ledger l.
func IndexOfFunc(l *Ledger, acquireLedger func(LedgerID) (*Ledger, error)) func(s Seq) LedgerID {
	return func(s Seq) LedgerID {
		if s > l.Seq {
			panic("not found")
		}
		if s == 0 {
			return Genesis.ID()
		}
		var ll *Ledger
		for ll = l; ll.Seq != s && ll.Seq > 0; {
			var err error
			ll, err = acquireLedger(ll.ParentID)
			if err != nil {
				panic(err)
			}
		}
		return ll.ID()
	}
}

// OnAccept is called when ledger is accepted by consensus
func (p *Peer) OnAccept(result *Result, prevLedger *Ledger,
	closeResolution time.Duration, rawCloseTime *CloseTimes, mode Mode) {
	newLedger := &Ledger{
		ParentID:            prevLedger.ID(),
		Seq:                 prevLedger.Seq + 1,
		Txs:                 result.Txns.Clone(),
		CloseTimeResolution: closeResolution,
		ParentCloseTime:     prevLedger.CloseTime,
		CloseTimeAgree:      !result.Position.CloseTime.IsZero(),
	}
	newLedger.IndexOf = IndexOfFunc(newLedger, p.adaptor.AcquireLedger)
	if newLedger.CloseTimeAgree {
		newLedger.CloseTime = EffCloseTime(result.Position.CloseTime, closeResolution, prevLedger.CloseTime)
	} else {
		newLedger.CloseTime = prevLedger.CloseTime.Add(time.Second)
	}
	p.checkFullyValidated(newLedger)
	p.adaptor.OnAccept(newLedger)

	nid := newLedger.ID()
	pid := prevLedger.ID()
	log.Println("onaccept txset", hex.EncodeToString(result.Position.Position[:])[:4], "prev ledger", hex.EncodeToString(pid[:])[:4],
		"closetime", rawCloseTime.Self, "new ledger", hex.EncodeToString(nid[:])[:4], "seq", newLedger.Seq)
	log.Println("proposers", result.Proposers, "round time", result.RoundTime.Dur)
	p.PeerPositions = make(map[LedgerID][]*Proposal)
	p.lastClosedLedger = newLedger

	// Only send validation if the new ledger is compatible with our
	// fully validated ledger
	isCompatible := newLedger.IsAncestor(p.fullyValidatedLedger)

	// Can only send one validated ledger per seq
	consensusFail := result.State == StateMovedOn
	if p.runAsValidator && isCompatible && !consensusFail {
		enf := p.seqEnforcer.Try(time.Now(), newLedger.Seq)
		if enf {
			isFull := mode == ModeProposing

			v := Validation{
				LedgerID: newLedger.ID(),
				Seq:      newLedger.Seq,
				SignTime: time.Now(),
				SeenTime: time.Now(),
				NodeID:   p.id,
				Full:     isFull,
			}
			// share the new validation; it is trusted by the receiver
			p.adaptor.ShareValidaton(&v)
			// we trust ourselves
			p.addValidation(&v)
			p.lastValidation = &v
			p.lastProposal = nil
		}
	}
	// kick off the next round...
	// in the actual implementation, this passes back through
	// network ops
	// startRound sets the LCL state, so we need to call it once after
	// the last requested round completes
	p.startRound()
}
