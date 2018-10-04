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

// This is a rewrite of https://github.com/ripple/rippled/src/test/csf
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

package sim

import (
	"errors"
	"log"
	"math"
	"time"

	"github.com/AidosKuneen/consensus"
)

type processingDelays struct {
	ledgerAccept   time.Duration
	recvValidation time.Duration
}

func (d *processingDelays) onReceive(m interface{}) time.Duration {
	switch m.(type) {
	case *consensus.Validation:
		return d.recvValidation
	case consensus.Validation:
		return d.recvValidation
	default:
		return 0
	}
}

type valAdaptor struct {
	p *peer
}

func (a *valAdaptor) AcquireLedger(id consensus.LedgerID) (*consensus.Ledger, error) {
	return a.p.AcquireLedger(id)
}

// Handle a newly stale validation, this should do minimal work since
// it is called by Validations while it may be iterating Validations
// under lock
func (a *valAdaptor) OnStale(*consensus.Validation) {

}

// Flush the remaining validations (typically done on shutdown)
func (a *valAdaptor) Flush(remaining map[consensus.NodeID]*consensus.Validation) {

}

// Return the current network time (used to determine staleness)
func (a *valAdaptor) Now() time.Time {
	return a.p.now()
}

type peer struct {

	//! Generic consensus
	consensus *consensus.Consensus

	//! Our unique ID
	id consensus.NodeID

	//! Current signing key
	// key peerKey

	//! The oracle that manages unique ledgers
	oracle *ledgerOracle

	//! Scheduler of events
	scheduler *scheduler

	//! Handle to network for sending messages
	net *basicNetwork

	//! Handle to Trust graph of network
	trustGraph *trustGraph

	//! openTxs that haven't been closed in a ledger yet
	openTxs consensus.TxSet

	//! The last ledger closed by this node
	lastClosedLedger *consensus.Ledger

	//! Ledgers this node has closed or loaded from the network
	ledgers map[consensus.LedgerID]*consensus.Ledger

	//! Validations from trusted nodes
	validations *consensus.Validations

	//! The most recent ledger that has been fully validated by the network from
	//! the perspective of this Peer
	fullyValidatedLedger *consensus.Ledger

	//-------------------------------------------------------------------------
	// Store most network messages; these could be purged if memory use ever
	// becomes problematic

	//! Map from Ledger::ID to vector of Positions with that ledger
	//! as the prior ledger
	peerPositions map[consensus.LedgerID][]*consensus.Proposal
	//! TxSet associated with a TxSet::ID
	txSets map[consensus.TxSetID]consensus.TxSet

	// Ledgers/TxSets we are acquiring and when that request times out
	acquiringLedgers map[consensus.LedgerID]time.Time
	acquiringTxSets  map[consensus.TxSetID]time.Time

	//! The number of ledgers this peer has completed
	completedLedgers int

	//! The number of ledgers this peer should complete before stopping to run
	targetLedgers int // std::numeric_limits<int>::max();

	//! Skew of time relative to the common scheduler clock
	clockSkew time.Duration

	//! Simulated delays to use for internal processing
	processingDelays processingDelays

	//! Whether to simulate running as validator or a tracking node
	runAsValidator bool

	//! Enforce invariants on validation sequence numbers
	seqEnforcer consensus.SeqEnforcer

	//TODO: Consider removing these two, they are only a convenience for tests
	// Number of proposers in the prior round
	prevProposers int
	// Duration of prior round
	prevRoundTime consensus.Timer

	// Quorum of validations needed for a ledger to be fully validated
	// TODO: Use the logic in ValidatorList to set this dynamically
	quorum int

	//! The collectors to report events to
	collectors collectorRef

	router *routerT

	//-------------------------------------------------------------------------
	// Injects a specific transaction when generating the ledger following
	// the provided sequence.  This allows simulating a byzantine failure in
	// which a node generates the wrong ledger, even when consensus worked
	// properly.
	// TODO: Make this more robust
	txInjections map[consensus.Seq]*tx
}

func newPeer(i consensus.NodeID, s *scheduler, o *ledgerOracle, n *basicNetwork,
	c collectorRef, tg *trustGraph) *peer {
	p := &peer{
		id:        i,
		oracle:    o,
		scheduler: s,
		// key: peerKey{
		// 	i: 0,
		// },
		net:                  n,
		trustGraph:           tg,
		lastClosedLedger:     consensus.Genesis,
		fullyValidatedLedger: consensus.Genesis,
		collectors:           c,
		ledgers:              make(map[consensus.LedgerID]*consensus.Ledger),
		txInjections:         make(map[consensus.Seq]*tx),
		acquiringLedgers:     make(map[consensus.LedgerID]time.Time),
		acquiringTxSets:      make(map[consensus.TxSetID]time.Time),
		peerPositions:        make(map[consensus.LedgerID][]*consensus.Proposal),
		txSets:               make(map[consensus.TxSetID]consensus.TxSet),
		openTxs:              make(consensus.TxSet),
		router: &routerT{
			nextSeq:         1,
			lastObservedSeq: make(map[consensus.NodeID]map[uint]struct{}),
		},
		runAsValidator: true,
	}
	p.consensus = consensus.NewConsensus(s.clock, p)
	p.validations = consensus.NewValidations(&valAdaptor{
		p: p,
	}, p.scheduler.clock)
	p.ledgers[consensus.GenesisID] = p.lastClosedLedger
	tg.trust(p, p)
	return p
}

func (p *peer) schedule(when time.Duration, what func()) {
	if when == 0 {
		what()
	} else {
		p.scheduler.in(when, what)
	}
}

func (p *peer) issue(ev interface{}) {
	// Use the scheduler time and not the peer's (skewed) local time
	p.collectors.on(p.id, p.scheduler.clock.Now(), ev)
}

func (p *peer) trusts(oID consensus.NodeID) bool {
	for _, p := range p.trustGraph.trustedPeers(p) {
		if p.id == oID {
			return true
		}
	}
	return false
}
func (p *peer) AcquireLedger(ledgerID consensus.LedgerID) (*consensus.Ledger, error) {
	it, ok := p.ledgers[ledgerID]
	if ok {
		return it.Clone(), nil
	}
	// No peers
	if len(p.net.link(p)) == 0 {
		return nil, errors.New("not found")
	}
	// Don't retry if we already are acquiring it and haven't timed out
	aIt, ok := p.acquiringLedgers[ledgerID]
	if ok {
		if p.scheduler.clock.Now().Before(aIt) {
			return nil, errors.New("not found")
		}
	}

	minDuration := 10 * time.Second
	for _, link := range p.net.link(p) {
		if minDuration > link.data.delay {
			minDuration = link.data.delay
		}
		target := link.target
		// Send a messsage to neighbors to find the ledger
		p.net.send(
			p, target, func() {
				it, ok := target.ledgers[ledgerID]
				if ok {
					// if the ledger is found, send it back to the original
					// requesting peer where it is added to the available
					// ledgers
					target.net.send(target, p, func() {
						delete(p.acquiringLedgers, it.ID())
						p.ledgers[it.ID()] = it
					})
				}
			})
	}
	p.acquiringLedgers[ledgerID] = p.scheduler.clock.Now().Add(2 * minDuration)
	return nil, errors.New("not found")
}

// Attempt to acquire the TxSet associated with the given ID
func (p *peer) AcquireTxSet(setID consensus.TxSetID) (consensus.TxSet, error) {
	it, ok := p.txSets[setID]
	if ok {
		return it, nil
	}
	// No peers
	if len(p.net.link(p)) == 0 {
		return nil, errors.New("not found")
	}
	// Don't retry if we already are acquiring it and haven't timed out
	aIt, ok := p.acquiringTxSets[setID]
	if ok {
		if p.scheduler.clock.Now().Before(aIt) {
			return nil, errors.New("not found")
		}
	}

	minDuration := 10 * time.Second
	for _, link := range p.net.link(p) {
		if minDuration > link.data.delay {
			minDuration = link.data.delay
		}
		target := link.target
		// Send a messsage to neighbors to find the ledger
		p.net.send(
			p, target, func() {
				it, ok := target.txSets[setID]
				if ok {
					// if the ledger is found, send it back to the original
					// requesting peer where it is added to the available
					// ledgers
					target.net.send(target, p, func() {
						delete(p.acquiringTxSets, it.ID())
						p.handle(it)
					})
				}
			})
	}
	p.acquiringTxSets[setID] = p.scheduler.clock.Now().Add(2 * minDuration)
	return nil, errors.New("not found")
}
func (p *peer) HasOpenTransactions() bool {
	return len(p.openTxs) > 0
}

func (p *peer) ProposersValidated(prevLedger consensus.LedgerID) uint {
	return p.validations.NumTrustedForLedger(prevLedger)
}
func (p *peer) ProposersFinished(prevLedger *consensus.Ledger, prevLedgerID consensus.LedgerID) uint {
	return uint(p.validations.GetNodesAfter(prevLedger, prevLedgerID))
}

func (p *peer) OnClose(prevLedger *consensus.Ledger, closeTime time.Time,
	mode consensus.Mode) *consensus.Result {
	p.issue(&closeLedger{
		prevLedger: prevLedger.Clone(),
		txSetType:  p.openTxs.Clone(),
	})
	txns := p.openTxs.Clone()
	id := txns.ID()
	log.Println("closed #txs", len(p.openTxs), "pid", p.id[0], "txnsid", id[:2], "time", closeTime)
	return consensus.NewResult(txns,
		&consensus.Proposal{
			PreviousLedger: prevLedger.ID(),
			Position:       txns.ID(),
			CloseTime:      closeTime,
			Time:           p.now(),
			NodeID:         p.id,
			ProposeSeq:     0,
		})
}

func (p *peer) OnAccept(result *consensus.Result, prevLedger *consensus.Ledger,
	closeResolution time.Duration, rawCloseTime *consensus.CloseTimes,
	mode consensus.Mode) {
	log.Println("onaccept pid", p.id[0], "txset", result.Position.Position[:2], "prev", prevLedger.ID()[0], "closetime", rawCloseTime)
	p.schedule(p.processingDelays.ledgerAccept, func() {
		proposing := mode == consensus.ModeProposing
		consensusFail := result.State == consensus.StateMovedOn

		acceptedTxs := p.injectTxs(prevLedger, result.Txns)
		newLedger := p.oracle.accept(
			prevLedger,
			acceptedTxs,
			closeResolution,
			result.Position.CloseTime)
		p.ledgers[newLedger.ID()] = newLedger.Ledger

		p.issue(&acceptLedger{
			ledger: newLedger.Ledger,
			prior:  p.lastClosedLedger,
		},
		)
		p.prevProposers = int(result.Proposers)
		p.prevRoundTime = result.RoundTime
		p.lastClosedLedger = newLedger.Ledger

		for id := range p.openTxs {
			if _, ok := acceptedTxs[id]; ok {
				delete(p.openTxs, id)
			}
		}

		// Only send validation if the new ledger is compatible with our
		// fully validated ledger
		isCompatible := newLedger.IsAncestor(p.fullyValidatedLedger)

		// Can only send one validated ledger per seq
		if p.runAsValidator && isCompatible && !consensusFail &&
			p.seqEnforcer.Try(p.scheduler.clock.Now(), newLedger.Seq) {
			isFull := proposing

			v := consensus.Validation{
				LedgerID: newLedger.ID(),
				Seq:      newLedger.Seq,
				SignTime: p.now(),
				SeenTime: p.now(),
				// Key:      p.key,
				NodeID: p.id,
				Full:   isFull,
			}
			// share the new validation; it is trusted by the receiver
			p.share(v)
			// we trust ourselves
			p.addTrustedValidation(&v)
		}

		p.checkFullyValidated(newLedger.Ledger)

		// kick off the next round...
		// in the actual implementation, this passes back through
		// network ops
		p.completedLedgers++
		// startRound sets the LCL state, so we need to call it once after
		// the last requested round completes
		if p.completedLedgers <= p.targetLedgers {
			p.startRound()
		}
	})
}

func (p *peer) earliestAllowedSeq() consensus.Seq {
	return p.fullyValidatedLedger.Seq
}

func (p *peer) GetPrevLedger(
	ledgerID consensus.LedgerID,
	ledger *consensus.Ledger,
	mode consensus.Mode) consensus.LedgerID {
	// only do if we are past the genesis ledger
	if ledger.Seq == 0 {
		return ledgerID
	}

	netLgr :=
		p.validations.GetPreferred2(ledger, p.earliestAllowedSeq())

	if netLgr != ledgerID {
		p.issue(&wrongPrevLedger{
			wrong: ledgerID,
			right: netLgr,
		})
	}

	return netLgr
}

func (p *peer) Propose(pos *consensus.Proposal) {
	p.share(pos.Clone())
}

// Not interested in tracking consensus mode changes for now
func (p *peer) OnModeChange(consensus.Mode, consensus.Mode) {
}

// Share a message by broadcasting to all connected peers
func (p *peer) share(m interface{}) {
	p.issue(m)
	p.send(&broadcastMesg{
		mesg:   m,
		seq:    p.router.nextSeq,
		origin: p.id,
	}, p.id)
	p.router.nextSeq++
}

// Unwrap the Position and share the raw proposal
func (p *peer) SharePosition(pos *consensus.Proposal) {
	p.share(pos.Clone())
}

// Unwrap the Position and share the raw proposal
func (p *peer) ShareTx(tx consensus.TxT) {
	p.share(tx)
}
func (p *peer) ShareTxset(ts consensus.TxSet) {
	ts2 := consensus.NewTxSet()
	for k, v := range ts {
		ts2[k] = v
	}
	p.share(ts2)
}

//--------------------------------------------------------------------------
// Validation members

/** Add a trusted validation and return true if it is worth forwarding */
func (p *peer) addTrustedValidation(v *consensus.Validation) bool {
	v.Trusted = true
	v.SeenTime = p.now()
	res := p.validations.Add(v.NodeID, v)

	if res == consensus.VstatStale {
		return false
	}

	// Acquire will try to get from network if not already local
	if lgr, err := p.AcquireLedger(v.LedgerID); err == nil {
		p.checkFullyValidated(lgr)
	}
	return true
}

/** Check if a new ledger can be deemed fully validated */
func (p *peer) checkFullyValidated(ledger *consensus.Ledger) {
	// Only consider ledgers newer than our last fully validated ledger
	if ledger.Seq <= p.fullyValidatedLedger.Seq {
		return
	}
	count := p.validations.NumTrustedForLedger(ledger.ID())
	numTrustedPeers := p.trustGraph.graph.outDegree(p)
	p.quorum = int(math.Ceil(float64(numTrustedPeers) * 0.8))
	if count >= uint(p.quorum) && ledger.IsAncestor(p.fullyValidatedLedger) {
		p.issue(&fullyValidateLedger{
			ledger: ledger,
			prior:  p.fullyValidatedLedger,
		})
		p.fullyValidatedLedger = ledger
	}
}

//-------------------------------------------------------------------------
// Peer messaging members

// Basic Sequence number router
//   A message that will be flooded across the network is tagged with a
//   seqeuence number by the origin node in a BroadcastMesg. Receivers will
//   ignore a message as stale if they've already processed a newer sequence
//   number, or will process and potentially relay the message along.
//
//  The various bool handle(MessageType) members do the actual processing
//  and should return true if the message should continue to be sent to peers.
//
//  WARN: This assumes messages are received and processed in the order they
//        are sent, so that a peer receives a message with seq 1 from node 0
//        before seq 2 from node 0, etc.
//  TODO: Break this out into a class and identify type interface to allow
//        alternate routing strategies
type broadcastMesg struct {
	mesg   interface{}
	seq    uint
	origin consensus.NodeID
}

type routerT struct {
	nextSeq         uint //= 1;
	lastObservedSeq map[consensus.NodeID]map[uint]struct{}
}

func (p *peer) send(bm *broadcastMesg, from consensus.NodeID) {
	for _, link := range p.net.link(p) {
		if link.target.id != from && link.target.id != bm.origin {
			// cheat and don't bother sending if we know it has already been
			// used on the other end
			if _, ok := link.target.router.lastObservedSeq[bm.origin][bm.seq]; !ok {
				p.issue(&relay{
					to: link.target.id,
					V:  bm.mesg,
				})
				target := link.target
				p.net.send(
					p, target, func() { //[to = link.target, bm, id = this->id ] {
						target.receive(bm, p.id)
					})
			}
		}
	}
}

// Receive a shared message, process it and consider continuing to relay it
func (p *peer) receive(bm *broadcastMesg, from consensus.NodeID) {
	p.issue(&receive{
		from: from,
		V:    bm.mesg,
	})
	if _, ok := p.router.lastObservedSeq[bm.origin][bm.seq]; !ok {
		if _, ok := p.router.lastObservedSeq[bm.origin]; !ok {
			p.router.lastObservedSeq[bm.origin] = make(map[uint]struct{})
		}
		p.router.lastObservedSeq[bm.origin][bm.seq] = struct{}{}
		p.schedule(p.processingDelays.onReceive(bm.mesg), func() { //, [this, bm, from]
			if p.handle(bm.mesg) {
				p.send(bm, from)
			}
		})
	}
}
func (p *peer) handle(vv interface{}) bool {
	switch v := vv.(type) {
	case *consensus.Proposal:
		log.Println("peer", p.id[0], "handling a proposal from nodeid", v.NodeID[:2], "pseq", v.ProposeSeq)

		// Only relay untrusted proposals on the same ledger
		if !p.trusts(v.NodeID) {
			return v.PreviousLedger == p.lastClosedLedger.ID()
		}
		// TODO: This always suppresses relay of peer positions already seen
		// Should it allow forwarding if for a recent ledger ?
		dest := p.peerPositions[v.PreviousLedger]
		ok := false
		for _, d := range dest {
			if d.ID() == v.ID() {
				ok = true
				break
			}
		}
		if ok {
			return false
		}
		p.peerPositions[v.PreviousLedger] = append(p.peerPositions[v.PreviousLedger], v)

		// Rely on consensus to decide whether to relay
		return p.consensus.PeerProposal(p.now(), v)
	case consensus.TxSet:
		id := v.ID()
		log.Println("peer", p.id[0], "handling a txset", id[:2])
		_, ok := p.txSets[id]
		if !ok {
			p.txSets[id] = v
			p.consensus.GotTxSet(p.now(), v)
		}
		// relay only if new
		return !ok
	case *tx:
		log.Println("peer", p.id[0], "handling tx", v.id[:2])
		// Ignore and suppress relay of transactions already in last ledger
		lastClosedTxs := p.lastClosedLedger.Txs
		if _, ok := lastClosedTxs[v.id]; ok {
			return false
		}
		// only relay if it was new to our open ledger
		_, ok := p.openTxs[v.id]
		p.openTxs[v.id] = v
		return !ok
	case consensus.Validation:
		// TODO: This is not relaying untrusted validations
		if !p.trusts(v.NodeID) {
			return false
		}
		// Will only relay if current
		return p.addTrustedValidation(&v)
	default:
	}
	return false
}

func (p *peer) submit(t *tx) {
	p.issue(&submitTx{
		tx: t,
	})
	if p.handle(t) {
		p.share(t)
	}
}

func (p *peer) timerEntry() {
	p.consensus.TimerEntry(p.now())
	// only reschedule if not completed
	if p.completedLedgers < p.targetLedgers {
		p.scheduler.in(consensus.LedgerGranularity, func() {
			log.Println("entry pid", p.id[0])
			p.timerEntry()
		})
	}
}
func (p *peer) startRound() {
	// Between rounds, we take the majority ledger
	// In the future, consider taking peer dominant ledger if no validations
	// yet
	bestLCL :=
		p.validations.GetPreferred2(p.lastClosedLedger, p.earliestAllowedSeq())
	if bestLCL == consensus.GenesisID {
		bestLCL = p.lastClosedLedger.ID()
	}
	p.issue(&startRound{
		bestLedger: bestLCL,
		prevLedger: p.lastClosedLedger,
	})

	// Not yet modeling dynamic UNL.
	nowUntrusted := make(map[consensus.NodeID]struct{})
	log.Println("peer", p.id, "start", bestLCL)

	p.consensus.StartRound(
		p.now(), bestLCL, p.lastClosedLedger, nowUntrusted, p.runAsValidator)
}

// Start the consensus process assuming it is not yet running
// This runs forever unless targetLedgers is specified
func (p *peer) start() {
	// TODO: Expire validations less frequently?
	p.validations.Expire()
	p.scheduler.in(consensus.LedgerGranularity, func() {
		p.timerEntry()
	})
	p.startRound()
}

func (p *peer) now() time.Time {
	// We don't care about the actual epochs, but do want the
	// generated NetClock time to be well past its epoch to ensure
	// any subtractions of two NetClock::time_point in the consensus
	// code are positive. (e.g. proposeFRESHNESS)
	return p.scheduler.clock.Now().Add(86400 * time.Second).Add(p.clockSkew)
}
func (p *peer) PrevLedgerID() consensus.LedgerID {
	return p.consensus.PrevLedgerID()
}

/** Inject non-consensus Tx

  Injects a transactionsinto the ledger following prevLedger's sequence
  number.

  @param prevLedger The ledger we are building the new ledger on top of
  @param src The Consensus TxSet
  @return Consensus TxSet with inject transactions added if prevLedger.seq
          matches a previously registered Tx.
*/
func (p *peer) injectTxs(prevLedger *consensus.Ledger, src consensus.TxSet) consensus.TxSet {
	it, ok := p.txInjections[prevLedger.Seq]

	if !ok {
		return src
	}
	res := src.Clone()
	res[it.ID()] = it
	return res
}
