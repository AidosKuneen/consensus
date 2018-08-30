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
	"errors"
	"math"
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
	validationCURRENTWALL = 5 * time.Minute

	/** Duration a validation remains current after first observed.

	  The number of seconds a validation remains current after the time we
	  first saw it. This provides faster recovery in very rare cases where the
	  number of validations produced by the network is lower than normal
	*/
	validationCURRENTLOCAL = 3 * time.Minute

	/** Duration pre-close in which validations are acceptable.

	  The number of seconds before a close time that we consider a validation
	  acceptable. This protects against extreme clock errors
	*/
	validationCURRENTEARLY = 3 * time.Minute

	/** Duration a set of validations for a given ledger hash remain valid

	  The number of seconds before a set of validations for a given ledger
	  hash can expire.  This keeps validations for recent ledgers available
	  for a reasonable interval.
	*/
	validationSETEXPIRES = 10 * time.Minute
)

/** Enforce validation increasing sequence requirement.

  Helper class for enforcing that a validation must be larger than all
  unexpired validation sequence numbers previously issued by the validator
  tracked by the instance of this class.
*/
type seqEnforcer struct {
	largest Seq
	when    time.Time
}

/** Try advancing the largest observed validation ledger sequence

  Try setting the largest validation sequence observed, but return false
  if it violates the invariant that a validation must be larger than all
  unexpired validation sequence numbers.

  @param now The current time
  @param s The sequence number we want to validate
  @param p Validation parameters

  @return Whether the validation satisfies the invariant
*/
func (sf *seqEnforcer) do(now time.Time, s Seq) bool {
	if now.After(sf.when.Add(validationSETEXPIRES)) {
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

	return signTime.After(now.Add(-validationCURRENTEARLY)) &&
		signTime.Before(now.Add(validationCURRENTWALL)) &&
		(seenTime.IsZero() ||
			seenTime.Before(now.Add(validationCURRENTLOCAL)))
}

/** Status of newly received validation */
type valStatus byte

const (
	/// This was a new validation and was added
	current valStatus = iota
	/// Not current or was older than current from this node
	stale
	/// A validation violates the increasing seq requirement
	badSeq
)

func (m valStatus) string() string {
	switch m {
	case current:
		return "current"
	case stale:
		return "stale"
	case badSeq:
		return "badSeq"
	default:
		return "unknown"
	}
}

/** Maintains current and recent ledger validations.

  Manages storage and queries related to validations received on the network.
  Stores the most current validation from nodes and sets of recent
  validations grouped by ledger identifier.

  Stored validations are not necessarily from trusted nodes, so clients
  and implementations should take care to use `trusted` member functions or
  check the validation's trusted status.

  This class uses a generic interface to allow adapting Validations for
  specific applications. The Adaptor template implements a set of helper
  functions and type definitions. The code stubs below outline the
  interface and type requirements.


  @warning The Adaptor::MutexType is used to manage concurrent access to
           private members of Validations but does not manage any data in the
           Adaptor instance itself.

  @code

  // Conforms to the Ledger type requirements of LedgerTrie
  struct Ledger;

  struct Validation
  {
      using NodeID = ...;
      using NodeKey = ...;

      // Ledger ID associated with this validation
      Ledger::ID ledgerID() const;

      // Sequence number of validation's ledger (0 means no sequence number)
      Ledger::Seq seq() const

      // When the validation was signed
      NetClock::time_point signTime() const;

      // When the validation was first observed by this node
      NetClock::time_point seenTime() const;

      // Signing key of node that published the validation
      NodeKey key() const;

      // Whether the publishing node was trusted at the time the validation
      // arrived
      bool trusted() const;

      // Set the validation as trusted
      void setTrusted();

      // Set the validation as untrusted
      void setUntrusted();

      // Whether this is a full or partial validation
      bool full() const;

      // Identifier for this node that remains fixed even when rotating signing
      // keys
      NodeID nodeID()  const;

      implementation_specific_t
      unwrap() . return the implementation-specific type being wrapped

      // ... implementation specific
  };

  class Adaptor
  {
      using Mutex = std::mutex;
      using Validation = Validation;
      using Ledger = Ledger;

      // Handle a newly stale validation, this should do minimal work since
      // it is called by Validations while it may be iterating Validations
      // under lock
      void onStale(Validation && );

      // Flush the remaining validations (typically done on shutdown)
      void flush(hash_map<NodeID,Validation> && remaining);

      // Return the current network time (used to determine staleness)
      NetClock::time_point now() const;

      // Attempt to acquire a specific ledger.
      boost::optional<Ledger> acquire(Ledger::ID const & ledgerID);

      // ... implementation specific
  };
  @endcode

  @tparam Adaptor Provides type definitions and callbacks
*/
type validations struct {
	// Manages concurrent access to members
	mutex sync.Mutex

	// Validations from currently listed and trusted nodes (partial and full)
	current map[NodeID]validation

	// Used to enforce the largest validation invariant for the local node
	localSeqEnforcer seqEnforcer

	// Sequence of the largest validation received from each node
	seqEnforcers map[NodeID]seqEnforcer

	//! Validations from listed nodes, indexed by ledger id (partial and full)
	byLedger agedMap

	// Represents the ancestry of validated ledgers
	trie *ledgerTrie

	// Last (validated) ledger successfully acquired. If in this map, it is
	// accounted for in the trie.
	lastLedger map[NodeID]ledger

	// Set of ledgers being acquired from the network
	acquiring map[Seq]map[LedgerID]map[NodeID]struct{}

	// Adaptor instance
	// Is NOT managed by the mutex_ above
	adaptor adaptor
}

func (v *validations) removeTrie(nodeID NodeID, val validation) {
	if a1, ok := v.acquiring[val.Seq()]; ok {
		if a2, ok := a1[val.LedgerID()]; ok {
			delete(a2, nodeID)
			if len(a2) == 0 {
				delete(v.acquiring, val.Seq())
			}
		}
	}
	it, ok := v.lastLedger[nodeID]
	if ok && it.ID() == val.LedgerID() {
		v.trie.remove(it, 1)
		delete(v.lastLedger, nodeID)
	}
}

// Check if any pending acquire ledger requests are complete
func (v *validations) checkAcquired() {
	for it, m := range v.acquiring {
		for lid, nids := range m {
			l, err := v.adaptor.AcquireLedger(lid)
			if err != nil {
				continue
			}
			for nid := range nids {
				v.updateTrie(nid, l)
			}
			delete(v.acquiring, it)
		}
	}
}

// Update the trie to reflect a new validated ledger
func (v *validations) updateTrie(nodeID NodeID, l ledger) error {
	oldl, ok := v.lastLedger[nodeID]
	v.lastLedger[nodeID] = l
	if ok {
		v.trie.remove(oldl, 1)
	}
	return v.trie.insert(l, 1)
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
func (v *validations) updateTrie2(
	nodeID NodeID, val validation, seq Seq, id LedgerID) error {
	if !val.Trusted() {
		return errors.New("validator is not trusted")
	}

	// Clear any prior acquiring ledger for this node
	if seq != math.MaxUint64 {
		if it, ok := v.acquiring[seq]; ok {
			if it2, ok := it[id]; ok {
				delete(it2, nodeID)
				if len(it2) == 0 {
					delete(it, id)
				}
			}
		}
	}

	v.checkAcquired()

	if it, ok := v.acquiring[val.Seq()]; ok {
		if it2, ok := it[val.LedgerID()]; ok {
			it2[nodeID] = struct{}{}
		} else {
			ll, err := v.adaptor.AcquireLedger(val.LedgerID())
			if err == nil {
				if err := v.updateTrie(nodeID, ll); err != nil {
					return err
				}
			} else {
				v.acquiring[val.Seq()][val.LedgerID()][nodeID] = struct{}{}
			}
		}
	}
	return nil
}

/** Use the trie for a calculation

Accessing the trie through this helper ensures acquiring validations
are checked and any stale validations are flushed from the trie.

@param lock Existing lock of mutex_
@param f Invokable with signature (LedgerTrie<Ledger> &)

@warning The invokable `f` is expected to be a simple transformation of
		 its arguments and will be called with mutex_ under lock.

*/
func (v *validations) withTrie(f func(*ledgerTrie)) {
	// Call current to flush any stale validations
	v.doCurrent(func(i int) {}, func(n NodeID, v validation) {})
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

func (v *validations) doCurrent(pre func(int), f func(NodeID, validation)) {
	t := v.adaptor.Now()
	pre(len(v.current))
	for k, val := range v.current {
		// Check for staleness
		if !isCurrent(
			t, val.SignTime(), val.SeenTime()) {
			v.removeTrie(k, val)
			v.adaptor.OnStale(val)
			delete(v.current, k)
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
func (v *validations) loopByLedger(id LedgerID, pre func(int), f func(NodeID, validation)) {
	v.byLedger.loop(id, pre, f)
}

/** Constructor

  @param p ValidationParms to control staleness/expiration of validations
  @param c Clock to use for expiring validations stored by ledger
  @param ts Parameters for constructing Adaptor instance
*/
func newValidations(a adaptor) *validations {
	return &validations{
		byLedger:     make(agedMap),
		adaptor:      a,
		current:      make(map[NodeID]validation),
		seqEnforcers: make(map[NodeID]seqEnforcer),
		lastLedger:   make(map[NodeID]ledger),
		acquiring:    make(map[Seq]map[LedgerID]map[NodeID]struct{}),
	}
}

/** Return whether the local node can issue a validation for the given sequence
  number

  @param s The sequence number of the ledger the node wants to validate
  @return Whether the validation satisfies the invariant, updating the
          largest sequence number seen accordingly
*/
func (v *validations) canValidateSeq(s Seq) bool {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	return v.localSeqEnforcer.do(time.Now(), s)
}

/** Add a new validation

  Attempt to add a new validation.

  @param nodeID The identity of the node issuing this validation
  @param val The validation to store
  @return The outcome
*/
func (v *validations) add(nodeID NodeID, val validation) valStatus {
	if !isCurrent(v.adaptor.Now(), val.SignTime(), val.SeenTime()) {
		return stale
	}
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Check that validation sequence is greater than any non-expired
	// validations sequence from that validator
	now := time.Now()
	enforcer := v.seqEnforcers[nodeID]
	if !enforcer.do(now, val.Seq()) {
		return badSeq
	}
	// Use insert_or_assign when C++17 supported
	v.byLedger.add(val.LedgerID(), nodeID, val)
	oldVal, ok := v.current[nodeID]
	v.current[nodeID] = val
	if ok {
		// Replace existing only if this one is newer
		if val.SignTime().After(oldVal.SignTime()) {
			v.adaptor.OnStale(oldVal)
			if val.Trusted() {
				v.updateTrie2(nodeID, val, oldVal.Seq(), oldVal.LedgerID())
			} else {
				return stale
			}
		} else {
			if val.Trusted() {
				v.updateTrie2(nodeID, val, math.MaxUint64, zeroID)
			}
		}
	}
	return current
}

/** Expire old validation sets

  Remove validation sets that were accessed more than
  validationSET_EXPIRES ago.
*/
func (v *validations) expire() {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.expire(validationSETEXPIRES)
}

/** Update trust status of validations

  Updates the trusted status of known validations to account for nodes
  that have been added or removed from the UNL. This also updates the trie
  to ensure only currently trusted nodes' validations are used.

  @param added Identifiers of nodes that are now trusted
  @param removed Identifiers of nodes that are no longer trusted
*/
func (v *validations) trustChanged(added, removed map[NodeID]struct{}) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for id, val := range v.current {
		if _, ok := added[id]; ok {
			val.SetTrusted()
			v.updateTrie2(id, val, math.MaxUint64, zeroID)
		} else {
			if _, ok := removed[id]; ok {
				val.SetUntrusted()
				v.removeTrie(id, val)
			}
		}
	}
	for _, val := range v.byLedger {
		for nodeVal, val2 := range val.m {
			if _, ok := added[nodeVal]; ok {
				val2.SetTrusted()
			} else {
				if _, ok := removed[nodeVal]; ok {
					val2.SetUntrusted()
				}
			}
		}
	}
}

/** Return the sequence number and ID of the preferred working ledger

  A ledger is preferred if it has more support amongst trusted validators
  and is *not* an ancestor of the current working ledger; otherwise it
  remains the current working ledger.

  @param curr The local node's current working ledger

  @return The sequence and id of the preferred working ledger,
          or Seq{0},ID{0} if no trusted validations are available to
          determine the preferred ledger.
*/
func (v *validations) getPreferred(curr ledger) (Seq, LedgerID, error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	var preferred *spanTip
	var err error
	v.withTrie(func(trie *ledgerTrie) {
		preferred, err = trie.getPreferred(v.localSeqEnforcer.largest)
	})
	if err != nil {
		return 0, zeroID, err
	}

	// No trusted validations to determine branch
	if preferred.seq == 0 {
		var maxseq Seq
		var maxid LedgerID
		var size int
		for s, ln := range v.acquiring {
			for l, n := range ln {
				if size < len(n) || (size == len(n) && bytes.Compare(maxid[:], l[:]) < 0) {
					maxseq = s
					maxid = l
					size = len(n)
				}
			}
		}
		if len(v.acquiring) != 0 {
			return maxseq, maxid, nil
		}
		return preferred.seq, preferred.id, nil
	}

	// If we are the parent of the preferred ledger, stick with our
	// current ledger since we might be about to generate it
	lid := preferred.ancestor(curr.Seq())
	if preferred.seq == curr.Seq()+1 && lid == curr.ID() {
		return curr.Seq(), curr.ID(), nil
	}

	// A ledger ahead of us is preferred regardless of whether it is
	// a descendant of our working ledger or it is on a different chain
	if preferred.seq > curr.Seq() {
		return preferred.seq, preferred.id, nil
	}
	// Only switch to earlier or same sequence number
	// if it is a different chain.
	if curr.IndexOf(preferred.seq) != preferred.id {
		return preferred.seq, preferred.id, nil
	}
	// Stick with current ledger
	return curr.Seq(), curr.ID(), nil
}

/** Get the ID of the preferred working ledger that exceeds a minimum valid
  ledger sequence number

  @param curr Current working ledger
  @param minValidSeq Minimum allowed sequence number

  @return ID Of the preferred ledger, or curr if the preferred ledger
             is not valid
*/

func (v *validations) getPreferred2(curr ledger, minValidSeq Seq) (LedgerID, error) {
	preferredSeq, preferredID, err := v.getPreferred(curr)
	if err != nil {
		return zeroID, err
	}
	if preferredSeq >= minValidSeq && preferredID != zeroID {
		return preferredID, nil
	}
	return curr.ID(), nil
}

/** Determine the preferred last closed ledger for the next consensus round.

  Called before starting the next round of ledger consensus to determine the
  preferred working ledger. Uses the dominant peerCount ledger if no
  trusted validations are available.

  @param lcl Last closed ledger by this node
  @param minSeq Minimum allowed sequence number of the trusted preferred ledger
  @param peerCounts Map from ledger ids to count of peers with that as the
                    last closed ledger
  @return The preferred last closed ledger ID

  @note The minSeq does not apply to the peerCounts, since this function
        does not know their sequence number
*/
func (v *validations) getPreferredLCL(lcl ledger, minSeq Seq, peerCounts map[LedgerID]uint32) (LedgerID, error) {
	preferredSeq, preferredID, err := v.getPreferred(lcl)
	if err != nil {
		return zeroID, err
	}
	// Trusted validations exist
	if preferredID != zeroID && preferredSeq > 0 {
		if preferredSeq > minSeq {
			return preferredID, nil
		}
		return lcl.ID(), nil
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
		return maxid, nil
	}
	return lcl.ID(), nil
}

/** Count the number of current trusted validators working on a ledger
  after the specified one.

  @param ledger The working ledger
  @param ledgerID The preferred ledger
  @return The number of current trusted validators working on a descendant
          of the preferred ledger

  @note If ledger.id() != ledgerID, only counts immediate child ledgers of
        ledgerID
*/
func (v *validations) getNodesAfter(l ledger, id LedgerID) uint32 {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// Use trie if ledger is the right one
	var num uint32
	if l.ID() == id {
		v.withTrie(func(trie *ledgerTrie) {
			n := trie.branchSupport(l)
			num = -n
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

/** Get the currently trusted full validations
  @return Vector of validations from currently trusted validators
*/
func (v *validations) currentTrusted() []validation {
	var ret []validation
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.doCurrent(
		func(numValidations int) {
			ret = make([]validation, 0, numValidations)
		},
		func(n NodeID, v validation) {
			if v.Trusted() && v.Full() {
				ret = append(ret, v)
			}
		})
	return ret
}

// /** Get the set of node ids associated with current validations
//     @return The set of node ids for active, listed validators
// */
func (v *validations) getCurrentNodeIDs() map[NodeID]struct{} {
	ret := make(map[NodeID]struct{})
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.doCurrent(
		func(numValidations int) {
		},
		func(nid NodeID, v validation) {
			ret[nid] = struct{}{}
		})

	return ret
}

// /** Count the number of trusted full validations for the given ledger

//     @param ledgerID The identifier of ledger of interest
//     @return The number of trusted validations
// */
func (v *validations) numTrustedForLedger(id LedgerID) uint {
	var count uint
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.loop(
		id,
		func(i int) {}, // nothing to reserve
		func(nid NodeID, v validation) {
			if v.Trusted() && v.Full() {
				count++
			}
		})
	return count
}

// /**  Get trusted full validations for a specific ledger

//      @param ledgerID The identifier of ledger of interest
//      @return Trusted validations associated with ledger
// */
func (v *validations) getTrustedForLedger(id LedgerID) []validation {
	var res []validation
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.loop(
		id,
		func(numValidations int) {
			res = make([]validation, 0, numValidations)
		},
		func(nid NodeID, v validation) {
			if v.Trusted() && v.Full() {
				res = append(res, v)
			}
		})

	return res
}

// /** Returns fees reported by trusted full validators in the given ledger

//     @param ledgerID The identifier of ledger of interest
//     @param baseFee The fee to report if not present in the validation
//     @return Vector of fees
// */
func (v *validations) fees(id LedgerID, baseFee uint32) []uint32 {
	var res []uint32
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.byLedger.loop(
		id,
		func(numValidations int) {
			res = make([]uint32, 0, numValidations)
		},
		func(nid NodeID, v validation) {
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

// /** Flush all current validations
//  */
func (v *validations) flush() {
	flushed := make(map[NodeID]validation)
	v.mutex.Lock()
	defer v.mutex.Unlock()
	for nid, v := range v.current {
		flushed[nid] = v
	}
	v.current = make(map[NodeID]validation)
	v.adaptor.Flush(flushed)
}
