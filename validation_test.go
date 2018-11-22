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
	"sort"
	"testing"
	"time"
)

// func toNetClock() time.Time {
// 	return time.Now().Add(86400 * time.Second)
// }

type nodeT struct {
	nodeID  [32]byte
	trusted bool
	signIdx uint32
	loadFee uint32
}

func newNodeT(peerID NodeID) *nodeT {
	return &nodeT{
		trusted: true,
		nodeID:  peerID,
		signIdx: 1,
	}
}

func (n *nodeT) validate(id LedgerID, seq Seq, signOffset, seenOffset time.Duration, full bool) *Validation {
	return &Validation{
		LedgerID: id,
		Seq:      seq,
		SignTime: time.Now().Add(signOffset),
		SeenTime: time.Now().Add(seenOffset),
		// key:      n.currKey(),
		NodeID:  n.nodeID,
		Full:    full,
		Fee:     n.loadFee,
		Trusted: n.trusted,
	}
}
func (n *nodeT) validate2(l *Ledger, signOffset, seenOffset time.Duration) *Validation {
	return n.validate(l.ID(), l.Seq, signOffset, seenOffset, true)
}

func (n *nodeT) validate3(l *Ledger) *Validation {
	return n.validate(l.ID(), l.Seq, 0, 0, true)
}

func (n *nodeT) partial(l *Ledger) *Validation {
	return n.validate(l.ID(), l.Seq, 0, 0, false)
}

type staleData struct {
	stale   []*Validation
	flushed map[NodeID]*Validation
}

type testHarness struct {
	staledata  *staleData
	vals       *Validations
	nextNodeID NodeID
	clock      *mclock
}

type mclock struct {
}

func (c *mclock) Now() time.Time {
	return time.Now()
}
func newTestHarness() *testHarness {
	sd := staleData{
		flushed: make(map[NodeID]*Validation),
	}
	mc := &mclock{}
	vs := NewValidations(&adaptorT{
		staleData: &sd,
	}, mc)
	return &testHarness{
		vals:      vs,
		staledata: &sd,
		clock:     mc,
	}
}

func (h *testHarness) add(v *Validation) ValStatus {
	return h.vals.Add(v.NodeID, v)
}

func (h *testHarness) makeNode() *nodeT {
	h.nextNodeID[0]++
	return newNodeT(h.nextNodeID)
}

func TestAddValidation1(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ab := newLedger("ab")
	// az := newLedger("az")
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	abcde := newLedger("abcde")

	harness := newTestHarness()
	n := harness.makeNode()
	v := n.validate3(a)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid add")
	}
	if s := harness.add(v); s != VstatBadSeq {
		t.Error("invalid add", s)
	}
	time.Sleep(time.Second)
	if len(harness.staledata.stale) != 0 {
		t.Error("stale is not empty")
	}
	v = n.validate3(ab)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid add")
	}
	if len(harness.staledata.stale) != 1 {
		t.Error("incorrect stale")
	}
	if harness.staledata.stale[0].LedgerID != a.ID() {
		t.Error("invalid stale")
	}
	if harness.vals.NumTrustedForLedger(ab.ID()) != 1 {
		t.Error("invalid")
	}
	if harness.vals.NumTrustedForLedger(abc.ID()) != 0 {
		t.Error("invalid")
	}
	n.signIdx++
	time.Sleep(time.Second)

	v = n.validate3(ab)
	if harness.add(v) != VstatBadSeq {
		t.Error("invalid add")
	}
	if harness.add(v) != VstatBadSeq {
		t.Error("invalid add")
	}
	time.Sleep(time.Second)

	v = n.validate3(abc)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.vals.NumTrustedForLedger(ab.ID()) != 1 {
		t.Error("invalid")
	}
	if harness.vals.NumTrustedForLedger(abc.ID()) != 1 {
		t.Error("invalid")
	}

	time.Sleep(2 * time.Second)

	vABCDE := n.validate3(abcde)

	time.Sleep(4 * time.Second)
	vABCD := n.validate3(abcd)

	if harness.add(vABCD) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(vABCDE) != VstatStale {
		t.Error("invalid add")
	}
}

func TestAddValidation2(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ab := newLedger("ab")
	abc := newLedger("abc")

	harness := newTestHarness()
	n := harness.makeNode()
	v := n.validate3(a)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid add")
	}
	v = n.validate2(ab, -time.Second, -time.Second)
	if s := harness.add(v); s != VstatStale {
		t.Error("invalid add", s)
	}
	v = n.validate2(abc, time.Second, time.Second)
	if s := harness.add(v); s != VstatCurrent {
		t.Error("invalid add", s)
	}
}

func TestAddValidation3(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")

	harness := newTestHarness()
	n := harness.makeNode()
	v := n.validate2(a, -validationCurrentEarly-time.Second, 0)
	if harness.add(v) != VstatStale {
		t.Error("invalid add")
	}

	v = n.validate2(a, validationCurrentWall+time.Second, 0)
	if s := harness.add(v); s != VstatStale {
		t.Error("invalid add", s)
	}
	v = n.validate2(a, 0, validationCurrentLocal+time.Second)
	if s := harness.add(v); s != VstatStale {
		t.Error("invalid add", s)
	}
}
func TestAddValidation4(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	ab := newLedger("ab")
	abc := newLedger("abc")
	az := newLedger("az")

	for _, doFull := range []bool{true, false} {
		harness := newTestHarness()
		n := harness.makeNode()
		process := func(lgr *Ledger) ValStatus {
			if doFull {
				return harness.add(n.validate3(lgr))
			}
			return harness.add(n.partial(lgr))
		}
		if process(abc) != VstatCurrent {
			t.Error("invalid add")
		}
		time.Sleep(time.Second)
		if ab.Seq >= abc.Seq {
			t.Error("invalid seq")
		}
		if process(ab) != VstatBadSeq {
			t.Error("invalid add")
		}
		if process(az) != VstatBadSeq {
			t.Error("invalid add")
		}
		b := validationSetExpires
		validationSetExpires = 5 * time.Second
		time.Sleep(validationSetExpires + time.Millisecond)
		if process(az) != VstatCurrent {
			t.Error("invalid add")
		}
		validationSetExpires = b
	}
}

func TestOnStale(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ab := newLedger("ab")
	triggers := []func(*Validations){
		func(vals *Validations) {
			vals.CurrentTrusted()
		},
		func(vals *Validations) {
			vals.GetCurrentNodeIDs()
		},
		func(vals *Validations) {
			vals.GetPreferred(Genesis)
		},
		func(vals *Validations) {
			vals.GetNodesAfter(a, a.ID())
		},
	}
	for i, tr := range triggers {
		harness := newTestHarness()
		n := harness.makeNode()
		if harness.add(n.validate3(ab)) != VstatCurrent {
			t.Error("invalid add")
		}
		tr(harness.vals)
		if harness.vals.GetNodesAfter(a, a.ID()) != 1 {
			t.Fatal("invalid getnodesafetr", i)
		}
		s, id := harness.vals.GetPreferred(Genesis)
		if s != ab.Seq || id != ab.ID() {
			t.Error("invalid getPreferred")
		}
		if len(harness.staledata.stale) != 0 {
			t.Error("stale is not zero")
		}
		b := validationCurrentLocal
		validationCurrentLocal = 5 * time.Second
		time.Sleep(validationCurrentLocal)
		tr(harness.vals)
		if len(harness.staledata.stale) != 1 {
			t.Fatal("invalid stale", len(harness.staledata.stale), i)
		}
		if harness.staledata.stale[0].LedgerID != ab.ID() {
			t.Fatal("invalid stale")
		}
		if harness.vals.GetNodesAfter(a, a.ID()) != 0 {
			t.Error("invalid getnodeafter")
		}
		s, id = harness.vals.GetPreferred(Genesis)
		if s != 0 || id != Genesis.ID() {
			t.Error("invalid getPreferred", s, id)
		}
		validationCurrentLocal = b
	}
}
func TestGetNodesAfter(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ab := newLedger("ab")
	abc := newLedger("abc")
	ad := newLedger("ad")
	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nc := harness.makeNode()
	nd := harness.makeNode()
	nc.trusted = false
	if harness.add(na.validate3(a)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(nb.validate3(a)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(nc.validate3(a)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(nd.partial(a)) != VstatCurrent {
		t.Error("invalid add")
	}
	for _, l := range []*Ledger{a, ab, abc, ad} {
		if harness.vals.GetNodesAfter(l, l.ID()) != 0 {
			t.Error("invalid")
		}
	}
	time.Sleep(5 * time.Second)
	if harness.add(na.validate3(ab)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(nb.validate3(abc)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(nc.validate3(ab)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(nd.partial(abc)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.vals.GetNodesAfter(a, a.ID()) != 3 {
		t.Error("invalid")
	}
	if harness.vals.GetNodesAfter(ab, ab.ID()) != 2 {
		t.Error("invalid")
	}
	if harness.vals.GetNodesAfter(abc, abc.ID()) != 0 {
		t.Error("invalid")
	}
	if harness.vals.GetNodesAfter(ad, ad.ID()) != 0 {
		t.Error("invalid")
	}

	// If given a ledger inconsistent with the id, is still able to check
	// using slower method
	if harness.vals.GetNodesAfter(ad, a.ID()) != 1 {
		t.Error("invalid")
	}
	if harness.vals.GetNodesAfter(ad, ab.ID()) != 2 {
		t.Error("invalid")
	}
}
func TestGetCurrentTrusted(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	b := newLedger("b")
	ac := newLedger("ac")
	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nb.trusted = false

	if harness.add(na.validate3(a)) != VstatCurrent {
		t.Error("invalid add")
	}
	if harness.add(nb.validate3(b)) != VstatCurrent {
		t.Error("invalid add")
	}
	if len(harness.vals.CurrentTrusted()) != 1 {
		t.Error("invalid")
	}
	tr := harness.vals.CurrentTrusted()
	if tr[0].LedgerID != a.ID() {
		t.Error("invalid")
	}
	if tr[0].Seq != a.Seq {
		t.Error("invalid")
	}
	time.Sleep(3 * time.Second)
	for _, n := range []*nodeT{na, nb} {
		if harness.add(n.validate3(ac)) != VstatCurrent {
			t.Error("invalid add")
		}
	}
	if len(harness.vals.CurrentTrusted()) != 1 {
		t.Error("invalid")
	}
	tr = harness.vals.CurrentTrusted()
	if tr[0].LedgerID != ac.ID() {
		t.Error("invalid")
	}
	if tr[0].Seq != ac.Seq {
		t.Error("invalid")
	}
	vc := validationCurrentLocal
	validationCurrentLocal = 5 * time.Second
	time.Sleep(validationCurrentLocal)
	if len(harness.vals.CurrentTrusted()) != 0 {
		t.Error("invalid")
	}
	validationCurrentLocal = vc
}
func TestGetCurrentPublicKeys(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ac := newLedger("ac")
	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nb.trusted = false

	for _, n := range []*nodeT{na, nb} {
		if harness.add(n.validate3(a)) != VstatCurrent {
			t.Error("invalid add")
		}
	}

	ns := harness.vals.GetCurrentNodeIDs()
	if _, ok := ns[na.nodeID]; !ok {
		t.Error("invalid")
	}
	if _, ok := ns[nb.nodeID]; !ok {
		t.Error("invalid")
	}
	if len(ns) != 2 {
		t.Error("invalid")
	}
	time.Sleep(3 * time.Second)
	na.signIdx++
	nb.signIdx++

	for _, n := range []*nodeT{na, nb} {
		if harness.add(n.validate3(ac)) != VstatCurrent {
			t.Error("invalid add")
		}
	}
	ns = harness.vals.GetCurrentNodeIDs()
	if _, ok := ns[na.nodeID]; !ok {
		t.Error("invalid")
	}
	if _, ok := ns[nb.nodeID]; !ok {
		t.Error("invalid")
	}
	if len(ns) != 2 {
		t.Error("invalid")
	}
	vc := validationCurrentLocal
	validationCurrentLocal = 5 * time.Second
	time.Sleep(validationCurrentLocal)
	if len(harness.vals.GetCurrentNodeIDs()) != 0 {
		t.Error("invalid")
	}
	validationCurrentLocal = vc
}
func TestTrustedByLedgerFunctions(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nc := harness.makeNode()
	nd := harness.makeNode()
	ne := harness.makeNode()
	nc.trusted = false

	na.loadFee = 12
	nb.loadFee = 1
	nc.loadFee = 12
	ne.loadFee = 12

	trustedValidations := make(map[LedgerID][]*Validation)
	a := newLedger("a")
	b := newLedger("b")
	ac := newLedger("ac")
	var id100 LedgerID
	id100[0] = 100
	trustedValidations[id100] = nil
	for _, n := range []*nodeT{na, nb, nc} {
		v := n.validate3(a)
		if harness.add(v) != VstatCurrent {
			t.Error("invalid")
		}
		if v.Trusted {
			trustedValidations[v.LedgerID] = append(trustedValidations[v.LedgerID], v)
		}
	}
	v := nd.validate3(b)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid")
	}
	trustedValidations[v.LedgerID] = append(trustedValidations[v.LedgerID], v)
	v = ne.partial(a)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid")
	}
	time.Sleep(5 * time.Second)
	for _, n := range []*nodeT{na, nb, nc} {
		v2 := n.validate3(ac)
		if harness.add(v2) != VstatCurrent {
			t.Error("invalid")
		}
		if v2.Trusted {
			trustedValidations[v2.LedgerID] = append(trustedValidations[v2.LedgerID], v2)
		}
	}
	v = nd.validate3(a)
	if harness.add(v) != VstatBadSeq {
		t.Error("invalid")
	}
	v = ne.partial(ac)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid")
	}

	for id, v2 := range trustedValidations {
		sort.Slice(v2, func(i, j int) bool {
			return bytes.Compare(v2[i].bytes(), v2[j].bytes()) < 0
		})
		if int(harness.vals.NumTrustedForLedger(id)) != len(v2) {
			t.Error("invalid")
		}
		vs1 := harness.vals.GetTrustedForLedger(id)
		sort.Slice(vs1, func(i, j int) bool {
			vi := vs1[i]
			vj := vs1[j]
			return bytes.Compare(vi.bytes(), vj.bytes()) < 0
		})
		for i := range vs1 {
			vi := vs1[i]
			if !bytes.Equal(vi.bytes(), v2[i].bytes()) {
				t.Error("invalid")
			}
		}
	}
}

func TestExpire(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	a := newLedger("a")
	if harness.add(na.validate3(a)) != VstatCurrent {
		t.Error("invalid")
	}
	if harness.vals.NumTrustedForLedger(a.ID()) == 0 {
		t.Error("invalid")
	}
	e := validationSetExpires
	validationSetExpires = 5 * time.Second
	time.Sleep(validationSetExpires)
	harness.vals.Expire()
	if harness.vals.NumTrustedForLedger(a.ID()) != 0 {
		t.Error("invalid")
	}
	validationSetExpires = e
}
func TestFlush(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nc := harness.makeNode()
	nc.trusted = false
	a := newLedger("a")
	ab := newLedger("ab")

	exp := make(map[NodeID]*Validation)

	for _, n := range []*nodeT{na, nb, nc} {
		v := n.validate3(a)
		if harness.add(v) != VstatCurrent {
			t.Error("invalid")
		}
		exp[n.nodeID] = v
	}
	staleA := exp[na.nodeID]
	// Send in a new validation for a, saving the new one into the expected
	// map after setting the proper prior ledger ID it replaced
	time.Sleep(time.Second)
	v := na.validate3(ab)
	if harness.add(v) != VstatCurrent {
		t.Error("invalid")
	}
	exp[na.nodeID] = v
	harness.vals.Flush()
	if len(harness.staledata.stale) != 1 {
		t.Error("invalid")
	}
	if harness.staledata.stale[0] != staleA {
		t.Error("invalid")
	}
	vv := harness.staledata.stale[0]
	if vv.NodeID != na.nodeID {
		t.Error("invalid")
	}
	for k, v := range harness.staledata.flushed {
		if !bytes.Equal(exp[k].bytes(), v.bytes()) {
			t.Error("invalid")
		}
	}
	if len(harness.staledata.flushed) != len(exp) {
		t.Error("invalid")
	}
}

func TestGetPreferredLedger(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nc := harness.makeNode()
	nd := harness.makeNode()
	nc.trusted = false
	a := newLedger("a")
	b := newLedger("b")
	ac := newLedger("ac")
	acd := newLedger("acd")

	s, lid := harness.vals.GetPreferred(a)
	if lid != Genesis.ID() || s != 0 {
		t.Error("invalid")
	}
	if harness.add(na.validate3(b)) != VstatCurrent {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(a)
	if lid != b.ID() || s != b.Seq {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(b)
	if lid != b.ID() || s != b.Seq {
		t.Error("invalid")
	}
	lid = harness.vals.GetPreferred2(a, 10)
	if lid != a.ID() {
		t.Error("invalid")
	}

	if harness.add(nb.validate3(a)) != VstatCurrent {
		t.Error("invalid")
	}
	if harness.add(nc.validate3(a)) != VstatCurrent {
		t.Error("invalid")
	}
	if bytes.Compare(b.id[:], a.id[:]) <= 0 {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(a)
	if lid != b.ID() || s != b.Seq {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(b)
	if lid != b.ID() || s != b.Seq {
		t.Error("invalid")
	}

	if harness.add(nd.partial(a)) != VstatCurrent {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(a)
	if lid != a.ID() || s != a.Seq {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(b)
	if lid != a.ID() || s != a.Seq {
		t.Error("invalid")
	}

	time.Sleep(5 * time.Second)

	for _, n := range []*nodeT{na, nb, nc, nd} {
		v := n.validate3(ac)
		if harness.add(v) != VstatCurrent {
			t.Error("invalid")
		}
	}
	s, lid = harness.vals.GetPreferred(a)
	if lid != a.ID() || s != a.Seq {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(b)
	if lid != ac.ID() || s != ac.Seq {
		t.Error("invalid")
	}
	s, lid = harness.vals.GetPreferred(acd)
	if lid != acd.ID() || s != acd.Seq {
		t.Error("invalid")
	}

	time.Sleep(5 * time.Second)

	for _, n := range []*nodeT{na, nb, nc, nd} {
		v := n.validate3(acd)
		if harness.add(v) != VstatCurrent {
			t.Error("invalid")
		}
	}
	for _, l := range []*Ledger{a, b, acd} {
		s, lid = harness.vals.GetPreferred(l)
		if lid != acd.ID() || s != acd.Seq {
			t.Error("invalid")
		}
	}

}
func TestGetPreferredLCL(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	a := newLedger("a")
	b := newLedger("b")
	c := newLedger("c")

	counts := make(map[LedgerID]uint32)

	if harness.vals.GetPreferredLCL(a, 0, counts) != a.id {
		t.Error()
	}
	counts[b.id]++
	if harness.vals.GetPreferredLCL(a, 0, counts) != b.id {
		t.Error()
	}
	counts[c.id]++
	if bytes.Compare(c.id[:], b.id[:]) <= 0 {
		t.Error()
	}
	if harness.vals.GetPreferredLCL(a, 0, counts) != c.id {
		t.Error()
	}
	counts[c.id] += 1000
	if harness.add(na.validate3(a)) != VstatCurrent {
		t.Error("invalid")
	}
	if harness.vals.GetPreferredLCL(a, 0, counts) != a.id {
		t.Error()
	}
	if harness.vals.GetPreferredLCL(b, 0, counts) != a.id {
		t.Error()
	}
	if harness.vals.GetPreferredLCL(c, 0, counts) != a.id {
		t.Error()
	}
	if harness.vals.GetPreferredLCL(b, 2, counts) != b.id {
		t.Error()
	}
}
func TestAcquireValidatedLedger(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	var id2, id3, id4 LedgerID
	id2[0] = 2
	id3[0] = 3
	id4[0] = 4

	v := na.validate(id2, 2, 0, 0, true)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	if harness.vals.NumTrustedForLedger(id2) != 1 {
		t.Error()
	}
	if harness.vals.GetNodesAfter(Genesis, Genesis.ID()) != 0 {
		t.Error()
	}
	s, lid := harness.vals.GetPreferred(Genesis)
	if s != 2 || lid != id2 {
		t.Fatal(s, lid)
	}
	v = nb.validate(id3, 2, 0, 0, true)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	s, lid = harness.vals.GetPreferred(Genesis)
	if s != 2 || lid != id3 {
		t.Error()
	}
	ab := newLedger("ab")
	if harness.vals.GetNodesAfter(Genesis, Genesis.ID()) != 1 {
		t.Fatal(harness.vals.GetNodesAfter(Genesis, Genesis.ID()))
	}
	time.Sleep(5 * time.Second)

	v = na.validate(id4, 4, 0, 0, true)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	if harness.vals.NumTrustedForLedger(id4) != 1 {
		t.Error()
	}
	s, lid = harness.vals.GetPreferred(Genesis)
	if s != ab.Seq || lid != ab.id {
		t.Error()
	}
	v = nb.validate(id4, 4, 0, 0, true)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	if harness.vals.NumTrustedForLedger(id4) != 2 {
		t.Error()
	}
	s, lid = harness.vals.GetPreferred(Genesis)
	if s != ab.Seq || lid != ab.id {
		t.Error()
	}
	time.Sleep(5 * time.Second)
	abcde := newLedger("abcde")
	v = na.partial(abcde)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	v = nb.partial(abcde)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	s, lid = harness.vals.GetPreferred(Genesis)
	if s != abcde.Seq || lid != abcde.id {
		t.Error()
	}
}
func TestNumTrustedForLedger(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	a := newLedger("a")

	v := na.partial(a)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	if harness.vals.NumTrustedForLedger(a.id) != 0 {
		t.Error()
	}
	v = nb.validate3(a)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	if harness.vals.NumTrustedForLedger(a.id) != 1 {
		t.Error()
	}
}
func TestSeqEnforcer(t *testing.T) {
	ledgers = make(map[LedgerID]*Ledger)
	enf := &SeqEnforcer{}
	if !enf.Try(time.Now(), 1) {
		t.Error()
	}
	if !enf.Try(time.Now(), 10) {
		t.Error()
	}
	if enf.Try(time.Now(), 5) {
		t.Error()
	}
	if enf.Try(time.Now(), 9) {
		t.Error()
	}
	e := validationSetExpires
	validationSetExpires = 5 * time.Second
	time.Sleep(validationSetExpires - time.Millisecond)
	if enf.Try(time.Now(), 1) {
		t.Error()
	}
	time.Sleep(2 * time.Millisecond)
	if !enf.Try(time.Now(), 1) {
		t.Error()
	}
	validationSetExpires = e
}

func TestTrustChanged(t *testing.T) {
	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	ledgers = make(map[LedgerID]*Ledger)

	harness := newTestHarness()
	na := harness.makeNode()
	ab := newLedger("ab")
	v := na.validate3(ab)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	listed := map[NodeID]struct{}{
		na.nodeID: {},
	}
	trusted := []*Validation{v}
	checker := func() {
		tid := Genesis.ID()
		if len(trusted) > 0 {
			tid = trusted[0].LedgerID
		}
		for i, vv := range harness.vals.CurrentTrusted() {
			if !bytes.Equal(vv.bytes(), trusted[i].bytes()) {
				t.Error()
			}
		}
		if len(harness.vals.CurrentTrusted()) != len(trusted) {
			t.Error()
		}
		for k := range harness.vals.GetCurrentNodeIDs() {
			if _, ok := listed[k]; !ok {
				t.Error()
			}
		}
		if len(harness.vals.GetCurrentNodeIDs()) != len(listed) {
			t.Error()
		}
		if harness.vals.GetNodesAfter(Genesis, Genesis.ID()) != uint32(len(trusted)) {
			t.Error()
		}
		if _, lid := harness.vals.GetPreferred(Genesis); lid != tid {
			t.Error()
		}
		for i, vv := range harness.vals.GetTrustedForLedger(tid) {
			if !bytes.Equal(vv.bytes(), trusted[i].bytes()) {
				t.Error()
			}
		}
		if len(harness.vals.GetTrustedForLedger(tid)) != len(trusted) {
			t.Error()
		}
	}
	checker()
	trusted = nil
	harness.vals.TrustChanged(nil, map[NodeID]struct{}{
		na.nodeID: {},
	})
	checker()

	harness = newTestHarness()
	na = harness.makeNode()
	na.trusted = false
	ab = newLedger("ab")
	v = na.validate3(ab)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	listed = map[NodeID]struct{}{
		na.nodeID: {},
	}
	trusted = nil
	checker()

	trusted = append(trusted, v)
	harness.vals.TrustChanged(map[NodeID]struct{}{
		na.nodeID: {},
	}, nil)
	checker()

	var id2 LedgerID
	id2[0] = 2

	harness = newTestHarness()
	na = harness.makeNode()
	ledgers = make(map[LedgerID]*Ledger)
	v = na.validate(id2, 2, 0, 0, true)
	if harness.add(v) != VstatCurrent {
		t.Error()
	}
	listed = map[NodeID]struct{}{
		na.nodeID: {},
	}
	trusted = []*Validation{v}
	vals := harness.vals
	for i, v := range vals.CurrentTrusted() {
		if trusted[i] != v {
			t.Error()
		}
	}
	if len(vals.CurrentTrusted()) != len(trusted) {
		t.Error()
	}
	if _, lid := vals.GetPreferred(Genesis); lid != v.LedgerID {
		t.Error()
	}
	if vals.GetNodesAfter(Genesis, Genesis.ID()) != 0 {
		t.Error()
	}
	trusted = nil
	vals.TrustChanged(nil, map[NodeID]struct{}{
		na.nodeID: {},
	})
	newLedger("ab")
	for i, v := range vals.CurrentTrusted() {
		if trusted[i] != v {
			t.Error()
		}
	}
	if len(vals.CurrentTrusted()) != len(trusted) {
		t.Error()
	}
	if vals.GetNodesAfter(Genesis, Genesis.ID()) != 0 {
		t.Error()
	}
}
