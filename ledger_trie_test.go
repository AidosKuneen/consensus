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
	"log"
	"testing"
)

func TestInsert1(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// Single entry by itself
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	lt := newLedgerTrie()
	tl := newLedger("abc")
	lt.insert(tl, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 1 {
		t.Error("invalid branchsupport")
	}
	lt.insert(tl, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 2 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 2 {
		t.Error("invalid branchsupport")
	}
}
func TestInsert2(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// Suffix of existing (extending tree)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()
	tl := newLedger("abc")
	lt.insert(tl, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	tl2 := newLedger("abcd")
	lt.insert(tl2, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport")
	}

	tl3 := newLedger("abce")
	lt.insert(tl3, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 3 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl2) != 1 {
		t.Log(lt.tipSupport(tl2))
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport")
	}
	if lt.tipSupport(tl3) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 1 {
		t.Error("invalid branchsupport")
	}
}
func TestInsert3(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// uncommitted of existing node
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()
	tl := newLedger("abcd")
	lt.insert(tl, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	tl2 := newLedger("abcdf")
	lt.insert(tl2, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport")
	}

	tl3 := newLedger("abc")
	lt.insert(tl3, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl3) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 3 {
		t.Error("invalid branchsupport")
	}
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport")
	}
}
func TestInsert4(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// Suffix + uncommitted of existing node
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()
	tl := newLedger("abcd")
	lt.insert(tl, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	tl2 := newLedger("abce")
	lt.insert(tl2, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	tl3 := newLedger("abc")
	if lt.tipSupport(tl3) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 1 {
		t.Error("invalid branchsupport")
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport")
	}
}

// Suffix + uncommitted with existing child
func TestInsert5(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()
	tl := newLedger("abcd")
	lt.insert(tl, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	tl2 := newLedger("abcde")
	lt.insert(tl2, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	tl3 := newLedger("abcf")
	lt.insert(tl3, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}

	tl4 := newLedger("abc")
	if lt.tipSupport(tl4) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl4) != 3 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 2 {
		t.Error("invalid branchsupport")
	}
	if lt.tipSupport(tl3) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 1 {
		t.Error("invalid branchsupport")
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport")
	}
}

// Multiple counts
func TestInsert6(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()
	tl := newLedger("ab")
	lt.insert(tl, 4)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 4 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 4 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}

	tl2 := newLedger("abc")
	lt.insert(tl2, 2)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl2) != 2 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl) != 4 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 6 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}

	tl3 := newLedger("a")
	if lt.tipSupport(tl3) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 6 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
}

// Not in trie
func TestRemove1(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("abc")
	lt.insert(tl, 1)
	tl2 := newLedger("ab")
	if lt.remove(tl2, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	tl3 := newLedger("a")
	if lt.remove(tl3, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
}

// In trie but with 0 tip support
func TestRemove2(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("abcd")
	lt.insert(tl, 1)
	tl2 := newLedger("abce")
	lt.insert(tl2, 1)

	tl3 := newLedger("abc")
	if lt.tipSupport(tl3) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}

	if lt.remove(tl3, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl3) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
}

// In trie with > 1 tip support
func TestRemove3(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("abc")
	lt.insert(tl, 2)
	if lt.tipSupport(tl) != 2 {
		t.Error("invalid tipsupport")
	}
	if !lt.remove(tl, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}

	lt.insert(tl, 1)
	if lt.tipSupport(tl) != 2 {
		t.Error("invalid tipsupport")
	}
	if !lt.remove(tl, 2) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 0 {
		t.Error("invalid tipsupport")
	}

	lt.insert(tl, 3)
	if lt.tipSupport(tl) != 3 {
		t.Error("invalid tipsupport")
	}
	if !lt.remove(tl, 300) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 0 {
		t.Error("invalid tipsupport")
	}
}

func TestRemove4(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("ab")
	lt.insert(tl, 1)
	tl2 := newLedger("abc")
	lt.insert(tl2, 1)
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}

	if !lt.remove(tl2, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 1 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl2) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 0 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
}

func TestRemove5(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("ab")
	lt.insert(tl, 1)
	tl2 := newLedger("abc")
	lt.insert(tl2, 1)
	tl3 := newLedger("abcd")
	lt.insert(tl3, 1)

	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl3) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 1 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}

	if !lt.remove(tl2, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl2) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 1 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
	if lt.tipSupport(tl3) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl3) != 1 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
}

func TestRemove6(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("ab")
	lt.insert(tl, 1)
	tl2 := newLedger("abc")
	lt.insert(tl2, 1)
	tl3 := newLedger("abcd")
	lt.insert(tl3, 1)
	tl4 := newLedger("abce")
	lt.insert(tl4, 1)

	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 3 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}

	if !lt.remove(tl2, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl2) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl2) != 2 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
}

// In trie with = 1 tip support, parent compaction
func TestRemove7(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("ab")
	lt.insert(tl, 1)
	tl2 := newLedger("abc")
	lt.insert(tl2, 1)
	tl3 := newLedger("abd")
	lt.insert(tl3, 1)
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if !lt.remove(tl, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(tl3) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(tl) != 0 {
		t.Error("invalid tipsupport")
	}

	if !lt.remove(tl3, 1) {
		t.Error("invalid remove")
	}
	if err := lt.checkInvariants(); err != nil {
		t.Error("invalid ledgertrie", err)
	}
	if lt.tipSupport(tl2) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 1 {
		t.Error("invalid branchsupport", lt.branchSupport(tl))
	}
}

// In trie with = 1 tip support, parent compaction
func TestSupport(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	// Single entry by itself
	lt := newLedgerTrie()

	tl := newLedger("a")
	if lt.tipSupport(tl) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 0 {
		t.Error("invalid tipsupport")
	}
	tl = newLedger("axy")
	if lt.tipSupport(tl) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(tl) != 0 {
		t.Error("invalid tipsupport")
	}

	a := newLedger("a")
	ab := newLedger("ab")
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	lt.insert(abc, 1)
	if lt.tipSupport(a) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(ab) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(abc) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(abcd) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(a) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(ab) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(abc) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(abcd) != 0 {
		t.Error("invalid tipsupport")
	}

	abe := newLedger("abe")
	lt.insert(abe, 1)
	if lt.tipSupport(a) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(ab) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(abc) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(abe) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(a) != 2 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(ab) != 2 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(abc) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(abe) != 1 {
		t.Error("invalid tipsupport")
	}

	lt.remove(abc, 1)
	if lt.tipSupport(a) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(ab) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(abc) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.tipSupport(abe) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(a) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(ab) != 1 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(abc) != 0 {
		t.Error("invalid tipsupport")
	}
	if lt.branchSupport(abe) != 1 {
		t.Error("invalid tipsupport")
	}
}
