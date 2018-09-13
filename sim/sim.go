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
	"encoding/binary"
	"time"

	"github.com/AidosKuneen/consensus"
)

type mclock struct {
	now time.Time
}

func (c *mclock) Now() time.Time {
	return c.now
}

type sim struct {
	scheduler  *scheduler
	oracle     *ledgerOracle
	net        *basicNetwork
	trustGraph *trustGraph
	collectors collectorRef
	peers      []*peer
	allPeers   peerGroup
}

//NewSim creates a sim env.
func newSim() *sim {
	sch := &scheduler{
		clock: &mclock{
			now: time.Unix(86400, 0), //changed from origin
		},
	}
	s := &sim{
		oracle: newLedgerOracle(),
		net: &basicNetwork{
			links: &digraph{
				graph: make(graphT),
			},
			sch: sch,
		},
		trustGraph: &trustGraph{
			graph: digraph{
				graph: make(graphT),
			},
		},
		scheduler: sch,
	}
	return s
}

func (s *sim) createGroup(numPeers int) *peerGroup {
	newPeers := make([]*peer, numPeers)
	for i := 0; i < numPeers; i++ {
		var id consensus.NodeID
		binary.LittleEndian.PutUint64(id[:], uint64(len(s.peers)))
		p := newPeer(id, s.scheduler, s.oracle, s.net, s.collectors, s.trustGraph)
		s.peers = append(s.peers, p)
		newPeers[i] = p
	}
	res := &peerGroup{
		peers: newPeers,
	}

	s.allPeers.peers = append(s.allPeers.peers, newPeers...)
	s.allPeers.sort()
	return res
}

func (s *sim) run(ledgers int) {
	for _, p := range s.peers {
		p.targetLedgers = p.completedLedgers + ledgers
		p.start()
	}
	s.scheduler.step()
}

func (s *sim) synchronized(g *peerGroup) bool {
	if len(g.peers) < 1 {
		return true
	}
	ref := g.peers[0]
	sync := true
	for _, p := range g.peers {
		if p.lastClosedLedger.ID() != ref.lastClosedLedger.ID() ||
			p.fullyValidatedLedger.ID() != ref.fullyValidatedLedger.ID() {
			sync = false
		}
	}
	return sync
}
func (s *sim) branches(g *peerGroup) int {
	if len(g.peers) < 1 {
		return 0
	}
	ledgers := make(map[consensus.LedgerID]struct{})
	ls := make([]*consensus.Ledger, 0, len(g.peers))
	for _, peer := range g.peers {
		if _, ok := ledgers[peer.fullyValidatedLedger.ID()]; !ok {
			ledgers[peer.fullyValidatedLedger.ID()] = struct{}{}
			ls = append(ls, peer.fullyValidatedLedger)
		}
	}
	return s.oracle.branches(ls)
}
