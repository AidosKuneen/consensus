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
	"log"
	"time"
)

// Helper class to ensure adaptor is notified whenver the ConsensusMode
// changes
type monitoredMode struct {
	mode consensusMode
}

func (m *monitoredMode) set(mode consensusMode, a adaptor) {
	a.OnModeChange(m.mode, mode)
	m.mode = mode
}

/** Generic implementation of consensus algorithm.

  Achieves consensus on the next ledger.

  Two things need consensus:

    1.  The set of transactions included in the ledger.
    2.  The close time for the ledger.

  The basic flow:

    1. A call to `startRound` places the node in the `Open` phase.  In this
       phase, the node is waiting for transactions to include in its open
       ledger.
    2. Successive calls to `timerEntry` check if the node can close the ledger.
       Once the node `Close`s the open ledger, it transitions to the
       `Establish` phase.  In this phase, the node shares/receives peer
       proposals on which transactions should be accepted in the closed ledger.
    3. During a subsequent call to `timerEntry`, the node determines it has
       reached consensus with its peers on which transactions to include. It
       transitions to the `Accept` phase. In this phase, the node works on
       applying the transactions to the prior ledger to generate a new closed
       ledger. Once the new ledger is completed, the node shares the validated
       ledger with the network, does some book-keeping, then makes a call to
       `startRound` to start the cycle again.

  This class uses a generic interface to allow adapting Consensus for specific
  applications. The Adaptor template implements a set of helper functions that
  plug the consensus algorithm into a specific application.  It also identifies
  the types that play important roles in Consensus (transactions, ledgers, ...).
  The code stubs below outline the interface and type requirements.  The traits
  types must be copy constructible and assignable.

  @warning The generic implementation is not thread safe and the public methods
  are not intended to be run concurrently.  When in a concurrent environment,
  the application is responsible for ensuring thread-safety.  Simply locking
  whenever touching the Consensus instance is one option.

  @code
  // A single transaction
  struct Tx
  {
    // Unique identifier of transaction
    using ID = ...;

    ID id() const;

  };

  // A set of transactions
  struct TxSet
  {
    // Unique ID of TxSet (not of Tx)
    using ID = ...;
    // Type of individual transaction comprising the TxSet
    using Tx = Tx;

    bool exists(Tx::ID const &) const;
    // Return value should have semantics like Tx const *
    Tx const * find(Tx::ID const &) const ;
    ID const & id() const;

    // Return set of transactions that are not common to this set or other
    // boolean indicates which set it was in
    std::map<Tx::ID, bool> compare(TxSet const & other) const;

    // A mutable view of transactions
    struct MutableTxSet
    {
        MutableTxSet(TxSet const &);
        bool insert(Tx const &);
        bool erase(Tx::ID const &);
    };

    // Construct from a mutable view.
    TxSet(MutableTxSet const &);

    // Alternatively, if the TxSet is itself mutable
    // just alias MutableTxSet = TxSet

  };

  // Agreed upon state that consensus transactions will modify
  struct Ledger
  {
    using ID = ...;
    using Seq = ...;

    // Unique identifier of ledgerr
    ID const id() const;
    Seq seq() const;
    auto closeTimeResolution() const;
    auto closeAgree() const;
    auto closeTime() const;
    auto parentCloseTime() const;
    Json::Value getJson() const;
  };

  // Wraps a peer's ConsensusProposal
  struct PeerPosition
  {
    ConsensusProposal<
        std::uint32_t, //NodeID,
        typename Ledger::ID,
        typename TxSet::ID> const &
    proposal() const;

  };


  class Adaptor
  {
  public:
      //-----------------------------------------------------------------------
      // Define consensus types
      using Ledger_t = Ledger;
      using NodeID_t = std::uint32_t;
      using TxSet_t = TxSet;
      using PeerPosition_t = PeerPosition;

      //-----------------------------------------------------------------------
      //
      // Attempt to acquire a specific ledger.
      boost::optional<Ledger> acquireLedger(Ledger::ID const & ledgerID);

      // Acquire the transaction set associated with a proposed position.
      boost::optional<TxSet> acquireTxSet(TxSet::ID const & setID);

      // Whether any transactions are in the open ledger
      bool hasOpenTransactions() const;

      // Number of proposers that have validated the given ledger
      std::size_t proposersValidated(Ledger::ID const & prevLedger) const;

      // Number of proposers that have validated a ledger descended from the
      // given ledger; if prevLedger.id() != prevLedgerID, use prevLedgerID
      // for the determination
      std::size_t proposersFinished(Ledger const & prevLedger,
                                    Ledger::ID const & prevLedger) const;

      // Return the ID of the last closed (and validated) ledger that the
      // application thinks consensus should use as the prior ledger.
      Ledger::ID getPrevLedger(Ledger::ID const & prevLedgerID,
                      Ledger const & prevLedger,
                      Mode mode);

      // Called whenever consensus operating mode changes
      void onModeChange(ConsensusMode before, ConsensusMode after);

      // Called when ledger closes
      Result onClose(Ledger const &, Ledger const & prev, Mode mode);

      // Called when ledger is accepted by consensus
      void onAccept(Result const & result,
        RCLCxLedger const & prevLedger,
        NetClock::duration closeResolution,
        CloseTimes const & rawCloseTimes,
        Mode const & mode);

      // Called when ledger was forcibly accepted by consensus via the simulate
      // function.
      void onForceAccept(Result const & result,
        RCLCxLedger const & prevLedger,
        NetClock::duration closeResolution,
        CloseTimes const & rawCloseTimes,
        Mode const & mode);

      // Propose the position to peers.
      void propose(ConsensusProposal<...> const & pos);

      // Share a received peer proposal with other peer's.
      void share(PeerPosition_t const & prop);

      // Share a disputed transaction with peers
      void share(Txn const & tx);

      // Share given transaction set with peers
      void share(TxSet const &s);

      // Consensus timing parameters and constants
      ConsensusParms const &
      parms() const;
  };
  @endcode

  @tparam Adaptor Defines types and provides helper functions needed to adapt
                  Consensus to the larger application.
*/

type consensus struct {
	timePoint              time.Time
	adaptor                adaptor
	phase                  consensusPhase // accepted
	mode                   monitoredMode  //observing
	firstRound             bool           //= true;
	haveCloseTimeConsensus bool

	// How long the consensus convergence has taken, expressed as
	// a percentage of the time that we expected it to take.
	convergePercent int

	// How long has this round been open
	openTime consensusTimer

	closeResolution time.Duration //= ledgerDefaultTimeResolution;

	// Time it took for the last consensus round to converge
	prevRoundTime time.Duration

	//-------------------------------------------------------------------------
	// Network time measurements of consensus progress

	// The current network adjusted time.  This is the network time the
	// ledger would close if it closed now
	now           time.Time
	prevCloseTime time.Time

	//-------------------------------------------------------------------------
	// Non-peer (self) consensus data

	// Last validated ledger ID provided to consensus
	prevLedgerID LedgerID
	// Last validated ledger seen by consensus
	previousLedger ledger

	// Transaction Sets, indexed by hash of transaction tree
	acquired map[TxSetID]txSet

	result        *consensusResult
	rawCloseTimes *consensusCloseTimes

	//-------------------------------------------------------------------------
	// Peer related consensus data

	// Peer proposed positions for the current round
	currPeerPositions map[NodeID]peerPosition

	// Recently received peer positions, available when transitioning between
	// ledgers or rounds
	recentPeerPositions map[NodeID][]peerPosition

	// The number of proposers who participated in the last consensus round
	prevProposers uint

	// nodes that have bowed out of this consensus process
	deadNodes map[NodeID]struct{}
}

/** Constructor.

  @param clock The clock used to internally sample consensus progress
  @param adaptor The instance of the adaptor class
  @param j The journal to log debug output
*/
func newConsensus(adaptor adaptor) *consensus {
	log.Println("Creating consensus object")
	return &consensus{
		phase: accepted,
		mode: monitoredMode{
			mode: observing,
		},
		firstRound:          true,
		closeResolution:     ledgerDefaultTimeResolution,
		adaptor:             adaptor,
		acquired:            make(map[TxSetID]txSet),
		deadNodes:           make(map[NodeID]struct{}),
		recentPeerPositions: make(map[NodeID][]peerPosition),
		currPeerPositions:   make(map[NodeID]peerPosition),
		rawCloseTimes:       newConsensusCloseTimes(),
	}
}

/** Kick-off the next round of consensus.

  Called by the client code to start each round of consensus.

  @param now The network adjusted time
  @param prevLedgerID the ID of the last ledger
  @param prevLedger The last ledger
  @param nowUntrusted ID of nodes that are newly untrusted this round
  @param proposing Whether we want to send proposals to peers this round.

  @note @b prevLedgerID is not required to the ID of @b prevLedger since
  the ID may be known locally before the contents of the ledger arrive
*/

func (c *consensus) startRound(
	now time.Time,
	prevLedgerID LedgerID,
	prevLedger ledger,
	nowUntrusted map[NodeID]struct{},
	isProposing bool) {
	if c.firstRound {
		// take our initial view of closeTime_ from the seed ledger
		c.prevRoundTime = ledgerIdleInterval
		c.prevCloseTime = prevLedger.CloseTime()
		c.firstRound = false
	} else {
		c.prevCloseTime = c.rawCloseTimes.self
	}

	for n := range nowUntrusted {
		delete(c.recentPeerPositions, n)
	}
	startMode := observing
	if isProposing {
		startMode = proposing
	}
	// We were handed the wrong ledger
	if prevLedger.ID() != prevLedgerID {
		// try to acquire the correct one
		newLedger, err := c.adaptor.AcquireLedger(prevLedgerID)
		if err != nil {
			prevLedger = newLedger
		} else { // Unable to acquire the correct ledger
			startMode = wrongLedger
			log.Println("Entering consensus with: ", c.previousLedger.ID())
			log.Println("Correct LCL is: ", prevLedgerID)
		}
	}
	c.startRoundInternal(now, prevLedgerID, prevLedger, startMode)
}

func (c *consensus) startRoundInternal(
	now time.Time, prevLedgerID LedgerID, prevLedger ledger, mode consensusMode) {
	c.phase = open
	c.mode.set(mode, c.adaptor)
	c.now = now
	c.prevLedgerID = prevLedgerID
	c.previousLedger = prevLedger
	// c.result.reset()
	c.convergePercent = 0
	c.haveCloseTimeConsensus = false
	c.openTime.reset(time.Now())
	c.currPeerPositions = make(map[NodeID]peerPosition)
	c.acquired = make(map[TxSetID]txSet)
	c.rawCloseTimes.peers = make(map[unixTime]int)
	c.rawCloseTimes.self = time.Time{}
	c.deadNodes = make(map[NodeID]struct{})

	c.closeResolution = getNextLedgerTimeResolution(
		c.previousLedger.CloseTimeResolution(),
		c.previousLedger.CloseAgree(),
		c.previousLedger.Seq()+1)
	c.playbackProposals()
	if uint(len(c.currPeerPositions)) > (c.prevProposers / 2) {
		// We may be falling behind, don't wait for the timer
		// consider closing the ledger immediately
		c.timerEntry(c.now)
	}
}

/** A peer has proposed a new position, adjust our tracking.

  @param now The network adjusted time
  @param newProposal The new proposal from a peer
  @return Whether we should do delayed relay of this proposal.
*/

func (c *consensus) peerProposal(
	now time.Time, newPeerPos peerPosition) bool {
	peerID := newPeerPos.Proposal().nodeID

	// Always need to store recent positions
	{
		props := c.recentPeerPositions[peerID]

		if len(props) >= 10 {
			props = props[1:]
		}

		props = append(props, newPeerPos)
	}
	return c.peerProposalInternal(now, newPeerPos)
}

/** Handle a replayed or a new peer proposal.
 */
func (c *consensus) peerProposalInternal(
	now time.Time,
	newPeerPos peerPosition) bool {
	// Nothing to do for now if we are currently working on a ledger
	if c.phase == accepted {
		return false
	}

	c.now = now

	newPeerProp := newPeerPos.Proposal()

	peerID := newPeerProp.nodeID

	if newPeerProp.previousLedger != c.prevLedgerID {
		log.Println("Got proposal for ", newPeerProp.previousLedger,
			" but we are on ", c.prevLedgerID)
		return false
	}

	if _, ok := c.deadNodes[peerID]; ok {
		log.Println("Position from dead node: ", peerID)
		return false
	}

	{
		// update current position
		peerPosIt, ok := c.currPeerPositions[peerID]

		if ok {
			if newPeerProp.proposeSeq <=
				peerPosIt.Proposal().proposeSeq {
				return false
			}
		}

		if newPeerProp.isBowOut() {
			log.Println("Peer bows out: ", peerID)
			if c.result != nil {
				for _, it := range c.result.disputes {
					it.unVote(peerID)
				}
			}
			if ok {
				delete(c.currPeerPositions, peerID)
			}
			c.deadNodes[peerID] = struct{}{}

			return true
		}

		if ok {
			peerPosIt = newPeerPos
		} else {
			c.currPeerPositions[peerID] = newPeerPos
		}
	}

	if newPeerProp.isInitial() {
		// Record the close time estimate
		log.Println("Peer reports close time as ",
			newPeerProp.closeTime.Unix())
		c.rawCloseTimes.peers[unixTime(newPeerProp.closeTime.Unix())]++
	}

	log.Println("Processing peer proposal ", newPeerProp.proposeSeq,
		"/", newPeerProp.position)

	{
		ait, ok := c.acquired[newPeerProp.position]
		if !ok {
			// acquireTxSet will return the set if it is available, or
			// spawn a request for it and return none/nullptr.  It will call
			// gotTxSet once it arrives
			if set := c.adaptor.AcquireTxSet(newPeerProp.position); set != nil {
				c.gotTxSet(c.now, set)
			} else {
				log.Println("Don't have tx set for peer")
			}
		} else {
			if c.result != nil {
				c.updateDisputes(newPeerProp.nodeID, ait)
			}
		}
		return true
	}
}

/** Call periodically to drive consensus forward.

  @param now The network adjusted time
*/

func (c *consensus) timerEntry(now time.Time) {
	// Nothing to do if we are currently working on a ledger
	if c.phase == accepted {
		return
	}

	c.now = now

	// Check we are on the proper ledger (this may change phase_)
	c.checkLedger()

	if c.phase == open {
		c.phaseOpen()
	} else {
		if c.phase == establish {
			c.phaseEstablish()
		}
	}
}

/** Process a transaction set acquired from the network

  @param now The network adjusted time
  @param txSet the transaction set
*/
func (c *consensus) gotTxSet(now time.Time, ts txSet) {
	// Nothing to do if we've finished work on a ledger
	if c.phase == accepted {
		return
	}
	c.now = now

	id := ts.ID()

	// If we've already processed this transaction set since requesting
	// it from the network, there is nothing to do now
	if _, ok := c.acquired[id]; ok {
		return
	}

	if c.result == nil {
		log.Println("Not creating disputes: no position yet.")
	} else {
		// Our position is added to acquired_ as soon as we create it,
		// so this txSet must differ
		if id == c.result.position.position {
			panic("invalid id")
		}
		any := false
		for nid, pos := range c.currPeerPositions {
			if pos.Proposal().position == id {
				c.updateDisputes(nid, ts)
				any = true
			}
		}

		if !any {
			log.Println("By the time we got ", id, " no peers were proposing it")
		}
	}
}

/** Simulate the consensus process without any network traffic.

  The end result, is that consensus begins and completes as if everyone
  had agreed with whatever we propose.

  This function is only called from the rpc "ledger_accept" path with the
  server in standalone mode and SHOULD NOT be used during the normal
  consensus process.

  Simulate will call onForceAccept since clients are manually driving
  consensus to the accept phase.

  @param now The current network adjusted time.
  @param consensusDelay Duration to delay between closing and accepting the
                        ledger. Uses 100ms if unspecified.
*/

func (c *consensus) simulate(
	now time.Time,
	consensusDelay time.Duration) {
	log.Println("Simulating consensus")
	c.now = now
	c.closeLedger()
	if consensusDelay > 0 {
		c.result.roundTime.tick(consensusDelay)
	} else {
		c.result.roundTime.tick(100 * time.Millisecond)
	}
	c.prevProposers = uint(len(c.currPeerPositions))
	c.result.proposers = c.prevProposers
	c.prevRoundTime = c.result.roundTime.read()
	c.phase = accepted
	c.adaptor.OnForceAccept(
		c.result,
		c.previousLedger,
		c.closeResolution,
		c.rawCloseTimes,
		c.mode.mode)
	log.Println("Simulation complete")
}

// Change our view of the previous ledger
// Handle a change in the prior ledger during a consensus round
func (c *consensus) handleWrongLedger(lgrID LedgerID) {
	if lgrID == c.prevLedgerID && c.previousLedger.ID() == lgrID {
		panic("invalid arguments")
	}

	// Stop proposing because we are out of sync
	c.leaveConsensus()

	// First time switching to this ledger
	if c.prevLedgerID != lgrID {
		c.prevLedgerID = lgrID

		// Clear out state
		if c.result != nil {
			c.result.disputes = make(map[TxID]*disputedTx)
			c.result.compares = make(map[TxSetID]txSet)
		}

		c.currPeerPositions = make(map[NodeID]peerPosition)
		c.rawCloseTimes.peers = make(map[unixTime]int)
		c.deadNodes = make(map[NodeID]struct{})

		// Get back in sync, this will also recreate disputes
		c.playbackProposals()
	}

	if c.previousLedger.ID() == c.prevLedgerID {
		return
	}

	// we need to switch the ledger we're working from
	newLedger, err := c.adaptor.AcquireLedger(c.prevLedgerID)
	if err == nil {
		log.Println("Have the consensus ledger ", c.prevLedgerID)
		c.startRoundInternal(
			c.now, lgrID, newLedger, switchedLedger)
	} else {
		c.mode.set(wrongLedger, c.adaptor)
	}
}

/** Check if our previous ledger matches the network's.

  If the previous ledger differs, we are no longer in sync with
  the network and need to bow out/switch modes.
*/
func (c *consensus) checkLedger() {
	netLgr :=
		c.adaptor.GetPrevLedger(c.prevLedgerID, c.previousLedger, c.mode.mode)

	if netLgr != c.prevLedgerID {
		log.Println("View of consensus changed during ",
			c.phase, " status=", c.phase,
			", ",
			" mode=", c.mode.mode)
		log.Println(c.prevLedgerID, " to ", netLgr)
		log.Println(c.previousLedger)
		log.Println("State on consensus change ", c)
		c.handleWrongLedger(netLgr)
	} else {
		if c.previousLedger.ID() != c.prevLedgerID {
			c.handleWrongLedger(netLgr)
		}
	}
}

/** If we radically changed our consensus context for some reason,
  we need to replay recent proposals so that they're not lost.
*/
func (c *consensus) playbackProposals() {
	for _, it := range c.recentPeerPositions {
		for _, pos := range it {
			if pos.Proposal().previousLedger == c.prevLedgerID {
				if c.peerProposalInternal(c.now, pos) {
					c.adaptor.SharePositoin(pos)
				}
			}
		}
	}
}

/** Handle pre-close phase.

  In the pre-close phase, the ledger is open as we wait for new
  transactions.  After enough time has elapsed, we will close the ledger,
  switch to the establish phase and start the consensus process.
*/
func (c *consensus) phaseOpen() {
	// it is shortly before ledger close time
	anyTransactions := c.adaptor.HasOpenTransactions()
	proposersClosed := len(c.currPeerPositions)
	proposersValidated := c.adaptor.ProposersValidated(c.prevLedgerID)

	c.openTime.tickTime(time.Now())

	// This computes how long since last ledger's close time
	var sinceClose time.Duration
	previousCloseCorrect :=
		(c.mode.mode != wrongLedger) &&
			c.previousLedger.CloseAgree() &&
			(!c.previousLedger.CloseTime().Equal(
				(c.previousLedger.ParentCloseTime().Add(1 * time.Second))))

	lastCloseTime := c.prevCloseTime // use the time we saw internally
	if previousCloseCorrect {
		lastCloseTime = c.previousLedger.CloseTime() // use consensus timing
	}

	if c.now.After(lastCloseTime) {
		sinceClose = c.now.Sub(lastCloseTime)
	} else {
		sinceClose = -lastCloseTime.Sub(c.now)
	}

	idleInterval := 2 * c.previousLedger.CloseTimeResolution()
	if idleInterval < ledgerIdleInterval {
		idleInterval = ledgerIdleInterval
	}

	// Decide if we should close the ledger
	if shouldCloseLedger(
		anyTransactions,
		c.prevProposers,
		uint(proposersClosed),
		proposersValidated,
		c.prevRoundTime,
		sinceClose,
		c.openTime.read(),
		idleInterval) {
		c.closeLedger()
	}
}

/** Handle establish phase.

  In the establish phase, the ledger has closed and we work with peers
  to reach consensus. Update our position only on the timer, and in this
  phase.

  If we have consensus, move to the accepted phase.
*/
func (c *consensus) phaseEstablish() {
	// can only establish consensus if we already took a stance
	if c.result == nil {
		panic("result is nil")
	}

	c.result.roundTime.tickTime(time.Now())
	c.result.proposers = uint(len(c.currPeerPositions))

	p := c.prevRoundTime
	if p < avMinConsensusTime {
		p = avMinConsensusTime
	}

	c.convergePercent = int(c.result.roundTime.read() * 100 / p)

	// Give everyone a chance to take an initial position
	if c.result.roundTime.read() < ledgerMinConsensus {
		return
	}

	c.updateOurPositions()

	// Nothing to do if we don't have consensus.
	if !c.haveConsensus() {
		return
	}

	if !c.haveCloseTimeConsensus {
		log.Println("We have TX consensus but not CT consensus")
		return
	}

	log.Println("Converge cutoff (", len(c.currPeerPositions),
		" participants)")
	c.prevProposers = uint(len(c.currPeerPositions))
	c.prevRoundTime = c.result.roundTime.read()
	c.phase = accepted
	c.adaptor.OnAccept(
		c.result,
		c.previousLedger,
		c.closeResolution,
		c.rawCloseTimes,
		c.mode.mode)
}

// Close the open ledger and establish initial position.
func (c *consensus) closeLedger() {
	// We should not be closing if we already have a position
	if c.result == nil {
		panic("result is nil")
	}

	c.phase = establish
	c.rawCloseTimes.self = c.now

	c.result = c.adaptor.OnClose(c.previousLedger, c.now, c.mode.mode)
	c.result.roundTime.reset(time.Now())
	// Share the newly created transaction set if we haven't already
	// received it from a peer
	_, ok := c.acquired[c.result.txns.ID()]
	c.acquired[c.result.txns.ID()] = c.result.txns
	if !ok {
		c.adaptor.ShareTxset(c.result.txns)
	}

	if c.mode.mode == proposing {
		c.adaptor.Propose(c.result.position)
	}

	// Create disputes with any peer positions we have transactions for
	for _, pit := range c.currPeerPositions {
		pos := pit.Proposal().position
		it, ok := c.acquired[pos]
		if ok {
			c.createDisputes(it)
		}
	}
}

/** How many of the participants must agree to reach a given threshold?

Note that the number may not precisely yield the requested percentage.
For example, with with size = 5 and percent = 70, we return 3, but
3 out of 5 works out to 60%. There are no security implications to
this.

@param participants The number of participants (i.e. validators)
@param percent The percent that we want to reach

@return the number of participants which must agree
*/
func (c *consensus) participantsNeeded(participants, percent int) int {
	result := ((participants * percent) + (percent / 2)) / 100
	if result == 0 {
		return 1
	}
	return result
}

// Adjust our positions to try to agree with other validators.
func (c *consensus) updateOurPositions() {
	// We must have a position if we are updating it
	if c.result == nil {
		panic("result is nil")
	}
	// Compute a cutoff time
	peerCutoff := c.now.Add(-proposeFRESHNESS)
	ourCutoff := c.now.Add(-proposeINTERVAL)

	// Verify freshness of peer positions and compute close times
	closeTimeVotes := make(map[unixTime]int)
	{
		for nid, pos := range c.currPeerPositions {
			peerProp := pos.Proposal()
			if peerProp.isStale(peerCutoff) {
				// peer's proposal is stale, so remove it
				peerID := peerProp.nodeID
				log.Println("Removing stale proposal from ", peerID)
				for _, dt := range c.result.disputes {
					dt.unVote(peerID)
				}
				delete(c.currPeerPositions, nid)
			} else {
				// proposal is still fresh
				closeTimeVotes[unixTime(c.asCloseTime(peerProp.closeTime).Unix())]++
			}
		}
	}

	// This will stay unseated unless there are any changes
	var ourNewSet txSet

	// Update votes on disputed transactions
	{
		var mutableSet txSet
		for txid, disp := range c.result.disputes {
			// Because the threshold for inclusion increases,
			//  time can change our position on a dispute
			if disp.updateVote(c.convergePercent, c.mode.mode == proposing) {
				if mutableSet != nil {
					mutableSet = c.result.txns
				}

				if disp.ourVote {
					// now a yes
					mutableSet.Insert(disp.tx)
				} else {
					// now a no
					mutableSet.Erase(txid)
				}
			}
		}

		if mutableSet != nil {
			ourNewSet = mutableSet
		}
	}

	consensusCloseTime := time.Time{}
	c.haveCloseTimeConsensus = false

	if len(c.currPeerPositions) == 0 {
		// no other times
		c.haveCloseTimeConsensus = true
		consensusCloseTime = c.asCloseTime(c.result.position.closeTime)
	} else {
		neededWeight := 0

		if c.convergePercent < avMidConsensusTime {
			neededWeight = avInitConsensusPCT
		} else {
			if c.convergePercent < avLateConsensusTime {
				neededWeight = avMIDConsensusPCT
			} else {
				if c.convergePercent < avStuckConsensusTime {
					neededWeight = avLateConsensusPCT
				} else {
					neededWeight = avStuckConsensusPCT
				}
			}
		}
		participants := len(c.currPeerPositions)
		if c.mode.mode == proposing {
			closeTimeVotes[unixTime(c.asCloseTime(c.result.position.closeTime).Unix())]++
			participants++
		}

		// Threshold for non-zero vote
		threshVote := c.participantsNeeded(participants, neededWeight)

		// Threshold to declare consensus
		threshConsensus :=
			c.participantsNeeded(participants, avCTConsensusPCT)

		log.Println("Proposers:", len(c.currPeerPositions),
			" nw:", neededWeight, " thrV:", threshVote,
			" thrC:", threshConsensus)

		for tim, cnt := range closeTimeVotes {
			log.Println(
				"CCTime: seq ",
				c.previousLedger.Seq()+1, ": ",
				tim, " has ", cnt,
				", ", threshVote, " required")

			if cnt >= threshVote {
				// A close time has enough votes for us to try to agree
				consensusCloseTime = time.Unix(int64(tim), 0)
				threshVote = cnt

				if threshVote >= threshConsensus {
					c.haveCloseTimeConsensus = true
				}
			}
		}

		if !c.haveCloseTimeConsensus {
			log.Println(
				"No CT consensus:",
				" Proposers:", len(c.currPeerPositions),
				" Mode:", c.mode.mode,
				" Thresh:", threshConsensus,
				" Pos:", consensusCloseTime)
		}
	}

	if ourNewSet != nil &&
		((consensusCloseTime != c.asCloseTime(c.result.position.closeTime)) ||
			c.result.position.isStale(ourCutoff)) {
		// close time changed or our position is stale
		ourNewSet = c.result.txns
	}

	if ourNewSet != nil {
		newID := ourNewSet.ID()

		c.result.txns = ourNewSet

		log.Println("Position change: CTime ",
			consensusCloseTime, ", tx ", newID)

		c.result.position.changePosition(newID, consensusCloseTime, c.now)

		// Share our new transaction set and update disputes
		// if we haven't already received it

		_, ok := c.acquired[newID]
		c.acquired[newID] = c.result.txns
		if !ok {
			if !c.result.position.isBowOut() {
				c.adaptor.ShareTxset(c.result.txns)
			}
			for nid, pos := range c.currPeerPositions {
				p := pos.Proposal()
				if p.position == newID {
					c.updateDisputes(nid, c.result.txns)
				}
			}
		}

		// Share our new position if we are still participating this round
		if !c.result.position.isBowOut() &&
			(c.mode.mode == proposing) {
			c.adaptor.Propose(c.result.position)
		}
	}
}

func (c *consensus) haveConsensus() bool {
	// Must have a stance if we are checking for consensus
	if c.result == nil {
		panic("result is nil")
	}

	// CHECKME: should possibly count unacquired TX sets as disagreeing
	var agree, disagree int

	ourPosition := c.result.position.position

	// Count number of agreements/disagreements with our position
	for nid, pos := range c.currPeerPositions {
		peerProp := pos.Proposal()
		if peerProp.position == ourPosition {
			agree++
		} else {
			log.Println(nid, " has ", peerProp.position)
			disagree++
		}
	}
	currentFinished :=
		c.adaptor.ProposersFinished(c.previousLedger, c.prevLedgerID)

	log.Println("Checking for TX consensus: agree=", agree,
		", disagree=", disagree)

	// Determine if we actually have consensus or not
	c.result.state = checkConsensus(
		c.prevProposers,
		uint(agree+disagree),
		uint(agree),
		currentFinished,
		c.prevRoundTime,
		c.result.roundTime.read(),
		c.mode.mode == proposing,
	)

	if c.result.state == no {
		return false
	}

	// There is consensus, but we need to track if the network moved on
	// without us.
	if c.result.state == movedOn {
		log.Println("Unable to reach consensus")
		log.Println(c)
	}

	return true
}

// Revoke our outstanding proposal, if any, and cease proposing
// until this round ends.
func (c *consensus) leaveConsensus() {
	if c.mode.mode == proposing {
		if c.result != nil && !c.result.position.isBowOut() {
			c.result.position.bowOut(c.now)
			c.adaptor.Propose(c.result.position)
		}

		c.mode.set(observing, c.adaptor)
		log.Println("Bowing out of consensus")
	}
}

// Create disputes between our position and the provided one.
func (c *consensus) createDisputes(o txSet) {
	// Cannot create disputes without our stance
	if c.result == nil {
		panic("result is nil")
	}

	// Only create disputes if this is a new set
	if _, ok := c.result.compares[o.ID()]; ok {
		return
	}

	// Nothing to dispute if we agree
	if c.result.txns.ID() == o.ID() {
		return
	}

	log.Println("createDisputes ", c.result.txns.ID(), " to ", o.ID())

	differences := c.result.txns.Compare(o)

	dc := 0

	for se, id := range differences {
		dc++
		// create disputed transactions (from the ledger that has them)
		if !((id && c.result.txns.Find(se) != nil && o.Find(se) == nil) ||
			(!id && c.result.txns.Find(se) == nil && o.Find(se) != nil)) {
			panic("invalid ledger")
		}

		tx := o.Find(se)
		if id {
			tx = c.result.txns.Find(se)
		}
		txID := tx.ID()

		if _, ok := c.result.disputes[txID]; ok {
			continue
		}

		log.Println("Transaction ", txID, " is disputed")

		num := c.prevProposers
		if num < uint(len(c.currPeerPositions)) {
			num = uint(len(c.currPeerPositions))
		}

		dtx := newDisputedTx(tx, c.result.txns.Exists(txID), num)

		// Update all of the available peer's votes on the disputed transaction
		for nid, pos := range c.currPeerPositions {
			peerProp := pos.Proposal()
			cit, ok := c.acquired[peerProp.position]
			if ok {
				dtx.setVote(nid, cit.Exists(txID))
			}
		}
		c.adaptor.ShareTx(dtx.tx)

		c.result.disputes[txID] = dtx
	}
	log.Println(dc, " differences found")
}

// Update our disputes given that this node has adopted a new position.
// Will call createDisputes as needed.
func (c *consensus) updateDisputes(node NodeID, other txSet) {
	// Cannot updateDisputes without our stance
	if c.result == nil {
		panic("result is nil")
	}

	// Ensure we have created disputes against this set if we haven't seen
	// it before
	if _, ok := c.result.compares[other.ID()]; !ok {
		c.createDisputes(other)
	}

	for _, it := range c.result.disputes {
		it.setVote(node, other.Exists(it.tx.ID()))
	}
}
func (c *consensus) asCloseTime(raw time.Time) time.Time {
	if useRoundedCloseTime {
		return roundCloseTime(raw, c.closeResolution)
	}
	return effCloseTime(raw, c.closeResolution, c.previousLedger.CloseTime())
}
