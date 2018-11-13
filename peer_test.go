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

package consensus

import (
	"context"
	"errors"
	"log"
	"sync"
	"testing"
	"time"
)

var ledgers2 = make(map[LedgerID]*Ledger)
var txSets = make(map[TxSetID]TxSet)
var mutex sync.RWMutex

type tx struct {
	id TxID
}

func (t *tx) ID() TxID {
	return t.id
}

//PeerInterface is the funcs for intererctint Peer struct.
type adaptor struct {
	others  []*adaptor
	peer    *Peer
	t       *testing.T
	openTxs TxSet
	ledger  *Ledger
}

// Attempt to acquire a specific ledger.
func (a *adaptor) AcquireLedger(id LedgerID) (*Ledger, error) {
	mutex.RLock()
	l, ok := ledgers2[id]
	mutex.RUnlock()
	if ok {
		return l, nil
	}
	return nil, errors.New("not found")
}

// Handle a newly stale validation, this should do minimal work since
// it is called by Validations while it may be iterating Validations
// under lock
func (a *adaptor) OnStale(*Validation) {} //nothing

// Flush the remaining validations (typically done on shutdown)
func (a *adaptor) Flush(remaining map[NodeID]*Validation) {} //nothing

// Acquire the transaction set associated with a proposed position.
func (a *adaptor) AcquireTxSet(id TxSetID) ([]TxT, error) {
	mutex.RLock()
	l, ok := txSets[id]
	mutex.RUnlock()
	if ok {
		txs := make([]TxT, 0, len(l))
		for _, ll := range l {
			txs = append(txs, ll)
		}
		return txs, nil
	}
	return nil, errors.New("not found")
}

// Whether any transactions are in the open ledger
func (a *adaptor) HasOpenTransactions() bool {
	return len(a.openTxs) > 0
}

// Called whenever consensus operating mode changes
func (a *adaptor) OnModeChange(Mode, Mode) {} //nothing

// Called when ledger closes
func (a *adaptor) OnClose(*Ledger, time.Time, Mode) TxSet {
	return a.openTxs
}
func (a *adaptor) ShouldAccept(*Result) bool {
	return true
}

// Called when ledger is accepted by consensus
func (a *adaptor) OnAccept(l *Ledger) {
	log.Println("onaccept")
	mutex.Lock()
	ledgers2[l.ID()] = l.Clone()
	mutex.Unlock()
	a.openTxs = make(TxSet)
	log.Println(a.peer.id, l)
	a.ledger = l
	log.Println("end of onaccept", a.peer.id[:2], a.ledger.ID())
}

// Propose the position to Peers.
func (a *adaptor) Propose(prop *Proposal) {
	for _, o := range a.others {
		go func(o *adaptor) {
			o.peer.AddProposal(prop.Clone())
		}(o)
	}
}

// Share a received Peer proposal with other Peer's.
func (a *adaptor) SharePosition(prop *Proposal) {
	for _, o := range a.others {
		go func(o *adaptor) {
			o.peer.AddProposal(prop.Clone())
		}(o)
	}
}

func (a *adaptor) receiveTx(t TxT) {
	// a.openTxs[t.ID()] = t
}

// Share a disputed transaction with Peers
func (a *adaptor) ShareTx(t TxT) {
	for _, o := range a.others {
		o.receiveTx(t)
	}
}

// Share given transaction set with Peers
func (a *adaptor) ShareTxset(ts TxSet) {
	mutex.Lock()
	txSets[ts.ID()] = ts.Clone()
	mutex.Unlock()
}

// Share my validation
func (a *adaptor) ShareValidaton(v *Validation) {
	for _, o := range a.others {
		go func(o *adaptor) {
			vv := *v
			o.peer.AddValidation(&vv)
		}(o)
	}
}

func TestPeer(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	ledgers2 = make(map[LedgerID]*Ledger)
	txSets = make(map[TxSetID]TxSet)

	var nid [4]NodeID
	for i := range nid {
		nid[i][0] = byte(i)
	}
	var a [4]*adaptor
	for i := range a {
		a[i] = &adaptor{
			t:       t,
			openTxs: make(TxSet),
		}
	}
	unl := []NodeID{nid[0], nid[1], nid[2], nid[3]}
	var p [4]*Peer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ctxs [4]context.Context
	var cancels [4]context.CancelFunc
	for i := range p {
		p[i] = NewPeer(a[i], nid[i], unl, true, Genesis)
		a[i].peer = p[i]
	}
	a[0].others = []*adaptor{a[1], a[2], a[3]}
	a[1].others = []*adaptor{a[0], a[2], a[3]}
	a[2].others = []*adaptor{a[0], a[1], a[3]}
	a[3].others = []*adaptor{a[0], a[1], a[2]}
	for i, pp := range p {
		ctxs[i], cancels[i] = context.WithCancel(ctx)
		pp.Start(ctxs[i])
	}
	time.Sleep(5 * time.Second)
	for i, pp := range a {
		if pp.ledger == nil {
			t.Error("invalid ledger", i)
		}
		if pp.ledger.Seq != 1 {
			t.Error("invalid ledger")
		}
	}
	log.Println("*************************same txs")
	var tid [15]TxID
	for i := range tid {
		tid[i][0] = byte(i)
	}
	var txs [len(tid)]*tx
	for i := range txs {
		txs[i] = &tx{
			id: tid[i],
		}

	}
	for _, v := range a {
		v.openTxs[tid[0]] = txs[0]
		txSets[v.openTxs.ID()] = v.openTxs.Clone()
	}
	time.Sleep(5 * time.Second)
	for _, v := range a {
		if v.ledger.Seq != 2 {
			t.Error("invalid ledger")
		}
		if _, ok := v.ledger.Txs[tid[0]]; !ok {
			t.Error("invalid ledger")
		}
	}
	for _, v := range a {
		if len(v.openTxs) != 0 {
			t.Error("invalid opentxs")
		}
	}
	log.Println("*********************************different txs, but agree")
	a[0].openTxs[tid[1]] = txs[1]
	txSets[a[0].openTxs.ID()] = a[0].openTxs.Clone()
	log.Println("txid", tid[1][:2], "txset id", a[0].openTxs.ID())
	for i := 1; i < len(a); i++ {
		a[i].openTxs[tid[2]] = txs[2]
		txSets[a[i].openTxs.ID()] = a[i].openTxs.Clone()
		log.Println("txid", tid[2][:2], "txset id", a[i].openTxs.ID())
	}
	time.Sleep(5 * time.Second)
	for _, v := range a {
		if v.ledger.Seq != 3 {
			t.Error("invalid ledger")
		}
		if _, ok := v.ledger.Txs[tid[2]]; !ok {
			t.Error("invalid ledger")
		}
		if _, ok := v.ledger.Txs[tid[1]]; ok {
			t.Error("invalid ledger")
		}
		if _, ok := v.ledger.Txs[tid[0]]; ok {
			t.Error("invalid ledger")
		}
	}
	log.Println("*****************************don't agree")
	for i := range a {
		a[i].openTxs[tid[i+3]] = txs[i+3]
		txSets[a[i].openTxs.ID()] = a[i].openTxs.Clone()
		log.Println("txid", tid[i+3][:2], "txset id", a[i].openTxs.ID())
	}
	time.Sleep(5 * time.Second)
	for _, v := range a {
		if v.ledger.Seq != 4 {
			t.Error("invalid ledger")
		}
		if len(v.ledger.Txs) != 0 {
			t.Error("invalid ledger")
		}
	}
	resendValidationWaitTime = 3 * time.Second
	log.Println("*****************************die and alive")
	log.Println("p0 die")
	cancels[0]()
	for i := 1; i < len(a); i++ {
		a[i].openTxs[tid[8]] = txs[8]
		txSets[a[i].openTxs.ID()] = a[i].openTxs.Clone()
		log.Println("txid", tid[8][:2], "txset id", a[i].openTxs.ID())
	}
	time.Sleep(5 * time.Second)
	ctxs[0], cancels[0] = context.WithCancel(ctx)
	p[0].Start(ctxs[0])
	log.Println("p0 alive")
	time.Sleep(10 * time.Second)
	for _, v := range a {
		if v.ledger.Seq != 5 {
			t.Error("invalid ledger")
		}
		if len(v.ledger.Txs) != 1 {
			t.Error("invalid opentxs")
		}
		if v.ledger.ID() != a[3].ledger.ID() {
			t.Error("invalid ledger")
		}
		if _, ok := v.ledger.Txs[tid[8]]; !ok {
			t.Error("invalid ledger")
		}
	}

}
func TestPeer2(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)
	ledgers2 = make(map[LedgerID]*Ledger)
	txSets = make(map[TxSetID]TxSet)

	var nid [4]NodeID
	for i := range nid {
		nid[i][0] = byte(i)
	}
	var a [4]*adaptor
	for i := range a {
		a[i] = &adaptor{
			t:       t,
			openTxs: make(TxSet),
		}
	}
	unl := []NodeID{nid[0], nid[1], nid[2], nid[3]}
	var p [4]*Peer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ctxs [4]context.Context
	var cancels [4]context.CancelFunc
	p[0] = NewPeer(a[0], nid[0], unl, false, Genesis)
	a[0].peer = p[0]

	for i := 1; i < len(p); i++ {
		p[i] = NewPeer(a[i], nid[i], unl, true, Genesis)
		a[i].peer = p[i]
	}
	a[0].others = []*adaptor{a[1], a[2], a[3]}
	a[1].others = []*adaptor{a[0], a[2], a[3]}
	a[2].others = []*adaptor{a[0], a[1], a[3]}
	a[3].others = []*adaptor{a[0], a[1], a[2]}
	for i, pp := range p {
		ctxs[i], cancels[i] = context.WithCancel(ctx)
		pp.Start(ctxs[i])
	}
	time.Sleep(5 * time.Second)
	for _, pp := range a {
		if pp.ledger.Seq != 1 {
			t.Error("invalid ledger")
		}
	}
	log.Println("*************************")
	var tid [4]TxID
	for i := range tid {
		tid[i][0] = byte(i)
	}
	var txs [len(tid)]*tx
	for i := range txs {
		txs[i] = &tx{
			id: tid[i],
		}
	}
	a[0].openTxs[tid[0]] = txs[0]
	txSets[a[0].openTxs.ID()] = a[0].openTxs.Clone()
	log.Println("txid", tid[1][:2], "txset id", a[0].openTxs.ID())
	for i := 1; i < len(a); i++ {
		a[i].openTxs[tid[1]] = txs[1]
		txSets[a[i].openTxs.ID()] = a[i].openTxs.Clone()
		log.Println("txid", tid[1][:2], "txset id", a[i].openTxs.ID())
	}
	time.Sleep(5 * time.Second)
	for _, v := range a {
		if v.ledger.Seq != 2 {
			t.Error("invalid ledger")
		}
		if _, ok := v.ledger.Txs[tid[1]]; !ok {
			t.Error("invalid ledger")
		}
		if _, ok := v.ledger.Txs[tid[0]]; ok {
			t.Error("invalid ledger")
		}
	}
}
