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
	"time"

	"github.com/AidosKuneen/consensus"
)

type jumpCollectorByNode map[consensus.NodeID]jumpCollector

func (s jumpCollectorByNode) on(id consensus.NodeID, when time.Time, e interface{}) {
	c := s[id]
	c.on(id, when, e)
	s[id] = c
}

type simDurationCollector struct {
	init  bool
	start time.Time
	stop  time.Time
}

func (s *simDurationCollector) on(NodeID, when time.Time, e interface{}) {
	if !s.init {
		s.start = when
		s.init = true
	} else {
		s.stop = when
	}
}

type txCollector struct {
	submitted uint
	accepted  uint
	validated uint
	txs       map[consensus.TxID]*txTracker
	// submitToAccept   *histgram
	// submitToValidate *histgram
}

type txTracker struct {
	t         *tx
	submitted time.Time
	accepted  time.Time
	validated time.Time
}

func (c *txCollector) on(who consensus.NodeID, when time.Time, ee interface{}) {
	switch v := ee.(type) {
	case *submitTx:
		if _, ok := c.txs[v.tx.id]; !ok {
			c.txs[v.tx.id] = &txTracker{
				t:         v.tx,
				submitted: when,
			}
			c.submitted++
		}
	case *acceptLedger:
		for tx := range v.ledger.(*ledger).txs {
			tr, ok := c.txs[tx]
			if ok && !tr.accepted.IsZero() {
				tr.accepted = when
				c.accepted++
				// c.submitToAccept.insert(uint64(tr.accepted.Sub(tr.submitted)))
			}
		}
	case *fullyValidateLedger:
		for tx := range v.ledger.(*ledger).txs {
			tr, ok := c.txs[tx]
			if ok && !tr.validated.IsZero() {
				if tr.accepted.IsZero() {
					panic("not accepted")
				}
				tr.validated = when
				c.validated++
				// c.submitToValidate.insert(uint64(tr.validated.Sub(tr.submitted)))
			}
		}
	}
}
func (c *txCollector) orphaned() int {
	cnt := 0
	for _, it := range c.txs {
		if it.accepted.IsZero() {
			cnt++
		}
	}
	return cnt
}
func (c *txCollector) unvalidated() int {
	cnt := 0
	for _, it := range c.txs {
		if it.validated.IsZero() {
			cnt++
		}
	}
	return cnt
}

type ledgerCollector struct {
	accepted       uint
	fullyValidated uint
	ledgers        map[consensus.LedgerID]*ledgerTracker
	// acceptToFullyValid     *histgram
	// acceptToAccept         *histgram
	// fullyValidToFullyValid *histgram
}

type ledgerTracker struct {
	accepted       time.Time
	fullyValidated time.Time
}

func (c *ledgerCollector) on(who consensus.NodeID, when time.Time, ee interface{}) {
	switch v := ee.(type) {
	case *acceptLedger:
		if _, ok := c.ledgers[v.ledger.ID()]; ok {
			return
		}
		c.ledgers[v.ledger.ID()] = &ledgerTracker{
			accepted: when,
		}
		c.accepted++
		// l := v.ledger.(*ledger)
		// if v.prior.ID() == l.parentID {
		// 	it, ok := c.ledgers[l.parentID]
		// 	if ok {
		// 		c.acceptToAccept.insert(uint64(when.Sub(it.accepted)))
		// 	}
		// }
	case *fullyValidateLedger:
		l := v.ledger.(*ledger)
		if v.prior.ID() == l.parentID {
			tr, ok := c.ledgers[v.ledger.ID()]
			if !ok {
				panic("not found")
			}
			// first time fully validated
			if !tr.fullyValidated.IsZero() {
				c.fullyValidated++
				tr.fullyValidated = when
				// c.acceptToFullyValid.insert(uint64(when.Sub(tr.accepted)))

				parentTr, ok := c.ledgers[l.parentID]
				if ok {
					if !parentTr.fullyValidated.IsZero() {
						// c.fullyValidToFullyValid.insert(
						// 	uint64(when.Sub(parentTr.fullyValidated)))
					}
				}
			}
		}
	}
}

func (c *ledgerCollector) unvalidated() int {
	cnt := 0
	for _, it := range c.ledgers {
		if it.fullyValidated.IsZero() {
			cnt++
		}
	}
	return cnt
}

type jump struct {
	id   consensus.NodeID
	when time.Time
	from *ledger
	to   *ledger
}

type jumpCollector struct {
	closeJumps          []*jump
	fullyValidatedJumps []*jump
}

func (c *jumpCollector) on(who consensus.NodeID, when time.Time, ee interface{}) {
	switch v := ee.(type) {
	case acceptLedger:
		if v.prior.ID() != v.ledger.(*ledger).parentID {
			c.closeJumps = append(c.closeJumps, &jump{
				id:   who,
				when: when,
				from: v.prior.(*ledger),
				to:   v.ledger.(*ledger),
			})
		}
	case *acceptLedger:
		if v.prior.ID() != v.ledger.(*ledger).parentID {
			c.closeJumps = append(c.closeJumps, &jump{
				id:   who,
				when: when,
				from: v.prior.(*ledger),
				to:   v.ledger.(*ledger),
			})
		}
	case fullyValidateLedger:
		if v.prior.ID() != v.ledger.(*ledger).parentID {
			c.fullyValidatedJumps = append(c.fullyValidatedJumps, &jump{
				id:   who,
				when: when,
				from: v.prior.(*ledger),
				to:   v.ledger.(*ledger),
			})
		}
	case *fullyValidateLedger:
		if v.prior.ID() != v.ledger.(*ledger).parentID {
			c.fullyValidatedJumps = append(c.fullyValidatedJumps, &jump{
				id:   who,
				when: when,
				from: v.prior.(*ledger),
				to:   v.ledger.(*ledger),
			})
		}
	}
}
