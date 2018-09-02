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
	"encoding/binary"
	"log"
	"sort"
	"testing"
	"time"
)

func toNetClock() time.Time {
	return time.Now().Add(86400 * time.Second)
}

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

func (n *nodeT) currKey() [40]byte {
	var r [40]byte
	copy(r[:], n.nodeID[:])
	binary.LittleEndian.PutUint32(r[32:], n.signIdx)
	return r
}

func (n *nodeT) masterKey() [40]byte {
	var r [40]byte
	copy(r[:], n.nodeID[:])
	return r
}

func (n *nodeT) validate(id LedgerID, seq Seq, signOffset, seenOffset time.Duration, full bool) *validationT {
	return &validationT{
		id:       id,
		seq:      seq,
		signTime: toNetClock().Add(signOffset),
		seenTime: toNetClock().Add(seenOffset),
		key:      n.currKey(),
		nodeID:   n.nodeID,
		full:     full,
		loadFee:  n.loadFee,
		trusted:  n.trusted,
	}
}
func (n *nodeT) validate2(l ledger, signOffset, seenOffset time.Duration) *validationT {
	return n.validate(l.ID(), l.Seq(), signOffset, seenOffset, true)
}

func (n *nodeT) validate3(l ledger) *validationT {
	return n.validate(l.ID(), l.Seq(), 0, 0, true)
}

func (n *nodeT) partial(l ledger) *validationT {
	return n.validate(l.ID(), l.Seq(), 0, 0, false)
}

type staleData struct {
	stale   []validation
	flushed map[NodeID]validation
}

type testHarness struct {
	staledata  *staleData
	vals       *validations
	nextNodeID NodeID
}

func newTestHarness() *testHarness {
	sd := staleData{
		flushed: make(map[NodeID]validation),
	}
	vs := newValidations(&adaptorT{
		staleData: &sd,
	})
	return &testHarness{
		vals:      vs,
		staledata: &sd,
	}
}

func (h *testHarness) add(v *validationT) valStatus {
	return h.vals.add(v.nodeID, v)
}

func (h *testHarness) makeNode() *nodeT {
	h.nextNodeID[0]++
	return newNodeT(h.nextNodeID)
}

func TestAddValidation1(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ab := newLedger("ab")
	// az := newLedger("az")
	abc := newLedger("abc")
	abcd := newLedger("abcd")
	abcde := newLedger("abcde")

	harness := newTestHarness()
	n := harness.makeNode()
	v := n.validate3(a)
	if harness.add(v) != current {
		t.Error("invalid add")
	}
	if s := harness.add(v); s != badSeq {
		t.Error("invalid add", s)
	}
	time.Sleep(time.Second)
	if len(harness.staledata.stale) != 0 {
		t.Error("stale is not empty")
	}
	v = n.validate3(ab)
	if harness.add(v) != current {
		t.Error("invalid add")
	}
	if len(harness.staledata.stale) != 1 {
		t.Error("incorrect stale")
	}
	if harness.staledata.stale[0].LedgerID() != a.ID() {
		t.Error("invalid stale")
	}
	if harness.vals.numTrustedForLedger(ab.ID()) != 1 {
		t.Error("invalid")
	}
	if harness.vals.numTrustedForLedger(abc.ID()) != 0 {
		t.Error("invalid")
	}
	n.signIdx++
	time.Sleep(time.Second)

	v = n.validate3(ab)
	if harness.add(v) != badSeq {
		t.Error("invalid add")
	}
	if harness.add(v) != badSeq {
		t.Error("invalid add")
	}
	time.Sleep(time.Second)

	v = n.validate3(abc)
	if harness.add(v) != current {
		t.Error("invalid add")
	}
	if harness.vals.numTrustedForLedger(ab.ID()) != 1 {
		t.Error("invalid")
	}
	if harness.vals.numTrustedForLedger(abc.ID()) != 1 {
		t.Error("invalid")
	}

	time.Sleep(2 * time.Second)

	vABCDE := n.validate3(abcde)

	time.Sleep(4 * time.Second)
	vABCD := n.validate3(abcd)

	if harness.add(vABCD) != current {
		t.Error("invalid add")
	}
	if harness.add(vABCDE) != stale {
		t.Error("invalid add")
	}
}

func TestAddValidation2(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ab := newLedger("ab")
	abc := newLedger("abc")

	harness := newTestHarness()
	n := harness.makeNode()
	v := n.validate3(a)
	if harness.add(v) != current {
		t.Error("invalid add")
	}
	v = n.validate2(ab, -time.Second, -time.Second)
	if s := harness.add(v); s != stale {
		t.Error("invalid add", s)
	}
	v = n.validate2(abc, time.Second, time.Second)
	if s := harness.add(v); s != current {
		t.Error("invalid add", s)
	}
}

func TestAddValidation3(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")

	harness := newTestHarness()
	n := harness.makeNode()
	v := n.validate2(a, -validationCurrentEarly-time.Second, 0)
	if harness.add(v) != stale {
		t.Error("invalid add")
	}

	v = n.validate2(a, validationCurrentWall+time.Second, 0)
	if s := harness.add(v); s != stale {
		t.Error("invalid add", s)
	}
	v = n.validate2(a, 0, validationCurrentLocal+time.Second)
	if s := harness.add(v); s != stale {
		t.Error("invalid add", s)
	}
}
func TestAddValidation4(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	ab := newLedger("ab")
	abc := newLedger("abc")
	az := newLedger("az")

	for _, doFull := range []bool{true, false} {
		harness := newTestHarness()
		n := harness.makeNode()
		process := func(lgr ledger) valStatus {
			if doFull {
				return harness.add(n.validate3(lgr))
			}
			return harness.add(n.partial(lgr))
		}
		if process(abc) != current {
			t.Error("invalid add")
		}
		time.Sleep(time.Second)
		if ab.Seq() >= abc.Seq() {
			t.Error("invalid seq")
		}
		if process(ab) != badSeq {
			t.Error("invalid add")
		}
		if process(az) != badSeq {
			t.Error("invalid add")
		}
		b := validationSetExpires
		validationSetExpires = 5 * time.Second
		time.Sleep(validationSetExpires + time.Millisecond)
		if process(az) != current {
			t.Error("invalid add")
		}
		validationSetExpires = b
	}
}

func TestOnStale(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ab := newLedger("ab")
	triggers := []func(*validations){
		func(vals *validations) {
			vals.currentTrusted()
		},
		func(vals *validations) {
			vals.getCurrentNodeIDs()
		},
		func(vals *validations) {
			vals.getPreferred(genesisLedger)
		},
		func(vals *validations) {
			vals.getNodesAfter(a, a.ID())
		},
	}
	for i, tr := range triggers {
		harness := newTestHarness()
		n := harness.makeNode()
		if harness.add(n.validate3(ab)) != current {
			t.Error("invalid add")
		}
		tr(harness.vals)
		if harness.vals.getNodesAfter(a, a.ID()) != 1 {
			t.Fatal("invalid getnodesafetr", i)
		}
		s, id, err := harness.vals.getPreferred(genesisLedger)
		if err != nil {
			t.Error(err)
		}
		if s != ab.Seq() || id != ab.ID() {
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
		if harness.staledata.stale[0].LedgerID() != ab.ID() {
			t.Fatal("invalid stale")
		}
		if harness.vals.getNodesAfter(a, a.ID()) != 0 {
			t.Error("invalid getnodeafter")
		}
		s, id, err = harness.vals.getPreferred(genesisLedger)
		if err != nil {
			t.Error(err)
		}
		if s != 0 || id != genesisID {
			t.Error("invalid getPreferred", s, id)
		}
		validationCurrentLocal = b
	}
}
func TestGetNodesAfter(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
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
	if harness.add(na.validate3(a)) != current {
		t.Error("invalid add")
	}
	if harness.add(nb.validate3(a)) != current {
		t.Error("invalid add")
	}
	if harness.add(nc.validate3(a)) != current {
		t.Error("invalid add")
	}
	if harness.add(nd.partial(a)) != current {
		t.Error("invalid add")
	}
	for _, l := range []ledger{a, ab, abc, ad} {
		if harness.vals.getNodesAfter(l, l.ID()) != 0 {
			t.Error("invalid")
		}
	}
	time.Sleep(5 * time.Second)
	if harness.add(na.validate3(ab)) != current {
		t.Error("invalid add")
	}
	if harness.add(nb.validate3(abc)) != current {
		t.Error("invalid add")
	}
	if harness.add(nc.validate3(ab)) != current {
		t.Error("invalid add")
	}
	if harness.add(nd.partial(abc)) != current {
		t.Error("invalid add")
	}
	if harness.vals.getNodesAfter(a, a.ID()) != 3 {
		t.Error("invalid")
	}
	if harness.vals.getNodesAfter(ab, ab.ID()) != 2 {
		t.Error("invalid")
	}
	if harness.vals.getNodesAfter(abc, abc.ID()) != 0 {
		t.Error("invalid")
	}
	if harness.vals.getNodesAfter(ad, ad.ID()) != 0 {
		t.Error("invalid")
	}

	// If given a ledger inconsistent with the id, is still able to check
	// using slower method
	if harness.vals.getNodesAfter(ad, a.ID()) != 1 {
		t.Error("invalid")
	}
	if harness.vals.getNodesAfter(ad, ab.ID()) != 2 {
		t.Error("invalid")
	}
}
func TestGetCurrentTrusted(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	b := newLedger("b")
	ac := newLedger("ac")
	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nb.trusted = false

	if harness.add(na.validate3(a)) != current {
		t.Error("invalid add")
	}
	if harness.add(nb.validate3(b)) != current {
		t.Error("invalid add")
	}
	if len(harness.vals.currentTrusted()) != 1 {
		t.Error("invalid")
	}
	tr := harness.vals.currentTrusted()
	if tr[0].LedgerID() != a.ID() {
		t.Error("invalid")
	}
	if tr[0].Seq() != a.Seq() {
		t.Error("invalid")
	}
	time.Sleep(3 * time.Second)
	for _, n := range []*nodeT{na, nb} {
		if harness.add(n.validate3(ac)) != current {
			t.Error("invalid add")
		}
	}
	if len(harness.vals.currentTrusted()) != 1 {
		t.Error("invalid")
	}
	tr = harness.vals.currentTrusted()
	if tr[0].LedgerID() != ac.ID() {
		t.Error("invalid")
	}
	if tr[0].Seq() != ac.Seq() {
		t.Error("invalid")
	}
	vc := validationCurrentLocal
	validationCurrentLocal = 5 * time.Second
	time.Sleep(validationCurrentLocal)
	if len(harness.vals.currentTrusted()) != 0 {
		t.Error("invalid")
	}
	validationCurrentLocal = vc
}
func TestGetCurrentPublicKeys(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	a := newLedger("a")
	ac := newLedger("ac")
	harness := newTestHarness()
	na := harness.makeNode()
	nb := harness.makeNode()
	nb.trusted = false

	for _, n := range []*nodeT{na, nb} {
		if harness.add(n.validate3(a)) != current {
			t.Error("invalid add")
		}
	}

	ns := harness.vals.getCurrentNodeIDs()
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
		if harness.add(n.validate3(ac)) != current {
			t.Error("invalid add")
		}
	}
	ns = harness.vals.getCurrentNodeIDs()
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
	if len(harness.vals.getCurrentNodeIDs()) != 0 {
		t.Error("invalid")
	}
	validationCurrentLocal = vc
}
func TestTrustedByLedgerFunctions(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

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

	trustedValidations := make(map[LedgerID][]*validationT)
	a := newLedger("a")
	b := newLedger("b")
	ac := newLedger("ac")
	var id100 LedgerID
	id100[0] = 100
	trustedValidations[id100] = nil
	for _, n := range []*nodeT{na, nb, nc} {
		v := n.validate3(a)
		if harness.add(v) != current {
			t.Error("invalid")
		}
		if v.trusted {
			trustedValidations[v.LedgerID()] = append(trustedValidations[v.LedgerID()], v)
		}
	}
	v := nd.validate3(b)
	if harness.add(v) != current {
		t.Error("invalid")
	}
	trustedValidations[v.LedgerID()] = append(trustedValidations[v.LedgerID()], v)
	v = ne.partial(a)
	if harness.add(v) != current {
		t.Error("invalid")
	}
	time.Sleep(5 * time.Second)
	for _, n := range []*nodeT{na, nb, nc} {
		v := n.validate3(ac)
		if harness.add(v) != current {
			t.Error("invalid")
		}
		if v.trusted {
			trustedValidations[v.LedgerID()] = append(trustedValidations[v.LedgerID()], v)
		}
	}
	v = nd.validate3(a)
	if harness.add(v) != badSeq {
		t.Error("invalid")
	}
	v = ne.partial(ac)
	if harness.add(v) != current {
		t.Error("invalid")
	}

	for id, v := range trustedValidations {
		sort.Slice(v, func(i, j int) bool {
			return bytes.Compare(v[i].bytes(), v[j].bytes()) < 0
		})
		if int(harness.vals.numTrustedForLedger(id)) != len(v) {
			t.Error("invalid")
		}
		vs1 := harness.vals.getTrustedForLedger(id)
		sort.Slice(vs1, func(i, j int) bool {
			vi := vs1[i].(*validationT)
			vj := vs1[j].(*validationT)
			return bytes.Compare(vi.bytes(), vj.bytes()) < 0
		})
		for i := range vs1 {
			vi := vs1[i].(*validationT)
			if !bytes.Equal(vi.bytes(), v[i].bytes()) {
				t.Error("invalid")
			}
		}
	}
}

func TestExpire(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

	harness := newTestHarness()
	na := harness.makeNode()
	a := newLedger("a")
	if harness.add(na.validate3(a)) != current {
		t.Error("invalid")
	}
	if harness.vals.numTrustedForLedger(a.ID()) == 0 {
		t.Error("invalid")
	}
	e := validationSetExpires
	validationSetExpires = 5 * time.Second
	time.Sleep(validationSetExpires)
	harness.vals.expire()
	if harness.vals.numTrustedForLedger(a.ID()) != 0 {
		t.Error("invalid")
	}
	validationSetExpires = e
}
