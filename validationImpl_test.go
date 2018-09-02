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

package consensus

import (
	"encoding/binary"
	"time"
)

type validationT struct {
	id       LedgerID
	seq      Seq
	signTime time.Time
	seenTime time.Time
	key      [40]byte
	nodeID   NodeID
	trusted  bool
	full     bool
	loadFee  uint32
}

func (v *validationT) bytes() []byte {
	bs := make([]byte, 32+8+8+8+40+32+4+1)
	copy(bs, v.id[:])
	binary.LittleEndian.PutUint64(bs[32:], uint64(v.seq))
	binary.LittleEndian.PutUint64(bs[40:], uint64(v.signTime.Unix()))
	binary.LittleEndian.PutUint64(bs[48:], uint64(v.seenTime.Unix()))
	copy(bs[56:], v.key[:])
	copy(bs[96:], v.nodeID[:])
	binary.LittleEndian.PutUint32(bs[126:], v.loadFee)
	if v.full {
		bs[130] = 1
	}
	return bs
}

// Ledger ID associated with this validation
func (v *validationT) LedgerID() LedgerID {
	return v.id
}

// Sequence number of validation's ledger (0 means no sequence number)
func (v *validationT) Seq() Seq {
	return v.seq
}

// When the validation was signed
func (v *validationT) SignTime() time.Time {
	return v.signTime
}

// When the validation was first observed by this node
func (v *validationT) SeenTime() time.Time {
	return v.seenTime

}

// Whether the publishing node was Trusted at the time the validation
// arrived
func (v *validationT) Trusted() bool {
	return v.trusted

}

// Set the validation as trusted
func (v *validationT) SetTrusted() {
	v.trusted = true

}

// Set the validation as untrusted
func (v *validationT) SetUntrusted() {
	v.trusted = false

}

// Whether this is a Full or partial validation
func (v *validationT) Full() bool {
	return v.full

}

func (v *validationT) LoadFee() uint32 {
	return v.loadFee
}
