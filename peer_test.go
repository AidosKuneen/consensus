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
	"errors"
	"log"
	"testing"
	"time"
)

var ledgers2 = make(map[LedgerID]*Ledger)
var txSets = make(map[TxSetID]TxSet)

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
	l, ok := ledgers2[id]
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
func (a *adaptor) AcquireTxSet(id TxSetID) (TxSet, error) {
	l, ok := txSets[id]
	if ok {
		return l, nil
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

// Called when ledger is accepted by consensus
func (a *adaptor) OnAccept(l *Ledger) {
	ledgers2[l.ID()] = l.Clone()
	a.openTxs = make(TxSet)
	a.ledger = l
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
		go func(o *adaptor) {
			o.receiveTx(t)
		}(o)
	}
}

// Share given transaction set with Peers
func (a *adaptor) ShareTxset(ts TxSet) {
	txSets[ts.ID()] = ts.Clone()
}

// Share my validation
func (a *adaptor) ShareValidaton(v *Validation) {
	for _, o := range a.others {
		go func(o *adaptor) {
			vv := *v
			o.peer.AdddValidation(&vv)
		}(o)
	}
}

func TestPeer(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Llongfile)

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
	for i := range p {
		p[i] = NewPeer(a[i], nid[i], unl, true)
		a[i].peer = p[i]
	}
	a[0].others = []*adaptor{a[1], a[2], a[3]}
	a[1].others = []*adaptor{a[0], a[2], a[3]}
	a[2].others = []*adaptor{a[0], a[1], a[3]}
	a[3].others = []*adaptor{a[0], a[1], a[2]}
	for _, pp := range p {
		pp.Start()
	}
	time.Sleep(5 * time.Second)
	for _, pp := range a {
		if pp.ledger.Seq != 1 {
			t.Error("invalid ledger")
		}
	}
	log.Println("*************************same txs")
	var tid [7]TxID
	for i := range tid {
		tid[i][0] = byte(i)
	}
	var txs [7]*tx
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
	for _, v := range p {
		v.Stop()
	}
}
