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
	"bytes"
	"crypto/sha256"
	"sort"
)

// TxSet is a set of transactions
type TxSet map[TxID]TxT

//NewTxSet returns a new TxSet.
func NewTxSet() TxSet {
	return make(TxSet)
}

//Clone clones the TxSet
func (t TxSet) Clone() TxSet {
	t2 := NewTxSet()
	for k, v := range t {
		t2[k] = v
	}
	return t2
}

//ID returns the id of the TxSet t.
func (t TxSet) ID() TxSetID {
	ids := make([]TxID, 0, len(t))
	for id := range t {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return bytes.Compare(ids[i][:], ids[j][:]) < 0
	})
	h := sha256.New()
	for _, id := range ids {
		h.Write(id[:])
	}
	var id TxSetID
	copy(id[:], h.Sum(nil))
	return id
}

// Return set of transactions that are not common to this set or other
// boolean indicates which set it was in
// If true I have the tx, otherwiwse o has it.
func (t TxSet) compare(o TxSet) map[TxID]bool {
	r := make(map[TxID]bool)
	for _, tt := range t {
		if _, ok := o[tt.ID()]; !ok {
			r[tt.ID()] = true
		}
	}
	for _, tt := range o {
		if _, ok := t[tt.ID()]; !ok {
			r[tt.ID()] = false
		}
	}
	return r
}
