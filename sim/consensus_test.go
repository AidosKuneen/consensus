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

package sim

import (
	"fmt"
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/AidosKuneen/consensus"
)

func expect(t *testing.T, b bool) bool {
	if !b {
		_, file, no, _ := runtime.Caller(1)
		t.Error("failed at", fmt.Sprintf("%v:%v", file, no))
	}
	return b
}

func TestStandalone(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	s := newSim()
	peers := s.createGroup(1)
	p := peers.peers[0]
	p.targetLedgers = 1
	p.start()
	var id consensus.TxID
	id[0] = 1
	p.submit(&tx{id: id})
	s.scheduler.step()
	lcl := p.lastClosedLedger
	expect(t, p.PrevLedgerID() == lcl.ID())
	expect(t, lcl.Seq == 1)
	expect(t, len(lcl.Txs) == 1)
	_, ok := lcl.Txs[id]
	expect(t, ok)
	expect(t, p.prevProposers == 0)
}
func TestPeersAgree(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	sim := newSim()
	peers := sim.createGroup(5)

	// Connected trust and network graphs with single fixed delay
	peers.trustAndConnect(
		peers, consensus.LedgerGranularity/5)

	// everyone submits their own ID as a TX
	for _, p := range peers.peers {
		p.submit(&tx{consensus.TxID(p.id)})
	}
	sim.run(1)

	// All peers are in sync
	if expect(t, sim.synchronized(&sim.allPeers)) {
		for _, peer := range peers.peers {
			lcl := peer.lastClosedLedger
			expect(t, lcl.ID() == peer.PrevLedgerID())
			expect(t, lcl.Seq == 1)
			// All peers proposed
			t.Log(peer.prevProposers)
			expect(t, peer.prevProposers == len(peers.peers)-1)
			// All transactions were accepted
			for i := 0; i < len(peers.peers); i++ {
				var id consensus.TxID
				id[0] = byte(i)
				_, ok := lcl.Txs[id]
				expect(t, ok)
			}
		}
	}
}
func TestSlowPeers1(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Several tests of a complete trust graph with a subset of peers
	// that have significantly longer network delays to the rest of the
	// network

	// Test when a slow peer doesn't delay a consensus quorum (4/5 agree)
	{
		sim := newSim()
		slow := sim.createGroup(1)
		fast := sim.createGroup(4)
		network := fast.append(slow)

		// Fully connected trust graph
		network.trust(network)

		// Fast and slow network connections
		fast.connect(
			fast, consensus.LedgerGranularity/5)

		// Fast and slow network connections
		slow.connect(
			network, consensus.LedgerGranularity*11/10)

		// All peers submit their own ID as a transaction
		for _, peer := range network.peers {
			peer.submit(&tx{
				id: consensus.TxID(peer.id),
			})
		}
		sim.run(1)

		// Verify all peers have same LCL but are missing transaction 0
		// All peers are in sync even with a slower peer 0
		if expect(t, sim.synchronized(&sim.allPeers)) {
			for _, peer := range network.peers {
				lcl := peer.lastClosedLedger
				expect(t, lcl.ID() == peer.PrevLedgerID())
				expect(t, lcl.Seq == 1)
				t.Log(peer.id, peer.prevProposers)
				expect(t, peer.prevProposers == len(network.peers)-1)
				var id consensus.TxID
				_, ok := lcl.Txs[id]
				expect(t, !ok)
				for i := 2; i < len(network.peers); i++ {
					id[0] = byte(i)
					_, ok2 := lcl.Txs[id]
					expect(t, ok2)
				}
				// Tx 0 didn't make it
				var id2 consensus.TxID
				_, ok = peer.openTxs[id2]
				expect(t, ok)
			}
		}
	}
}

func TestSlowPeers2(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Run two tests
	//  1. The slow peers are participating in consensus
	//  2. The slow peers are just observing

	// Test when the slow peers delay a consensus quorum (4/6 agree)
	for _, isParticipant := range []bool{true, false} {
		sim := newSim()
		slow := sim.createGroup(2)
		fast := sim.createGroup(4)
		network := fast.append(slow)

		// Fully connected trust graph
		network.trust(network)

		// Fast and slow network connections
		fast.connect(
			fast, consensus.LedgerGranularity/5)
		slow.connect(
			network, consensus.LedgerGranularity*11/10)

		for _, p := range slow.peers {
			p.runAsValidator = isParticipant
		}

		// All peers submit their own ID as a transaction
		for _, peer := range network.peers {
			peer.submit(&tx{
				id: consensus.TxID(peer.id),
			})
		}
		sim.run(1)

		// Verify all peers have same LCL but are missing transaction 0
		// All peers are in sync even with a slower peer 0
		if expect(t, sim.synchronized(&sim.allPeers)) {
			for _, peer := range network.peers {
				lcl := peer.lastClosedLedger
				expect(t, lcl.ID() == peer.PrevLedgerID())
				expect(t, lcl.Seq == 1)
				var id consensus.TxID
				_, ok := lcl.Txs[id]
				expect(t, !ok)
				id[0] = 1
				_, ok = lcl.Txs[id]
				expect(t, !ok)

				for i := len(slow.peers); i < len(network.peers); i++ {
					id[0] = byte(i)
					_, ok = lcl.Txs[id]
					expect(t, ok)
				}
				id[0] = 0
				_, ok = peer.openTxs[id]
				expect(t, ok)
				id[0] = 1
				_, ok = peer.openTxs[id]
				expect(t, ok)
			}
			slowp := slow.peers[0]
			if isParticipant {
				expect(t, slowp.prevProposers == len(network.peers)-1)
			} else {
				expect(t, slowp.prevProposers == len(fast.peers))
			}
			for _, p := range fast.peers {
				// Due to the network link delay settings
				//    Peer 0 initially proposes {0}
				//    Peer 1 initially proposes {1}
				//    Peers 2-5 initially propose {2,3,4,5}
				// Since peers 2-5 agree, 4/6 > the initial 50% needed
				// to include a disputed transaction, so Peer 0/1 switch
				// to agree with those peers. Peer 0/1 then closes with
				// an 80% quorum of agreeing positions (5/6) match.
				//
				// Peers 2-5 do not change position, since tx 0 or tx 1
				// have less than the 50% initial threshold. They also
				// cannot declare consensus, since 4/6 agreeing
				// positions are < 80% threshold. They therefore need an
				// additional timerEntry call to see the updated
				// positions from Peer 0 & 1.
				if isParticipant {
					expect(t, p.prevProposers == len(network.peers)-1)
					expect(t, p.prevRoundTime.Start.Add(p.prevRoundTime.Dur).After(
						slowp.prevRoundTime.Start.Add(slowp.prevRoundTime.Dur)))
				} else {
					expect(t, p.prevProposers == len(fast.peers)-1)
					expect(t, p.prevRoundTime == slowp.prevRoundTime)
				}
			}
		}
	}
}
func TestCloseTimeDisagree(t *testing.T) {
	// This is a very specialized test to get ledgers to disagree on
	// the close time. It unfortunately assumes knowledge about current
	// timing constants. This is a necessary evil to get coverage up
	// pending more extensive refactorings of timing constants.

	// In order to agree-to-disagree on the close time, there must be no
	// clear majority of nodes agreeing on a close time. This test
	// sets a relative offset to the peers internal clocks so that they
	// send proposals with differing times.

	// However, agreement is on the effective close time, not the
	// exact close time. The minimum closeTimeResolution is given by
	// ledgerPossibleTimeResolutions[0], which is currently 10s. This means
	// the skews need to be at least 10 seconds to have different effective
	// close times.

	// Complicating this matter is that nodes will ignore proposals
	// with times more than proposeFRESHNESS =20s in the past. So at
	// the minimum granularity, we have at most 3 types of skews
	// (0s,10s,20s).

	// This test therefore has 6 nodes, with 2 nodes having each type of
	// skew. Then no majority (1/3 < 1/2) of nodes will agree on an
	// actual close time.
	// log.SetFlags(log.Ltime | log.Llongfile)
	sim := newSim()
	grA := sim.createGroup(2)
	grB := sim.createGroup(2)
	grC := sim.createGroup(2)
	network := grA.append(grB).append(grC)

	network.trust(network)
	network.connect(
		network, consensus.LedgerGranularity/5)

	// Run consensus without skew until we have a short close time
	// resolution
	p := grA.peers[0]
	for p.lastClosedLedger.CloseTimeResolution >= proposeFreshness {
		sim.run(1)
	}
	log.Println("end of init")
	// Introduce a shift on the time of 2/3 of peers
	for _, peer := range grA.peers {
		peer.clockSkew = proposeFreshness / 2
	}
	for _, peer := range grB.peers {
		peer.clockSkew = proposeFreshness
	}
	sim.run(1)
	// All nodes agreed to disagree on the close time
	if expect(t, sim.synchronized(&sim.allPeers)) {
		for _, peer := range network.peers {
			t.Log(peer.lastClosedLedger.CloseTime)
			expect(t, !peer.lastClosedLedger.CloseTimeAgree)
		}
	}
}
func TestWrongLCL1(t *testing.T) {
	// Vary the time it takes to process validations to exercise detecting
	// the wrong LCL at different phases of consensus
	for _, validationDelay := range []time.Duration{0, ledgerMinClose} {
		t.Log("delay", validationDelay)
		// Consider 10 peers:
		// 0 1         2 3 4       5 6 7 8 9
		// minority   majorityA   majorityB
		//
		// Nodes 0-1 trust nodes 0-4
		// Nodes 2-9 trust nodes 2-9
		//
		// By submitting tx 0 to nodes 0-4 and tx 1 to nodes 5-9,
		// nodes 0-1 will generate the wrong LCL (with tx 0). The remaining
		// nodes will instead accept the ledger with tx 1.

		// Nodes 0-1 will detect this mismatch during a subsequent round
		// since nodes 2-4 will validate a different ledger.

		// Nodes 0-1 will acquire the proper ledger from the network and
		// resume consensus and eventually generate the dominant network
		// ledger.

		// This topology can potentially fork with the above trust relations
		// but that is intended for this test.

		// log.SetFlags(log.Ltime | log.Llongfile)
		sim := newSim()
		jumps := make(jumpCollectorByNode)
		sim.collectors = append(sim.collectors, jumps.on)

		minority := sim.createGroup(2)
		majorityA := sim.createGroup(3)
		majorityB := sim.createGroup(5)
		majority := majorityA.append(majorityB)
		network := minority.append(majority)

		delay := LedgerGranularity / 5

		minority.trustAndConnect(minority.append(majorityA), delay)
		majority.trustAndConnect(majority, delay)

		expect(t, sim.trustGraph.canfork(float64(minConsensusPCT)/100.0))

		// initial round to set prior state
		sim.run(1)
		log.Println("end of round1")
		// Nodes in smaller UNL have seen tx 0, nodes in other unl have seen
		// tx 1
		for _, peer := range network.peers {
			peer.processingDelays.recvValidation = validationDelay
		}
		var id consensus.TxID
		for _, peer := range minority.append(majorityA).peers {
			peer.openTxs[id] = &tx{
				id: id,
			}
		}
		id[0] = 1
		for _, peer := range majorityB.peers {
			peer.openTxs[id] = &tx{
				id: id,
			}
		}
		// Run for additional rounds
		// With no validation delay, only 2 more rounds are needed.
		//  1. Round to generate different ledgers
		//  2. Round to detect different prior ledgers (but still generate
		//    wrong ones) and recover within that round since wrong LCL
		//    is detected before we close
		//
		// With a validation delay of ledgerMIN_CLOSE, we need 3 more
		// rounds.
		//  1. Round to generate different ledgers
		//  2. Round to detect different prior ledgers (but still generate
		//     wrong ones) but end up declaring consensus on wrong LCL (but
		//     with the right transaction set!). This is because we detect
		//     the wrong LCL after we have closed the ledger, so we declare
		//     consensus based solely on our peer proposals. But we haven't
		//     had time to acquire the right ledger.
		//  3. Round to correct
		sim.run(3)

		// The network never actually forks, since node 0-1 never see a
		// quorum of validations to fully validate the incorrect chain.

		// However, for a non zero-validation delay, the network is not
		// synchronized because nodes 0 and 1 are running one ledger behind
		if expect(t, sim.branches(&sim.allPeers) == 1) {
			for _, peer := range majority.peers {
				// No jumps for majority nodes
				expect(t, len(jumps[peer.id].closeJumps) == 0)
				expect(t, len(jumps[peer.id].fullyValidatedJumps) == 0)
			}
			for _, peer := range minority.peers {
				t.Log("pid", peer.id)
				peerJumps := jumps[peer.id]
				// last closed ledger jump between chains
				{
					t.Log(len(peerJumps.closeJumps))
					if expect(t, len(peerJumps.closeJumps) == 1) {
						jump :=
							peerJumps.closeJumps[0]
						// Jump is to a different chain
						expect(t, jump.from.Seq <= jump.to.Seq)
						expect(t, !jump.to.IsAncestor(jump.from))
					}
				}
				// fully validated jump forward in same chain
				{
					t.Log(len(peerJumps.fullyValidatedJumps))

					if expect(t,
						len(peerJumps.fullyValidatedJumps) == 1) {
						jump :=
							peerJumps.fullyValidatedJumps[0]
						// Jump is to a different chain with same seq
						expect(t, jump.from.Seq < jump.to.Seq)
						expect(t, jump.to.IsAncestor(jump.from))
					}
				}
			}
		}
	}
}
func TestWrongLCL2(t *testing.T) {
	// Additional test engineered to switch LCL during the establish
	// phase. This was added to trigger a scenario that previously
	// crashed, in which switchLCL switched from establish to open
	// phase, but still processed the establish phase logic.

	// Loner node will accept an initial ledger A, but all other nodes
	// accept ledger B a bit later. By delaying the time it takes
	// to process a validation, loner node will detect the wrongLCL
	// after it is already in the establish phase of the next round.

	// log.SetFlags(log.Ltime | log.Llongfile)
	sim := newSim()

	loner := sim.createGroup(1)
	friends := sim.createGroup(3)
	loner.trust(loner.append(friends))

	others := sim.createGroup(6)
	clique := friends.append(others)
	clique.trust(clique)
	network := loner.append(clique)

	network.connect(network, LedgerGranularity/5)
	sim.run(1)

	var id consensus.TxID
	for _, peer := range loner.append(friends).peers {
		peer.openTxs[id] = &tx{
			id: id,
		}
	}
	id[0] = 1
	for _, peer := range others.peers {
		peer.openTxs[id] = &tx{
			id: id,
		}
	}
	for _, peer := range network.peers {
		peer.processingDelays.recvValidation = LedgerGranularity
	}

	sim.run(2)
	for _, peer := range network.peers {
		// No jumps for majority nodes
		expect(t, peer.PrevLedgerID() == network.peers[0].PrevLedgerID())
	}
}
func TestConsensusCloseTimeRounding(t *testing.T) {

	// This is a specialized test engineered to yield ledgers with different
	// close times even though the peers believe they had close time
	// consensus on the ledger.

	for _, useRoundedCloseTime := range []bool{false, true} {
		t.Log(useRoundedCloseTime)
		// log.SetFlags(log.Ltime | log.Llongfile)
		sim := newSim()

		// This requires a group of 4 fast and 2 slow peers to create a
		// situation in which a subset of peers requires seeing additional
		// proposals to declare consensus.
		slow := sim.createGroup(2)
		fast := sim.createGroup(4)
		network := fast.append(slow)

		consensus.SetUseRoundedCloseTime(useRoundedCloseTime)
		// Connected trust graph
		network.trust(network)

		// Fast and slow network connections
		fast.connect(fast, LedgerGranularity/5)
		slow.connect(network, LedgerGranularity*11/10)

		// Run to the ledger *prior* to decreasing the resolution
		sim.run(increaseLedgerTimeResolutionEvery - 2)
		log.Println("end of round1")
		// In order to create the discrepency, we want a case where if
		//   X = effCloseTime(closeTime, resolution, parentCloseTime)
		//   X != effCloseTime(X, resolution, parentCloseTime)
		//
		// That is, the effective close time is not a fixed point. This can
		// happen if X = parentCloseTime + 1, but a subsequent rounding goes
		// to the next highest multiple of resolution.

		// So we want to find an offset  (now + offset) % 30s = 15
		//                               (now + offset) % 20s = 15
		// This way, the next ledger will close and round up   Due to the
		// network delay settings, the round of consensus will take 5s, so
		// the next ledger's close time will

		when := network.peers[0].now()

		// Check we are before the 30s to 20s transition
		resolution :=
			network.peers[0].lastClosedLedger.CloseTimeResolution
		t.Log(resolution)
		expect(t, resolution == 30*time.Second)

		for when.Second()%30 != 15 || when.Second()%20 != 15 {
			when = when.Add(time.Second)
		}
		log.Println(when)
		// Advance the clock without consensus running (IS THIS WHAT
		// PREVENTS IT IN PRACTICE?)
		sim.scheduler.stepFor(when.Sub(network.peers[0].now()))
		log.Println("end of round2")

		// Run one more ledger with 30s resolution
		sim.run(1)
		log.Println("end of round3")

		if expect(t, sim.synchronized(&sim.allPeers)) {
			// close time should be ahead of clock time since we engineered
			// the close time to round up
			for _, peer := range network.peers {
				expect(t,
					peer.lastClosedLedger.CloseTime.After(peer.now()))
				expect(t, peer.lastClosedLedger.CloseTimeAgree)
			}
		}

		// All peers submit their own ID as a transaction
		for _, peer := range network.peers {
			peer.submit(&tx{
				id: consensus.TxID(peer.id),
			})
		}
		// Run 1 more round, this time it will have a decreased
		// resolution of 20 seconds.

		// The network delays are engineered so that the slow peers
		// initially have the wrong tx hash, but they see a majority
		// of agreement from their peers and declare consensus
		//
		// The trick is that everyone starts with a raw close time of
		//  84681s
		// Which has
		//   effCloseTime(86481s, 20s,  86490s) = 86491s
		// However, when the slow peers update their position, they change
		// the close time to 86451s. The fast peers declare consensus with
		// the 86481s as their position still.
		//
		// When accepted the ledger
		// - fast peers use eff(86481s) . 86491s as the close time
		// - slow peers use eff(eff(86481s)) . eff(86491s) . 86500s!

		sim.run(1)
		log.Println("end of round4")

		if useRoundedCloseTime {
			expect(t, sim.synchronized(&sim.allPeers))
		} else {
			// Not currently synchronized
			expect(t, !sim.synchronized(&sim.allPeers))

			// All slow peers agreed on LCL
			for _, p := range slow.peers {
				expect(t, p.lastClosedLedger.ID() == slow.peers[0].lastClosedLedger.ID())
			}
			// All fast peers agreed on LCL
			for _, p := range fast.peers {
				expect(t, p.lastClosedLedger.ID() == fast.peers[0].lastClosedLedger.ID())
			}

			slowLCL := slow.peers[0].lastClosedLedger
			fastLCL := fast.peers[0].lastClosedLedger

			// Agree on parent close and close resolution
			expect(t,
				slowLCL.ParentCloseTime == fastLCL.ParentCloseTime)
			expect(t,
				slowLCL.CloseTimeResolution ==
					fastLCL.CloseTimeResolution)

			// Close times disagree ...
			expect(t, slowLCL.CloseTime != fastLCL.CloseTime)
			// Effective close times agree! The slow peer already rounded!
			expect(t,
				consensus.EffCloseTime(
					slowLCL.CloseTime,
					slowLCL.CloseTimeResolution,
					slowLCL.ParentCloseTime) ==
					consensus.EffCloseTime(
						fastLCL.CloseTime,
						fastLCL.CloseTimeResolution,
						fastLCL.ParentCloseTime))
		}
	}
	consensus.SetUseRoundedCloseTime(true)
}

func TestFork(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Llongfile)

	numPeers := 10
	// Vary overlap between two UNLs
	for overlap := 0; overlap <= numPeers; overlap++ {
		sim := newSim()

		numA := (numPeers - overlap) / 2
		numB := numPeers - numA - overlap

		aOnly := sim.createGroup(numA)
		bOnly := sim.createGroup(numB)
		commonOnly := sim.createGroup(overlap)

		a := aOnly.append(commonOnly)
		b := bOnly.append(commonOnly)

		network := a.append(b)

		delay := LedgerGranularity / 5
		a.trustAndConnect(a, delay)
		b.trustAndConnect(b, delay)

		// Initial round to set prior state
		sim.run(1)
		for _, peer := range network.peers {
			// Nodes have only seen transactions from their neighbors
			id := consensus.TxID(peer.id)
			peer.openTxs[id] = &tx{
				id: id,
			}
			for _, to := range sim.trustGraph.trustedPeers(peer) {
				id := consensus.TxID(to.id)
				peer.openTxs[id] = &tx{
					id: id,
				}
			}
		}
		sim.run(1)

		// Fork should not happen for 40% or greater overlap
		// Since the overlapped nodes have a UNL that is the union of the
		// two cliques, the maximum sized UNL list is the number of peers
		if overlap > numPeers*2/5 {
			expect(t, sim.synchronized(&sim.allPeers))
		} else {
			// Even if we do fork, there shouldn't be more than 3 ledgers
			// One for cliqueA, one for cliqueB and one for nodes in both
			expect(t, sim.branches(&sim.allPeers) <= 3)
		}
	}
}

func TestHubNetwork(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Llongfile)

	// Simulate a set of 5 validators that aren't directly connected but
	// rely on a single hub node for communication

	sim := newSim()
	validators := sim.createGroup(5)
	center := sim.createGroup(1)
	validators.trust(validators)
	center.trust(validators)

	delay := LedgerGranularity / 5
	validators.connect(center, delay)

	center.peers[0].runAsValidator = false

	// prep round to set initial state.
	sim.run(1)

	// everyone submits their own ID as a TX and relay it to peers
	for _, p := range validators.peers {
		p.submit(&tx{
			id: consensus.TxID(p.id),
		})
	}
	sim.run(1)

	// All peers are in sync
	expect(t, sim.synchronized(&sim.allPeers))
}

type disruptor struct {
	network     *peerGroup
	groupCfast  *peerGroup
	gropuCsplit *peerGroup
	delay       time.Duration
	reconnected bool
}

func (d *disruptor) on(who consensus.NodeID, when time.Time, ee interface{}) {
	switch e := ee.(type) {
	case fullyValidateLedger:
		if who == d.groupCfast.peers[0].id && e.ledger.Seq == 2 {
			d.network.disconnect(d.gropuCsplit)
			d.network.disconnect(d.groupCfast)
		}
	case *fullyValidateLedger:
		if who == d.groupCfast.peers[0].id && e.ledger.Seq == 2 {
			d.network.disconnect(d.gropuCsplit)
			d.network.disconnect(d.groupCfast)
		}
	case acceptLedger:
		if !d.reconnected && e.ledger.Seq == 3 {
			d.reconnected = true
			d.network.connect(d.gropuCsplit, d.delay)
		}
	case *acceptLedger:
		if !d.reconnected && e.ledger.Seq == 3 {
			d.reconnected = true
			d.network.connect(d.gropuCsplit, d.delay)
		}
	}

}

func TestPreferredByBranch(t *testing.T) {
	// Simulate network splits that are prevented from forking when using
	// preferred ledger by trie.  This is a contrived example that involves
	// excessive network splits, but demonstrates the safety improvement
	// from the preferred ledger by trie approach.

	// Consider 10 validating nodes that comprise a single common UNL
	// Ledger history:
	// 1:           A
	//            _/ \_
	// 2:         B    C
	//          _/  _/  \_
	// 3:       D   C'  |||||||| (8 different ledgers)

	// - All nodes generate the common ledger A
	// - 2 nodes generate B and 8 nodes generate C
	// - Only 1 of the C nodes sees all the C validations and fully
	//   validates C. The rest of the C nodes split at just the right time
	//   such that they never see any C validations but their own.
	// - The C nodes continue and generate 8 different child ledgers.
	// - Meanwhile, the D nodes only saw 1 validation for C and 2 validations
	//   for B.
	// - The network reconnects and the validations for generation 3 ledgers
	//   are observed (D and the 8 C's)
	// - In the old approach, 2 votes for D outweights 1 vote for each C'
	//   so the network would avalanche towards D and fully validate it
	//   EVEN though C was fully validated by one node
	// - In the new approach, 2 votes for D are not enough to outweight the
	//   8 implicit votes for C, so nodes will avalanche to C instead
	log.SetFlags(log.Ltime | log.Llongfile)

	sim := newSim()
	dc := &disruptor{}
	sim.collectors = append(sim.collectors, dc.on)

	// Goes A.B.D
	groupABD := sim.createGroup(2)
	// Single node that initially fully validates C before the split
	groupCfast := sim.createGroup(1)
	// Generates C, but fails to fully validate before the split
	groupCsplit := sim.createGroup(7)

	groupNotFastC := groupABD.append(groupCsplit)
	network := groupABD.append(groupCsplit).append(groupCfast)

	delay := LedgerGranularity / 5
	fDelay := LedgerGranularity / 10

	dc.network = network
	dc.groupCfast = groupCfast
	dc.gropuCsplit = groupCsplit
	dc.delay = delay

	network.trust(network)
	// C must have a shorter delay to see all the validations before the
	// other nodes
	network.connect(groupCfast, fDelay)
	// The rest of the network is connected at the same speed
	groupNotFastC.connect(groupNotFastC, delay)

	// Consensus round to generate ledger A
	sim.run(1)
	expect(t, sim.synchronized(&sim.allPeers))

	// Next round generates B and C
	// To force B, we inject an extra transaction in to those nodes
	var id consensus.TxID
	id[0] = 42
	for _, peer := range groupABD.peers {
		peer.txInjections[peer.lastClosedLedger.Seq] = &tx{
			id: id,
		}
	}
	// The Disruptor will ensure that nodes disconnect before the C
	// validations make it to all but the fastC node
	sim.run(1)

	// We are no longer in sync, but have not yet forked:
	// 9 nodes consider A the last fully validated ledger and fastC sees C
	expect(t, !sim.synchronized(&sim.allPeers))
	expect(t, sim.branches(&sim.allPeers) == 1)

	//  Run another round to generate the 8 different C' ledgers
	for _, p := range network.peers {
		p.submit(&tx{
			id: consensus.TxID(p.id),
		})
	}
	sim.run(1)

	// Still not forked
	expect(t, !sim.synchronized(&sim.allPeers))
	expect(t, sim.branches(&sim.allPeers) == 1)

	// Disruptor will reconnect all but the fastC node
	sim.run(1)

	if expect(t, sim.branches(&sim.allPeers) == 1) {
		expect(t, sim.synchronized(&sim.allPeers))
	} else { // old approach caused a fork
		expect(t, sim.branches(groupNotFastC) == 1)
		expect(t, sim.synchronized(groupNotFastC))
	}
}
