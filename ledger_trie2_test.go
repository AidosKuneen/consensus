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
	"bytes"
	"log"
	"math/rand"
	"testing"
)

func TestGetPreferred1(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	p := lt.getPreferred(0)
	if p.id != genesisID {
		t.Error("invalid getpreffered")
	}
	p = lt.getPreferred(2)
	if p.id != genesisID {
		t.Error("invalid getpreffered")
	}
}

func TestGetPreferred2(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abc := newLedger("abc")
	lt.insert(abc, 1)
	p := lt.getPreferred(3)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
}
func TestGetPreferred3(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	lt.insert(abc, 1)
	lt.insert(abcd, 1)
	p := lt.getPreferred(3)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
}
func TestGetPreferred4(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	lt.insert(abc, 1)
	lt.insert(abcd, 2)
	p := lt.getPreferred(3)
	if p.id != abcd.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abcd.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
}
func TestGetPreferred5(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	abce := newLedger("abce")
	lt.insert(abc, 1)
	lt.insert(abcd, 1)
	lt.insert(abce, 1)
	p := lt.getPreferred(3)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}

	lt.insert(abc, 1)
	p = lt.getPreferred(3)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
}
func TestGetPreferred6(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	abce := newLedger("abce")
	lt.insert(abc, 1)
	lt.insert(abcd, 2)
	lt.insert(abce, 1)
	p := lt.getPreferred(3)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}

	lt.insert(abcd, 1)
	p = lt.getPreferred(3)
	if p.id != abcd.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abcd.id {
		t.Error("invalid getpreffered", p.seq, abc.seq)
	}
}
func TestGetPreferred7(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abcd := newLedger("abcd")
	abce := newLedger("abce")
	lt.insert(abcd, 2)
	lt.insert(abce, 2)

	if bytes.Compare(abce.id[:], abcd.id[:]) <= 0 {
		t.Fatal("invalid id")
	}

	p := lt.getPreferred(4)
	if p.id != abce.id {
		t.Error("invalid getpreffered", p.seq, abce.seq)
	}
	lt.insert(abcd, 1)

	p = lt.getPreferred(4)
	if p.id != abcd.id {
		t.Error("invalid getpreffered", p.seq, abcd.seq)
	}
}

func TestGetPreferred8(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	abce := newLedger("abce")
	lt.insert(abc, 1)
	lt.insert(abcd, 1)
	lt.insert(abce, 2)

	if bytes.Compare(abce.id[:], abcd.id[:]) <= 0 {
		t.Fatal("invalid id")
	}
	p := lt.getPreferred(3)
	if p.id != abce.id {
		t.Error("invalid getpreffered", p.seq, abce.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abce.id {
		t.Error("invalid getpreffered", p.seq, abce.seq)
	}

	lt.remove(abce, 1)
	lt.insert(abcd, 1)

	p = lt.getPreferred(3)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abcd.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abc.id {
		t.Error("invalid getpreffered", p.seq, abcd.seq)
	}
}

func TestGetPreferred9(t *testing.T) {
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	abcde := newLedger("abcde")
	lt.insert(abc, 1)
	lt.insert(abcd, 2)
	lt.insert(abcde, 4)
	p := lt.getPreferred(3)
	if p.id != abcde.id {
		t.Error("invalid getpreffered", p.seq, abcde.seq)
	}
	p = lt.getPreferred(4)
	if p.id != abcde.id {
		t.Error("invalid getpreffered", p.seq, abcde.seq)
	}
	p = lt.getPreferred(5)
	if p.id != abcde.id {
		t.Error("invalid getpreffered", p.seq, abcde.seq)
	}
}
func TestGetPreferred10(t *testing.T) {
	/** Build the tree below with initial tip support annotated
	        A
	       / \
	    B(1)  C(1)
	   /  |   |
	  H   D   F(1)
	      |
	      E(2)
	      |
	      G
	*/
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	a := newLedger("a")
	ab := newLedger("ab")
	ac := newLedger("ac")
	acf := newLedger("acf")
	abde := newLedger("abde")
	abdeg := newLedger("abdeg")
	abh := newLedger("abh")
	lt.insert(ab, 1)
	lt.insert(ac, 1)
	lt.insert(acf, 1)
	lt.insert(abde, 2)

	p := lt.getPreferred(1)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(2)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(3)
	if p.id != a.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(4)
	if p.id != a.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}

	/** One of E advancing to G doesn't change anything
	        A
	       / \
	    B(1)  C(1)
	   /  |   |
	  H   D   F(1)
	      |
	      E(1)
	      |
	      G(1)
	*/
	lt.remove(abde, 1)
	lt.insert(abdeg, 1)
	p = lt.getPreferred(1)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(2)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(3)
	if p.id != a.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(4)
	if p.id != a.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(5)
	if p.id != a.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}

	/** C advancing to H does advance the seq 3 preferred ledger
	        A
	       / \
	    B(1)  C
	   /  |   |
	  H(1)D   F(1)
	      |
	      E(1)
	      |
	      G(1)
	*/
	lt.remove(ac, 1)
	lt.insert(abh, 1)
	p = lt.getPreferred(1)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(2)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(3)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(4)
	if p.id != a.id {
		t.Fatal("invalid getpreffered", p.seq, a.seq)
	}
	p = lt.getPreferred(5)
	if p.id != a.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}

	/** F advancing to E also moves the preferred ledger forward
	        A
	       / \
	    B(1)  C
	   /  |   |
	  H(1)D   F
	      |
	      E(2)
	      |
	      G(1)
	*/
	lt.remove(acf, 1)
	lt.insert(abde, 1)
	p = lt.getPreferred(1)
	if p.id != abde.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(2)
	if p.id != abde.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(3)
	if p.id != abde.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(4)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
	p = lt.getPreferred(5)
	if p.id != ab.id {
		t.Error("invalid getpreffered", p.seq, ab.seq)
	}
}

// Since the root is a special node that breaks the no-single child
// invariant, do some tests that exercise it.
func TestRootRelated(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	if lt.remove(GenesisLedger, 1) {
		t.Error("invalid removal")
	}
	if lt.branchSupport(GenesisLedger) != 0 {
		t.Error("invalid rootsuport")
	}
	if lt.tipSupport(GenesisLedger) != 0 {
		t.Error("invalid rootsuport")
	}

	a := newLedger("a")
	lt.insert(a, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error(err)
	}
	if lt.branchSupport(GenesisLedger) != 1 {
		t.Error("invalid rootsuport")
	}
	if lt.tipSupport(GenesisLedger) != 0 {
		t.Error("invalid rootsuport")
	}

	e := newLedger("e")
	lt.insert(e, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error(err)
	}
	if lt.branchSupport(GenesisLedger) != 2 {
		t.Error("invalid rootsuport")
	}
	if lt.tipSupport(GenesisLedger) != 0 {
		t.Error("invalid rootsuport")
	}

	if !lt.remove(e, 1) {
		t.Error("invalid removal")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error(err)
	}
	if lt.branchSupport(GenesisLedger) != 1 {
		t.Error("invalid rootsuport")
	}
	if lt.tipSupport(GenesisLedger) != 0 {
		t.Error("invalid rootsuport")
	}
}
func TestStress(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()

	// Test quasi-randomly add/remove supporting for different ledgers
	// from a branching history.

	// Ledgers have sequence 1,2,3,4
	const depth int = 4
	// Each ledger has 4 possible children
	const width int = 4

	const iterations = 10000

	rand.Seed(42)
	for i := 0; i < iterations; i++ {
		curr := ""
		cdepth := rand.Intn(depth)
		offset := 0
		for d := 0; d < cdepth; d++ {
			a := offset + rand.Intn(width)
			curr += string(a)
			offset = (a + 1) * width
		}
		var tl Ledger
		if curr == "" {
			tl = GenesisLedger
		} else {
			tl = newLedger(curr)
		}
		if rand.Intn(2) == 0 {
			lt.insert(tl, 1)
		} else {
			lt.remove(tl, 1)
		}
		if err := lt.checkInvariants(); err != nil {
			t.Error(err)
		}
	}
}
