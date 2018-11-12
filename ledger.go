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

package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"time"
)

/** A Ledger is a set of observed transactions and a sequence number
  identifying the Ledger.

  Peers in the consensus process are trying to agree on a set of transactions
  to include in a Ledger. For simulation, each transaction is a single
  integer and the Ledger is the set of observed integers. This means future
  Ledgers have prior Ledgers as subsets, e.g.

      Ledger 0 :  {}
      Ledger 1 :  {1,4,5}
      Ledger 2 :  {1,2,4,5,10}
      ....

  Ledgers are immutable value types. All Ledgers with the same sequence
  number, transactions, close time, etc. will have the same Ledger ID. The
  LedgerOracle class below manges ID assignments for a simulation and is the
  only way to close and create a new Ledger. Since the parent Ledger ID is
  part of type, this also means Ledgers with distinct histories will have
  distinct ids, even if they have the same set of transactions, sequence
  number and close time.
*/

// Ledger which is Agreed upon state that consensus transactions will modify
type Ledger struct {
	ParentID            LedgerID
	Seq                 Seq
	Txs                 TxSet
	CloseTimeResolution time.Duration
	CloseTime           time.Time
	ParentCloseTime     time.Time
	CloseTimeAgree      bool
	IndexOf             func(Seq) LedgerID `msgpack:"-"`
	id                  LedgerID           //for test
}

//Genesis is a genesis Ledger.
var Genesis = &Ledger{
	Seq:                 0,
	CloseTimeResolution: LedgerDefaultTimeResolution,
	CloseTimeAgree:      true,
	IndexOf: func(s Seq) LedgerID {
		if s == 0 {
			return GenesisID
		}
		panic("not found" + strconv.Itoa(int(s)))
	},
}

//ID is the ID of the ledger l.
func (l *Ledger) ID() LedgerID {
	if l.Seq == 0 {
		return GenesisID
	}
	if l.id != GenesisID {
		return l.id
	}
	return LedgerID(sha256.Sum256(l.bytes()))
}

//Clone clones the leger l.
func (l *Ledger) Clone() *Ledger {
	l2 := *l
	l2.Txs = l.Txs.Clone()
	return &l2
}

func (l *Ledger) bytes() []byte {
	bs := make([]byte, 8+32+8+8+32+8)
	binary.LittleEndian.PutUint64(bs, uint64(l.Seq))
	id := l.Txs.ID()
	copy(bs[8:], id[:])
	binary.LittleEndian.PutUint64(bs[8+32:], uint64(l.CloseTimeResolution))
	binary.LittleEndian.PutUint64(bs[8+32+8:], uint64(l.CloseTime.Unix()))
	copy(bs[8+32+8+8:], l.ParentID[:])
	binary.LittleEndian.PutUint64(bs[8+32+8+8+32:], uint64(l.ParentCloseTime.Unix()))
	return bs
}

//IsAncestor returns true if a is an ancestor of l.
func (l *Ledger) IsAncestor(a *Ledger) bool {
	if a.Seq < l.Seq {
		return l.IndexOf(a.Seq) == a.ID()
	}
	return false
}
