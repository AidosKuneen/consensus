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

// This is a rewrite of https://github.com/ripple/rippled/src/ripple/consensus
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
	"sync"
	"time"
)

/** Timing parameters to control validation staleness and expiration.

  @note These are protocol level parameters that should not be changed without
        careful consideration.  They are *not* implemented as static constexpr
        to allow simulation code to test alternate parameter settings.
*/

var (
	/** The number of seconds a validation remains current after its ledger's
	  close time.

	  This is a safety to protect against very old validations and the time
	  it takes to adjust the close time accuracy window.
	*/
	validationCurrentWall = 5 * time.Minute

	/** Duration a validation remains current after first observed.

	  The number of seconds a validation remains current after the time we
	  first saw it. This provides faster recovery in very rare cases where the
	  number of validations produced by the network is lower than normal
	*/
	validationCurrentLocal = 3 * time.Minute

	/** Duration pre-close in which validations are acceptable.

	  The number of seconds before a close time that we consider a validation
	  acceptable. This protects against extreme clock errors
	*/
	validationCurrentEarly = 3 * time.Minute

	/** Duration a set of validations for a given ledger hash remain valid

	  The number of seconds before a set of validations for a given ledger
	  hash can expire.  This keeps validations for recent ledgers available
	  for a reasonable interval.
	*/
	validationSetExpires = 10 * time.Minute
)

// SeqEnforcer enforce validation increasing sequence requirement.
//   Helper class for enforcing that a validation must be larger than all
//   unexpired validation sequence numbers previously issued by the validator
//   tracked by the instance of this class.
type SeqEnforcer struct {
	largest Seq
	when    time.Time
}

// Try advancing the largest observed validation ledger sequence
//   Try setting the largest validation sequence observed, but return false
//   if it violates the invariant that a validation must be larger than all
//   unexpired validation sequence numbers.
//   @param now The current time
//   @param s The sequence number we want to validate
//   @param p Validation parameters
//   @return Whether the validation satisfies the invariant
func (sf *SeqEnforcer) Try(now time.Time, s Seq) bool {
	if now.After(sf.when.Add(validationSetExpires)) {
		sf.largest = 0
	}
	if s <= sf.largest {
		return false
	}
	sf.largest = s
	sf.when = now
	return true
}

/** Whether a validation is still current

  Determines whether a validation can still be considered the current
  validation from a node based on when it was signed by that node and first
  seen by this node.

  @param p ValidationParms with timing parameters
  @param now Current time
  @param signTime When the validation was signed
  @param seenTime When the validation was first seen locally
*/

func isCurrent(now, signTime, seenTime time.Time) bool {
	// Because this can be called on untrusted, possibly
	// malicious validations, we do our math in a way
	// that avoids any chance of overflowing or underflowing
	// the signing time.
	return signTime.After(now.Add(-validationCurrentEarly)) &&
		signTime.Before(now.Add(validationCurrentWall)) &&
		(seenTime.IsZero() ||
			seenTime.Before(now.Add(validationCurrentLocal)) &&
				seenTime.After(now.Add(-validationCurrentLocal))) //added from original
}

//ValStatus is status of newly received validation
type ValStatus byte

const (
	//VstatCurrent means this was a new validation and was added
	VstatCurrent ValStatus = iota
	//VstatStale means not current or was older than current from this node
	VstatStale
	//VstatBadSeq means a validation violates the increasing seq requirement
	VstatBadSeq
)

func (m ValStatus) String() string {
	switch m {
	case VstatCurrent:
		return "current"
	case VstatStale:
		return "stale"
	case VstatBadSeq:
		return "badSeq"
	default:
		return "unknown"
	}
}

// Validations maintains current and recent ledger Validations.
//   Manages storage and queries related to Validations received on the network.
//   Stores the most current validation from nodes and sets of recent
//   Validations grouped by ledger identifier.
//   Stored Validations are not necessarily from trusted nodes, so clients
//   and implementations should take care to use `trusted` member functions or
//   check the validation's trusted status.
//   This class uses a generic interface to allow adapting Validations for
//   specific applications. The Adaptor template implements a set of helper
//   functions and type definitions. The code stubs below outline the
//   interface and type requirements.
//   @warning The Adaptor::MutexType is used to manage concurrent access to
//            private members of Validations but does not manage any data in the
// 		   Adaptor instance itself.
type Validations struct {
	// Manages concurrent access to members
	mutex sync.Mutex

	// Validations from currently listed and trusted nodes (partial and full)
	current map[NodeID]Validation

	// Used to enforce the largest validation invariant for the local node
	localSeqEnforcer SeqEnforcer

	// Sequence of the largest validation received from each node
	seqEnforcers map[NodeID]SeqEnforcer

	//! Validations from listed nodes, indexed by ledger id (partial and full)
	byLedger agedMap

	// Represents the ancestry of validated ledgers
	trie *ledgerTrie

	// Last (validated) ledger successfully acquired. If in this map, it is
	// accounted for in the trie.
	lastLedger map[NodeID]Ledger

	// Set of ledgers being acquired from the network
	acquiring map[seqLedgerID]map[NodeID]struct{}

	// Adaptor instance
	// Is NOT managed by the mutex_ above
	Adaptor Adaptor
}

func (v *Validations) removeTrie(nodeID NodeID, val Validation) {
	slid := toSeqLedgerID(val.Seq(), val.LedgerID())
	if a, ok := v.acquiring[slid]; ok {
		delete(a, nodeID)
		if len(a) == 0 {
			delete(v.acquiring, slid)
		}
	}
	it, ok := v.lastLedger[nodeID]
	if ok && it.ID() == val.LedgerID() {
		v.trie.remove(it, 1)
		delete(v.lastLedger, nodeID)
	}
}

// Check if any pending acquire ledger requests are complete
func (v *Validations) checkAcquired() {
	for sli, m := range v.acquiring {
		_, lid := fromSeqLedgerID(sli)
		l, err := v.Adaptor.AcquireLedger(lid)
		if err != nil {
			continue
		}
		for nid := range m {
			v.updateTrie(nid, l)
		}
		delete(v.acquiring, sli)
	}
}

// Update the trie to reflect a new validated ledger
func (v *Validations) updateTrie(nodeID NodeID, l Ledger) {
	oldl, ok := v.lastLedger[nodeID]
	v.lastLedger[nodeID] = l
	if ok {
		v.trie.remove(oldl, 1)
	}
	v.trie.insert(l, 1)
}

/** Process a new validation

  Process a new trusted validation from a validator. This will be
  reflected only after the validated ledger is successfully acquired by
  the local node. In the interim, the prior validated ledger from this
  node remains.

  @param lock Existing lock of mutex_
  @param nodeID The node identifier of the validating node
  @param val The trusted validation issued by the node
  @param prior If not none, the last current validated ledger Seq,ID of key
*/
func (v *Validations) updateTrie2(
	nodeID NodeID, val Validation, seq Seq, id LedgerID) {
	// Clear any prior acquiring ledger for this node
	slid := toSeqLedgerID(seq, id)
	if it, ok := v.acquiring[slid]; ok {
		delete(it, nodeID)
		if len(it) == 0 {
			delete(v.acquiring, slid)
		}
	}
	v.updateTrie3(nodeID, val)
}

func (v *Validations) updateTrie3(nodeID NodeID, val Validation) {
	if !val.Trusted() {
		panic("validator is not trusted")
	}
	v.checkAcquired()

	slid := toSeqLedgerID(val.Seq(), val.LedgerID())
	if it, ok := v.acquiring[slid]; ok {
		it[nodeID] = struct{}{}
		return
	}
	ll, err := v.Adaptor.AcquireLedger(val.LedgerID())
	if err == nil {
		v.updateTrie(nodeID, ll)
	} else {
		v.acquiring[slid] = map[NodeID]struct{}{
			nodeID: struct{}{},
		}
	}
}

/** Use the trie for a calculation

Accessing the trie through this helper ensures acquiring validations
are checked and any stale validations are flushed from the trie.

@param lock Existing lock of mutex_
@param f Invokable with signature (LedgerTrie<Ledger> &)

@warning The invokable `f` is expected to be a simple transformation of
		 its arguments and will be called with mutex_ under lock.

*/
func (v *Validations) withTrie(f func(*ledgerTrie)) {
	// Call current to flush any stale validations
	v.doCurrent(func(i int) {}, func(n NodeID, v Validation) {})
	v.checkAcquired()
	f(v.trie)
}

/** Iterate current validations.

  Iterate current validations, flushing any which are stale.

  @param lock Existing lock of mutex_
  @param pre Invokable with signature (std::size_t) called prior to
             looping.
  @param f Invokable with signature (NodeID const &, Validations const &)
           for each current validation.

  @note The invokable `pre` is called _prior_ to checking for staleness
        and reflects an upper-bound on the number of calls to `f.
  @warning The invokable `f` is expected to be a simple transformation of
           its arguments and will be called with mutex_ under lock.
*/

func (v *Validations) doCurrent(pre func(int), f func(NodeID, Validation)) {
	t := v.Adaptor.Now()
	pre(len(v.current))
	for k, val := range v.current {
		// Check for staleness
		if !isCurrent(t, val.SignTime(), val.SeenTime()) {
			v.removeTrie(k, val)
			v.Adaptor.OnStale(val)
			delete(v.current, k)
			log.Println("and staled")
		} else {
			// contains a live record
			f(k, val)
		}
	}
}

/** Iterate the set of validations associated with a given ledger id

   @param lock Existing lock on mutex_
   @param ledgerID The identifier of the ledger
   @param pre Invokable with signature(std::size_t)
   @param f Invokable with signature (NodeID const &, Validation const &)

   @note The invokable `pre` is called prior to iterating validations. The
         argument is the number of times `f` will be called.
   @warning The invokable f is expected to be a simple transformation of
  its arguments and will be called with mutex_ under lock.
*/
func (v *Validations) loopByLedger(id LedgerID, pre func(int), f func(NodeID, Validation)) {
	v.byLedger.loop(id, pre, f)
}

// NewValidations is the constructor of Validatoins
//   @param p ValidationParms to control staleness/expiration of validations
//   @param c Clock to use for expiring validations stored by ledger
//   @param ts Parameters for constructing Adaptor instance
func NewValidations(a Adaptor) *Validations {
	return &Validations{
		byLedger:     make(agedMap),
		Adaptor:      a,
		current:      make(map[NodeID]Validation),
		seqEnforcers: make(map[NodeID]SeqEnforcer),
		lastLedger:   make(map[NodeID]Ledger),
		acquiring:    make(map[seqLedgerID]map[NodeID]struct{}),
		trie:         newLedgerTrie(),
	}
}

// CanValidateSeq eturns whether the local node can issue a validation for the given sequence
//   number
//   @param s The sequence number of the ledger the node wants to validate
//   @return Whether the validation satisfies the invariant, updating the
//           largest sequence number seen accordingly
func (v *Validations) CanValidateSeq(s Seq) bool {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	return v.localSeqEnforcer.Try(time.Now(), s)
}

//Add a new validation
//   Attempt to Add a new validation.
//   @param nodeID The identity of the node issuing this validation
//   @param val The validation to store
//   @return The outcome
func (v *Validations) Add(nodeID NodeID, val Validation) ValStatus {
	if !isCurrent(v.Adaptor.Now(), val.SignTime(), val.SeenTime()) {
		return VstatStale
	}
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Check that validation sequence is greater than any non-expired
	// validations sequence from that validator
	now := time.Now()
	enforcer := v.seqEnforcers[nodeID]
	if !enforcer.Try(now, val.Seq()) {
		return VstatBadSeq
	}
	v.seqEnforcers[nodeID] = enforcer
	// Use insert_or_assign when C++17 supported
	v.byLedger.add(val.LedgerID(), nodeID, val)
	oldVal, ok := v.current[nodeID]
	if !ok {
		v.current[nodeID] = val
		if val.Trusted() {
			v.updateTrie3(nodeID, val)
		}
		return VstatCurrent
	}
	if !val.SignTime().After(oldVal.SignTime()) {
		return VstatStale
	}
	v.Adaptor.OnStale(oldVal)
	v.current[nodeID] = val
	if val.Trusted() {
		v.updateTrie2(nodeID, val, oldVal.Seq(), oldVal.LedgerID())
	}
	return VstatCurrent
}

// Expire old validation sets
//   Remove validation sets that were accessed more than
//   validationSET_EXPIRES ago.
func (v *Validations) Expire() {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.expire(validationSetExpires)
}

// TrustChanged updates trust status of validations
//   Updates the trusted status of known validations to account for nodes
//   that have been added or removed from the UNL. This also updates the trie
//   to ensure only currently trusted nodes' validations are used.
//   @param added Identifiers of nodes that are now trusted
//   @param removed Identifiers of nodes that are no longer trusted
func (v *Validations) TrustChanged(added, removed map[NodeID]struct{}) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for id, val := range v.current {
		if _, ok := added[id]; ok {
			val.SetTrusted()
			v.updateTrie3(id, val)
			continue
		}
		if _, ok := removed[id]; ok {
			val.SetUntrusted()
			v.removeTrie(id, val)
		}
	}
	for _, val := range v.byLedger {
		for nodeVal, val2 := range val.m {
			if _, ok := added[nodeVal]; ok {
				val2.SetTrusted()
				continue
			}
			if _, ok := removed[nodeVal]; ok {
				val2.SetUntrusted()
			}
		}
	}
}

// GetPreferred returns the sequence number and ID of the preferred working ledger
//   A ledger is preferred if it has more support amongst trusted validators
//   and is *not* an ancestor of the current working ledger; otherwise it
//   remains the current working ledger.
//   (Parent of preferred stays put)
//   @param curr The local node's current working ledger
//   @return The sequence and id of the preferred working ledger,
//           or Seq{0},ID{0} if no trusted validations are available to
//           determine the preferred ledger.
func (v *Validations) GetPreferred(curr Ledger) (Seq, LedgerID) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	var preferred *spanTip
	v.withTrie(func(trie *ledgerTrie) {
		preferred = trie.getPreferred(v.localSeqEnforcer.largest)
	})

	// No trusted validations to determine branch
	if preferred.seq == 0 {
		// fall back to majority over acquiring ledgers
		var maxseq Seq
		var maxid LedgerID
		var size int
		for slid, m := range v.acquiring {
			s, lid := fromSeqLedgerID(slid)
			// order by number of trusted peers validating that ledger
			// break ties with ledger ID
			if size < len(m) || (size == len(m) && bytes.Compare(maxid[:], lid[:]) < 0) {
				maxseq = s
				maxid = lid
				size = len(m)
			}
		}
		if len(v.acquiring) != 0 {
			return maxseq, maxid
		}
		return preferred.seq, preferred.id
	}

	// If we are the parent of the preferred ledger, stick with our
	// current ledger since we might be about to generate it
	if preferred.seq == curr.Seq()+1 &&
		preferred.ancestor(curr.Seq()) == curr.ID() {
		return curr.Seq(), curr.ID()
	}

	// A ledger ahead of us is preferred regardless of whether it is
	// a descendant of our working ledger or it is on a different chain
	if preferred.seq > curr.Seq() {
		return preferred.seq, preferred.id
	}
	// Only switch to earlier or same sequence number
	// if it is a different chain.
	if curr.IndexOf(preferred.seq) != preferred.id {
		return preferred.seq, preferred.id
	}
	// Stick with current ledger
	return curr.Seq(), curr.ID()
}

// GetPreferred2 Gets the ID of the preferred working ledger that exceeds a minimum valid
//   ledger sequence number
//   @param curr Current working ledger
//   @param minValidSeq Minimum allowed sequence number
//   @return ID Of the preferred ledger, or curr if the preferred ledger
//              is not valid
func (v *Validations) GetPreferred2(curr Ledger, minValidSeq Seq) LedgerID {
	preferredSeq, preferredID := v.GetPreferred(curr)
	if preferredSeq >= minValidSeq && preferredID != genesisID {
		return preferredID
	}
	return curr.ID()
}

// GetPreferredLCL Determines the preferred last closed ledger for the next consensus round.
//   Called before starting the next round of ledger consensus to determine the
//   preferred working ledger. Uses the dominant peerCount ledger if no
//   trusted validations are available.
//   @param lcl Last closed ledger by this node
//   @param minSeq Minimum allowed sequence number of the trusted preferred ledger
//   @param peerCounts Map from ledger ids to count of peers with that as the
//                     last closed ledger
//   @return The preferred last closed ledger ID
//   @note The minSeq does not apply to the peerCounts, since this function
//         does not know their sequence number
func (v *Validations) GetPreferredLCL(lcl Ledger, minSeq Seq, peerCounts map[LedgerID]uint32) LedgerID {
	preferredSeq, preferredID := v.GetPreferred(lcl)
	// Trusted validations exist
	if preferredID != genesisID && preferredSeq > 0 {
		if preferredSeq >= minSeq {
			return preferredID
		}
		return lcl.ID()
	}
	// Otherwise, rely on peer ledgers

	var maxCnt uint32
	var maxid LedgerID
	for id, n := range peerCounts {
		if maxCnt < n || (maxCnt == n && bytes.Compare(maxid[:], id[:]) < 0) {
			// Prefer larger counts, then larger ids on ties
			// (max_element expects this to return true if a < b)
			maxCnt = n
			maxid = id
		}
	}
	if len(peerCounts) > 0 {
		return maxid
	}
	return lcl.ID()
}

//GetNodesAfter counts the number of current trusted validators working on a ledger
//   after the specified one.
//   @param ledger The working ledger
//   @param ledgerID The preferred ledger
//   @return The number of current trusted validators working on a descendant
//           of the preferred ledger
//   @note If ledger.id() != ledgerID, only counts immediate child ledgers of
//         ledgerID
func (v *Validations) GetNodesAfter(l Ledger, id LedgerID) uint32 {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Use trie if ledger is the right one
	var num uint32
	if l.ID() == id {
		v.withTrie(func(trie *ledgerTrie) {
			num = trie.branchSupport(l) - trie.tipSupport(l)
		})
		return num
	}
	// Count parent ledgers as fallback
	for _, curr := range v.lastLedger {
		if curr.Seq() > 0 &&
			curr.IndexOf(curr.Seq()-1) == id {
			num++
		}
	}
	return num
}

// CurrentTrusted gets the currently trusted full validations
//   @return Vector of validations from currently trusted validators
func (v *Validations) CurrentTrusted() []Validation {
	var ret []Validation
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.doCurrent(
		func(numValidations int) {
			ret = make([]Validation, 0, numValidations)
		},
		func(n NodeID, v Validation) {
			if v.Trusted() && v.Full() {
				ret = append(ret, v)
			}
		})
	return ret
}

//GetCurrentNodeIDs gets the set of node ids associated with current validations
//@return The set of node ids for active, listed validators
func (v *Validations) GetCurrentNodeIDs() map[NodeID]struct{} {
	ret := make(map[NodeID]struct{})
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.doCurrent(
		func(numValidations int) {
		},
		func(nid NodeID, v Validation) {
			ret[nid] = struct{}{}
		})

	return ret
}

//NumTrustedForLedger  counts the number of trusted full validations for the given ledger
//     @param ledgerID The identifier of ledger of interest
//     @return The number of trusted validations
func (v *Validations) NumTrustedForLedger(id LedgerID) uint {
	var count uint
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.loop(
		id,
		func(i int) {}, // nothing to reserve
		func(nid NodeID, v Validation) {
			if v.Trusted() && v.Full() {
				count++
			}
		})
	return count
}

//GetTrustedForLedger  gets trusted full validations for a specific ledger
//      @param ledgerID The identifier of ledger of interest
//      @return Trusted validations associated with ledger
func (v *Validations) GetTrustedForLedger(id LedgerID) []Validation {
	var res []Validation
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.loop(
		id,
		func(numValidations int) {
			res = make([]Validation, 0, numValidations)
		},
		func(nid NodeID, v Validation) {
			if v.Trusted() && v.Full() {
				res = append(res, v)
			}
		})

	return res
}

//Fees returns fees reported by trusted full validators in the given ledger
//     @param ledgerID The identifier of ledger of interest
//     @param baseFee The fee to report if not present in the validation
//     @return Vector of Fees
func (v *Validations) Fees(id LedgerID, baseFee uint32) []uint32 {
	var res []uint32
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.loop(
		id,
		func(numValidations int) {
			res = make([]uint32, 0, numValidations)
		},
		func(nid NodeID, v Validation) {
			if v.Trusted() && v.Full() {
				loadFee := v.LoadFee()
				if loadFee > 0 {
					res = append(res, loadFee)
				} else {
					res = append(res, baseFee)
				}
			}
		})
	return res
}

//Flush all current validations
func (v *Validations) Flush() {
	flushed := make(map[NodeID]Validation)
	v.mutex.Lock()
	defer v.mutex.Unlock()
	for nid, v := range v.current {
		flushed[nid] = v
	}
	v.current = make(map[NodeID]Validation)
	v.Adaptor.Flush(flushed)
}
