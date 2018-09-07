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
	"time"
)

var ledgers = make(map[LedgerID]*tledger)

type tledger struct {
	id   LedgerID
	seq  Seq
	prev LedgerID
}

func (l *tledger) ID() LedgerID {
	return l.id
}

func (l *tledger) Seq() Seq {
	return l.seq
}

func (l *tledger) CloseTimeResolution() time.Duration {
	return 0
}

func (l *tledger) CloseAgree() bool {
	return false
}

func (l *tledger) CloseTime() time.Time {
	return time.Time{}
}

func (l *tledger) ParentCloseTime() time.Time {
	return time.Time{}
}

func (l *tledger) IndexOf(s Seq) LedgerID {
	if s == 0 {
		return GenesisID
	}
	for lid := l.id; lid != GenesisID; lid = ledgers[lid].prev {
		if ledgers[lid].Seq() == s {
			return ledgers[lid].id
		}
	}
	panic("not found")
}

func newLedger(path string) *tledger {
	var l *tledger
	pre := GenesisID
	var id LedgerID
	for i, p := range path {
		id[0] = byte(p + 1 - 'a')
		if ll, ok := ledgers[id]; ok {
			l = ll
		} else {
			l = &tledger{
				id:   id,
				seq:  Seq(i + 1),
				prev: pre,
			}
			ledgers[id] = l
		}
		pre = id
	}
	return l
}

//GenesisLedger is the genesis.
var genesisLedger = &tledger{
	id:   GenesisID,
	seq:  0,
	prev: GenesisID,
}
