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
	"bytes"
	"encoding/binary"
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
	id                  consensus.LedgerID
	parentID            consensus.LedgerID
	seq                 consensus.Seq
	ancestors           []consensus.LedgerID
	txs                 consensus.TxSet
	closeTimeResolution time.Duration
	closeTime           time.Time
	closeTimeAgree      bool
	parentCloseTime     time.Time
}

var genesis = &ledger{
	id:                  consensus.GenesisID,
	closeTimeResolution: consensus.LedgerDefaultTimeResolution,
	txs:                 make(consensus.TxSet),
	closeTimeAgree:      true,
}

func (l *ledger) clone() *ledger {
	l2 := *l
	l2.ancestors = make([]consensus.LedgerID, len(l.ancestors))
	for i, a := range l.ancestors {
		l2.ancestors[i] = a
	}
	l2.txs = l.txs.Clone()
	return &l2
}

func (l *ledger) bytes() []byte {
	bs := make([]byte, 8+32+8+8+1+32+8)
	binary.LittleEndian.PutUint64(bs, uint64(l.seq))
	id := l.txs.ID()
	copy(bs[8:], id[:])
	binary.LittleEndian.PutUint64(bs[8+32:], uint64(l.closeTimeResolution))
	binary.LittleEndian.PutUint64(bs[8+32+8:], uint64(l.closeTime.Unix()))
	if l.closeTimeAgree {
		bs[8+32+8+8] = 1
	}
	copy(bs[8+32+8+8+1:], l.parentID[:])
	binary.LittleEndian.PutUint64(bs[8+32+8+8+1+32:], uint64(l.parentCloseTime.Unix()))
	return bs
}
func (l *ledger) equals(l2 *ledger) bool {
	return bytes.Equal(l.bytes(), l2.bytes())
}
func (l *ledger) less(l2 *ledger) bool {
	return bytes.Compare(l.bytes(), l2.bytes()) < 0
}
func (l *ledger) ID() consensus.LedgerID {
	return l.id
}

func (l *ledger) Seq() consensus.Seq {
	return l.seq
}

func (l *ledger) CloseTimeResolution() time.Duration {
	return l.closeTimeResolution
}

func (l *ledger) CloseAgree() bool {
	return l.closeTimeAgree
}

func (l *ledger) CloseTime() time.Time {
	return l.closeTime
}

func (l *ledger) ParentCloseTime() time.Time {
	return l.parentCloseTime
}

func (l *ledger) IndexOf(s consensus.Seq) consensus.LedgerID {
	if s == l.seq {
		return l.id
	}
	if s > l.seq {
		panic("not found")
	}
	return l.ancestors[s]
}
func (l *ledger) isAncestor(a *ledger) bool {
	if a.seq < l.seq {
		return l.IndexOf(a.seq) == a.id
	}
	return false
}
func mismatch(a, b *ledger) consensus.Seq {
	var start consensus.Seq
	end := a.seq + 1
	if end > b.seq {
		end = b.seq
	}
	count := end - start
	for count > 0 {
		step := count / 2
		curr := start + step
		if a.IndexOf(curr) == b.IndexOf(curr) {
			curr++
			start -= curr
			count -= step + 1
		} else {
			count = step
		}
	}
	return start
}

type ledgerOracle struct {
	instances map[consensus.LedgerID]*ledger
}

func (l *ledgerOracle) nextID() consensus.LedgerID {
	var id consensus.LedgerID
	binary.LittleEndian.PutUint64(id[:], uint64(len(l.instances)+1))
	return id
}

func (l *ledgerOracle) accept(parent *ledger, txs consensus.TxSet,
	closeTimeResolution time.Duration, consensusCloseTime time.Time) *ledger {
	next := ledger{
		txs: make(consensus.TxSet),
	}
	for tid, t := range txs {
		next.txs[tid] = t
	}
	next.seq = parent.seq + 1
	next.closeTimeResolution = closeTimeResolution
	next.closeTimeAgree = !consensusCloseTime.IsZero()
	if next.closeTimeAgree {
		next.closeTime = consensus.EffCloseTime(
			consensusCloseTime, closeTimeResolution, parent.closeTime)
	} else {
		next.closeTime = parent.closeTime.Add(time.Second)
	}
	next.parentCloseTime = parent.closeTime
	next.parentID = parent.id
	next.ancestors = make([]consensus.LedgerID, len(parent.ancestors)+1)
	copy(next.ancestors, parent.ancestors)
	next.ancestors[len(parent.ancestors)] = parent.id

	for _, l := range l.instances {
		if bytes.Equal(l.bytes(), next.bytes()) {
			return l
		}
	}
	next.id = l.nextID()
	l.instances[next.id] = &next
	return &next
}

func (l *ledgerOracle) accept2(curr *ledger, t *tx) *ledger {
	return l.accept(
		curr,
		consensus.TxSet{
			t.id: t,
		},
		curr.closeTimeResolution,
		curr.closeTime.Add(time.Second))
}
func (l *ledgerOracle) lookup(id consensus.LedgerID) *ledger {
	return l.instances[id]
}

func (l *ledgerOracle) branches(ledgers []*ledger) int {
	// Tips always maintains the Ledgers with largest sequence number
	// along all known chains.
	tips := make([]*ledger, 0, len(ledgers))

	for _, l := range ledgers {
		// Three options,
		//  1. ledger is on a new branch
		//  2. ledger is on a branch that we have seen tip for
		//  3. ledger is the new tip for a branch
		found := false
		for idx := 0; idx < len(tips) && !found; idx++ {
			idxEarlier := tips[idx].seq < l.seq
			earlier := tips[idx]
			later := l
			if !idxEarlier {
				earlier, later = later, earlier
			}
			if later.isAncestor(earlier) {
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

type ledgerHistoryHelper struct {
	oracle  *ledgerOracle
	nextTx  uint64
	ledgers map[string]*ledger
	seen    map[byte]struct{}
}

func newLedgerHistoryHelper() *ledgerHistoryHelper {
	return &ledgerHistoryHelper{
		ledgers: make(map[string]*ledger),
	}
}

/** Get or create the ledger with the given string history.

  Creates any necessary intermediate ledgers, but asserts if
  a letter is re-used (e.g. "abc" then "adc" would assert)
*/
func (l *ledgerHistoryHelper) create(s string) *ledger {
	for k, v := range l.ledgers {
		if k == s {
			return v
		}
	}
	if _, ok := l.seen[s[len(s)-1]]; !ok {
		panic("found")
	}
	l.seen[s[len(s)-1]] = struct{}{}
	parent := l.create(s[:len(s)-1])
	l.nextTx++
	var tid consensus.TxID
	binary.LittleEndian.PutUint64(tid[:], l.nextTx)
	ll := l.oracle.accept2(parent, &tx{
		id: tid,
	})
	l.ledgers[s] = ll
	return ll
}
