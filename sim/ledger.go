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
	"log"
	"time"

	"github.com/AidosKuneen/consensus"
)

/** A ledger is a set of observed transactions and a sequence number
  identifying the ledger.

  Peers in the consensus process are trying to agree on a set of transactions
  to include in a ledger. For simulation, each transaction is a single
  integer and the ledger is the set of observed integers. This means future
  ledgers have prior ledgers as subsets, e.g.

      Ledger 0 :  {}
      Ledger 1 :  {1,4,5}
      Ledger 2 :  {1,2,4,5,10}
      ....

  Ledgers are immutable value types. All ledgers with the same sequence
  number, transactions, close time, etc. will have the same ledger ID. The
  LedgerOracle class below manges ID assignments for a simulation and is the
  only way to close and create a new ledger. Since the parent ledger ID is
  part of type, this also means ledgers with distinct histories will have
  distinct ids, even if they have the same set of transactions, sequence
  number and close time.
*/
type ledger struct {
	ancestors []consensus.LedgerID
	*consensus.Ledger
}

var genesis = &ledger{
	Ledger: consensus.Genesis,
}

func (l *ledger) indexOf(s consensus.Seq) consensus.LedgerID {
	if s == l.Seq {
		return l.ID()
	}
	if s > l.Seq {
		panic("not found")
	}
	return l.ancestors[s]
}

type ledgerOracle struct {
	instances map[consensus.LedgerID]*ledger
}

func newLedgerOracle() *ledgerOracle {
	return &ledgerOracle{
		instances: map[consensus.LedgerID]*ledger{
			consensus.Genesis.ID(): genesis,
		},
	}
}

// func (l *ledgerOracle) nextID() consensus.LedgerID {
// 	var id consensus.LedgerID
// 	binary.LittleEndian.PutUint64(id[:], uint64(len(l.instances)+1))
// 	return id
// }

func (l *ledgerOracle) accept(parent *consensus.Ledger, txs consensus.TxSet,
	closeTimeResolution time.Duration, consensusCloseTime time.Time) *ledger {
	next := ledger{
		Ledger: &consensus.Ledger{
			Txs: make(consensus.TxSet),
		},
	}
	next.IndexOf = next.indexOf
	for tid, t := range txs {
		next.Txs[tid] = t
	}
	next.Seq = parent.Seq + 1
	next.CloseTimeResolution = closeTimeResolution
	next.CloseTimeAgree = !consensusCloseTime.IsZero()
	if next.CloseTimeAgree {
		next.CloseTime = consensus.EffCloseTime(
			consensusCloseTime, closeTimeResolution, parent.CloseTime)
	} else {
		next.CloseTime = parent.CloseTime.Add(time.Second)
	}
	next.ParentCloseTime = parent.CloseTime
	next.ParentID = parent.ID()
	p, ok := l.instances[parent.ID()]
	if !ok {
		log.Fatal("not found", parent.ID())
	}
	next.ancestors = make([]consensus.LedgerID, len(p.ancestors)+1)
	copy(next.ancestors, p.ancestors)
	next.ancestors[len(p.ancestors)] = parent.ID()

	for _, l := range l.instances {
		if l.ID() == next.ID() {
			return l
		}
	}
	// next.id = l.nextID()
	l.instances[next.ID()] = &next
	return &next
}

func (l *ledgerOracle) branches(ledgers []*consensus.Ledger) int {
	// Tips always maintains the Ledgers with largest sequence number
	// along all known chains.
	tips := make([]*consensus.Ledger, 0, len(ledgers))

	for _, l := range ledgers {
		// Three options,
		//  1. ledger is on a new branch
		//  2. ledger is on a branch that we have seen tip for
		//  3. ledger is the new tip for a branch
		found := false
		for idx := 0; idx < len(tips) && !found; idx++ {
			idxEarlier := tips[idx].Seq < l.Seq
			earlier := tips[idx]
			later := l
			if !idxEarlier {
				earlier, later = later, earlier
			}
			if later.IsDescendantOf(earlier) {
				tips[idx] = later
				found = true
			}
		}

		if !found {
			tips = append(tips, l)
		}
	}
	// The size of tips is the number of branches
	return len(tips)
}
