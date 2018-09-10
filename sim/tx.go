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
	"crypto/sha256"
	"errors"
	"sort"

	"github.com/AidosKuneen/consensus"
)

// TxT is a single transaction
type tx struct {
	id consensus.TxID
}

func (t *tx) ID() consensus.TxID {
	return t.id
}
func (t *tx) less(t2 *tx) bool {
	return bytes.Compare(t.id[:], t2.id[:]) < 0
}
func (t *tx) equal(t2 *tx) bool {
	return bytes.Equal(t.id[:], t2.id[:])
}

// TxSet is A set of transactions
type txSet struct {
	id  consensus.TxSetID
	txs map[consensus.TxID]*tx
}

func newTxSet(id consensus.TxSetID) *txSet {
	return &txSet{
		id:  id,
		txs: make(map[consensus.TxID]*tx),
	}
}

func newTxSetFromTxSetType(set txSetType) *txSet {
	ts := &txSet{
		txs: make(map[consensus.TxID]*tx),
	}
	for k, v := range set {
		ts.txs[k] = v
	}
	ts.setID()
	return ts
}

func (t *txSet) setID() {
	ids := make([]consensus.TxID, 0, len(t.txs))
	for id := range t.txs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return bytes.Compare(ids[i][:], ids[j][:]) < 0
	})
	h := sha256.New()
	for _, id := range ids {
		h.Write(id[:])
	}
	copy(t.id[:], h.Sum(nil))
}

func (t *txSet) Exists(t2 consensus.TxID) bool {
	_, ok := t.txs[t2]
	return ok
}

// Return value should have semantics like Tx const *
func (t *txSet) Find(t2 consensus.TxID) (consensus.TxT, error) {
	tx, ok := t.txs[t2]
	if !ok {
		return nil, errors.New("not found")
	}
	return tx, nil
}
func (t *txSet) Clone() consensus.TxSet {
	t2 := &txSet{
		id:  t.id,
		txs: make(map[consensus.TxID]*tx),
	}
	for k, v := range t.txs {
		t2.txs[k] = v
	}
	return t2
}

func (t *txSet) ID() consensus.TxSetID {
	return t.id
}

// Return set of transactions that are not common to this set or other
// boolean indicates which set it was in
// If true I have the tx, otherwiwse o has it.
func (t *txSet) Compare(o consensus.TxSet) map[consensus.TxID]bool {
	r := make(map[consensus.TxID]bool)
	for _, tt := range t.txs {
		if _, err := o.Find(tt.ID()); err != nil {
			r[tt.ID()] = true
		}
	}
	oo := o.(*txSet)
	for _, tt := range oo.txs {
		if _, err := t.Find(tt.ID()); err != nil {
			r[tt.ID()] = false
		}
	}
	return r
}

func (t *txSet) Insert(t2 consensus.TxT) bool {
	tt := t2.(*tx)
	_, ok := t.txs[t2.ID()]
	t.txs[t2.ID()] = tt
	t.setID()
	return !ok
}
func (t *txSet) Erase(t2 consensus.TxID) bool {
	_, ok := t.txs[t2]
	delete(t.txs, t2)
	t.setID()
	return ok
}
