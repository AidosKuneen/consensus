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
	"encoding/binary"
	"time"
)

type agedMap struct {
	clock   clock
	ledgers map[LedgerID]*nodeVal
}

func newAgedMap(c clock) *agedMap {
	return &agedMap{
		clock:   c,
		ledgers: make(map[LedgerID]*nodeVal),
	}
}

type nodeVal struct {
	m       map[NodeID]*Validation
	touched time.Time
}

func (a *agedMap) add(lid LedgerID, nid NodeID, v *Validation) {
	if m, ok := a.ledgers[lid]; ok {
		m.m[nid] = v
		return
	}
	a.ledgers[lid] = &nodeVal{
		m:       make(map[NodeID]*Validation),
		touched: a.clock.Now(),
	}
	a.ledgers[lid].m[nid] = v
}

func (a *agedMap) expire(expire time.Duration) {
	for k, nv := range a.ledgers {
		if nv.touched.Add(expire).Before(a.clock.Now()) {
			delete(a.ledgers, k)
		}
	}
}

func (a *agedMap) touch(nv *nodeVal) {
	nv.touched = a.clock.Now()
}

func (a *agedMap) loop(id LedgerID, pre func(int), f func(NodeID, *Validation)) {
	nv, ok := a.ledgers[id]
	if !ok {
		return
	}
	a.touch(nv)
	pre(len(nv.m))
	for k, v := range nv.m {
		f(k, v)
	}
}

//---

func toSeqLedgerID(s Seq, lid LedgerID) seqLedgerID {
	var r seqLedgerID
	binary.LittleEndian.PutUint64(r[:], uint64(s))
	copy(r[8:], lid[:])
	return r
}

func fromSeqLedgerID(sl seqLedgerID) (Seq, LedgerID) {
	s := binary.LittleEndian.Uint64(sl[:8])
	var lid LedgerID
	copy(lid[:], sl[8:])
	return Seq(s), lid
}
