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
	"fmt"
	"sort"
	"strings"
)

/** The tip of a span of ledger ancestry
 */
type spanTip struct {
	// The sequence number of the tip ledger
	seq Seq
	// The ID of the tip ledger
	id     LedgerID
	ledger Ledger
}

/** Lookup the ID of an ancestor of the tip ledger

  @param s The sequence number of the ancestor
  @return The ID of the ancestor with that sequence number

  @note s must be less than or equal to the sequence number of the
        tip ledger
*/
func (st *spanTip) ancestor(s Seq) LedgerID {
	if s > st.seq {
		panic("invalid sequence")
	}
	return st.ledger.IndexOf(s)
}

type span struct {
	start  Seq
	end    Seq //1
	ledger Ledger
}

func newSpanFromLedger(l Ledger) *span {
	return &span{
		ledger: l,
		end:    l.Seq() + 1,
	}
}

//from Return the Span from [spot,end_) or none if no such valid span
func (s *span) from(spot Seq) (*span, error) {
	return s.sub(spot, s.end)
}

// before Return the Span from [start_,spot) or none if no such valid span
func (s *span) before(spot Seq) (*span, error) {
	return s.sub(s.start, spot)
}

// Return the ID of the ledger that starts this span
func (s *span) startID() LedgerID {
	return s.ledger.IndexOf(s.start)
}

// Return the ledger sequence number of the first possible difference
// between this span and a given ledger.
func (s *span) diff(o Ledger) Seq {
	end := s.ledger.Seq()
	if end > o.Seq() {
		end = o.Seq()
	}
	var i Seq
	for i = 0; i <= end; i++ {
		if s.ledger.IndexOf(i) != o.IndexOf(i) {
			break
		}
	}
	return s.clamp(i)
}

//  The tip of this span
func (s *span) tip() *spanTip {
	return &spanTip{
		seq:    s.end - 1,
		id:     s.ledger.IndexOf(s.end - 1),
		ledger: s.ledger,
	}
}

func (s *span) clamp(val Seq) Seq {
	tmp := s.start
	if tmp < val {
		tmp = val
	}
	if tmp > s.end {
		tmp = s.end
	}
	return tmp
}

// Return a span of this over the half-open interval [from,to)
func (s *span) sub(from, to Seq) (*span, error) {
	newFrom := s.clamp(from)
	newTo := s.clamp(to)
	if newFrom >= newTo {
		return nil, errors.New("invalid from or to")
	}
	return &span{
		start:  newFrom,
		end:    newTo,
		ledger: s.ledger,
	}, nil
}
func (s *span) String() string {
	return fmt.Sprint(s.tip(), "[", s.start, ",", s.end, ")")
}

// Return combined span, using ledger_ from higher sequence span
func mergeSpan(a, b *span) *span {
	start := a.start
	if start > b.start {
		start = b.start
	}
	if a.end < b.end {
		return &span{
			start:  start,
			end:    b.end,
			ledger: b.ledger,
		}
	}
	return &span{
		start:  start,
		end:    a.end,
		ledger: a.ledger,
	}
}

// A node in the trie
type node struct {
	span          *span
	tipSupport    uint32
	branchSupport uint32
	children      []*node
	parent        *node
}

func newNode(l Ledger) *node {
	if l.Seq() == 0 {
		return &node{
			span: newSpanFromLedger(l),
		}
	}
	return &node{
		span:          newSpanFromLedger(l),
		tipSupport:    1,
		branchSupport: 1,
	}
}
func newNodeFromSpan(s *span) *node {
	return &node{
		span: s,
	}
}

/** Remove the given node from this Node's children

  @param child The address of the child node to remove
  @note The child must be a member of the vector. The passed pointer
        will be dangling as a result of this call
*/
func (n *node) erase(child *node) {
	for i, c := range n.children {
		if child == c {
			copy(n.children[i:], n.children[i+1:])
			n.children[len(n.children)-1] = nil
			n.children = n.children[:len(n.children)-1]
			return
		}
	}
	panic("child not found")
}

func (n *node) String() string {
	return fmt.Sprint(n.span, "(T", n.tipSupport, ",B:", n.branchSupport, ")")
}

/** Ancestry trie of ledgers

  A compressed trie tree that maintains validation support of recent ledgers
  based on their ancestry.

  The compressed trie structure comes from recognizing that ledger history
  can be viewed as a string over the alphabet of ledger ids. That is,
  a given ledger with sequence number `seq` defines a length `seq` string,
  with i-th entry equal to the id of the ancestor ledger with sequence
  number i. "Sequence" strings with a common prefix share those ancestor
  ledgers in common. Tracking this ancestry information and relations across
  all validated ledgers is done conveniently in a compressed trie. A node in
  the trie is an ancestor of all its children. If a parent node has sequence
  number `seq`, each child node has a different ledger starting at `seq+1`.
  The compression comes from the invariant that any non-root node with 0 tip
  support has either no children or multiple children. In other words, a
  non-root 0-tip-support node can be combined with its single child.

  Each node has a tipSupport, which is the number of current validations for
  that particular ledger. The node's branch support is the sum of the tip
  support and the branch support of that node's children:

      @code
      node.branchSupport = node.tipSupport;
      for (child : node.children)
         node.branchSupport += child.branchSupport;
      @endcode

  The templated Ledger type represents a ledger which has a unique history.
  It should be lightweight and cheap to copy.

     @code
     // Identifier types that should be equality-comparable and copyable
     struct ID;
     struct Seq;

     struct Ledger
     {
        struct MakeGenesis{};

        // The genesis ledger represents a ledger that prefixes all other
        // ledgers
        Ledger(MakeGenesis{});

        Ledger(Ledger const&);
        Ledger& operator=(Ledger const&);

        // Return the sequence number of this ledger
        Seq seq() const;

        // Return the ID of this ledger's ancestor with given sequence number
        // or ID{0} if unknown
        ID
        operator[](Seq s);

     };

     // Return the sequence number of the first possible mismatching ancestor
     // between two ledgers
     Seq
     mismatch(ledgerA, ledgerB);
     @endcode

  The unique history invariant of ledgers requires any ledgers that agree
  on the id of a given sequence number agree on ALL ancestors before that
  ledger:

      @code
      Ledger a,b;
      // For all Seq s:
      if(a[s] == b[s]);
          for(Seq p = 0; p < s; ++p)
              assert(a[p] == b[p]);
      @endcode

  @tparam Ledger A type representing a ledger and its history
*/

type ledgerTrie struct {

	// The root of the trie. The root is allowed to break the no-single child
	// invariant.
	root *node

	// Count of the tip support for each sequence number
	seqSupport map[Seq]uint32
}

/** Find the node in the trie that represents the longest common ancestry
  with the given ledger.

  @return Pair of the found node and the sequence number of the first
          ledger difference.
*/
func (lt *ledgerTrie) find(l Ledger) (*node, Seq) {
	curr := lt.root
	// Root is always defined and is in common with all ledgers
	if curr == nil {
		panic("root is nil")
	}
	pos := curr.span.diff(l)
	done := false
	// Continue searching for a better span as long as the current position
	// matches the entire span
	for !done && pos == curr.span.end {
		done = true
		// Find the child with the longest ancestry match
		for _, child := range curr.children {
			childPos := child.span.diff(l)
			if childPos > pos {
				done = false
				pos = childPos
				curr = child
				break
			}
		}
	}
	return curr, pos
}

func (lt *ledgerTrie) dumpImpl(curr *node, offset int) string {
	var str strings.Builder
	if curr != nil {
		if offset > 0 {
			for i := 2; i < offset; i++ {
				str.WriteString(" ")
			}
			str.WriteString("|-")
		}
		str.WriteString(curr.String())
		for _, child := range curr.children {
			str.WriteString(lt.dumpImpl(child, offset+1+len(curr.String())+2))
		}
	}
	return str.String()
}

func newLedgerTrie(genesis Ledger) *ledgerTrie {
	return &ledgerTrie{
		root:       newNode(genesis),
		seqSupport: make(map[Seq]uint32),
	}
}

/** Insert and/or increment the support for the given ledger.

  @param ledger A ledger and its ancestry
  @param count The count of support for this ledger
*/
func (lt *ledgerTrie) insert(l Ledger, count /* =1 */ uint32) {
	loc, diffSeq := lt.find(l)
	// There is always a place to insert
	if loc == nil {
		panic("ledger not found")
	}
	// Node from which to start incrementing branchSupport
	incNode := loc

	// loc.span has the longest common prefix with Span{ledger} of all
	// existing nodes in the trie. The optional<Span>'s below represent
	// the possible common suffixes between loc.span and Span{ledger}.
	//
	// loc.span
	//  a b c  | d e f
	//  prefix | oldSuffix
	//
	// Span{ledger}
	//  a b c  | g h i
	//  prefix | newSuffix

	prefix, err := loc.span.before(diffSeq)
	// Loc truncates to prefix and newNode is its child
	if err != nil {
		panic(err)
	}
	oldSuffix, errOldSuffix := loc.span.from(diffSeq)
	newSuffix, errNewSuffix := newSpanFromLedger(l).from(diffSeq)

	if errOldSuffix == nil {
		// Have
		//   abcdef . ....
		// Inserting
		//   abc
		// Becomes
		//   abc . def . ...

		// Create oldSuffix node that takes over loc
		nNode := newNodeFromSpan(oldSuffix)
		nNode.tipSupport = loc.tipSupport
		nNode.branchSupport = loc.branchSupport
		nNode.children = loc.children
		loc.children = nil
		for _, child := range nNode.children {
			child.parent = nNode
		}

		loc.span = prefix
		nNode.parent = loc
		loc.children = append(loc.children, nNode)
		loc.tipSupport = 0
	}
	if errNewSuffix == nil {
		// Have
		//  abc . ...
		// Inserting
		//  abcdef. ...
		// Becomes
		//  abc . ...
		//     \. def

		nNode := newNodeFromSpan(newSuffix)
		nNode.parent = loc
		// increment support starting from the new node
		incNode = nNode
		loc.children = append(loc.children, nNode)
	}

	incNode.tipSupport += count
	for incNode != nil {
		incNode.branchSupport += count
		incNode = incNode.parent
	}
	lt.seqSupport[l.Seq()] += count
}

/** Decrease support for a ledger, removing and compressing if possible.

  @param ledger The ledger history to remove
  @param count The amount of tip support to remove

  @return Whether a matching node was decremented and possibly removed.
*/
func (lt *ledgerTrie) remove(l Ledger, count uint32 /* =1 */) bool {
	loc, diffSeq := lt.find(l)

	// Cannot erase root
	if loc == nil || loc == lt.root {
		return false
	}
	// Must be exact match with tip support
	if diffSeq != loc.span.end || diffSeq <= l.Seq() ||
		loc.tipSupport == 0 {
		return false
	}
	if count > loc.tipSupport {
		count = loc.tipSupport
	}
	loc.tipSupport -= count

	if sup, exist := lt.seqSupport[l.Seq()]; !exist || sup < count {
		panic("ledger not found")
	}
	lt.seqSupport[l.Seq()] -= count
	if lt.seqSupport[l.Seq()] == 0 {
		delete(lt.seqSupport, l.Seq())
	}
	decNode := loc
	for decNode != nil {
		decNode.branchSupport -= count
		decNode = decNode.parent
	}

loop:
	for loc.tipSupport == 0 && loc != lt.root {
		parent := loc.parent
		switch len(loc.children) {
		case 1:
			// This node can be combined with its child
			child := loc.children[0]
			child.span = mergeSpan(loc.span, child.span)
			child.parent = parent
			parent.children = append(parent.children, child)
			fallthrough
		case 0:
			// this node can be erased
			parent.erase(loc)
			loc = parent
		default:
			break loop
		}
	}
	return true
}

/** Return count of tip support for the specific ledger.

  @param ledger The ledger to lookup
  @return The number of entries in the trie for this *exact* ledger
*/
func (lt *ledgerTrie) tipSupport(l Ledger) uint32 {
	loc, diffSeq := lt.find(l)

	// Exact match
	if loc != nil && diffSeq == loc.span.end && diffSeq > l.Seq() {
		return loc.tipSupport
	}
	return 0
}

/** Return the count of branch support for the specific ledger

@param ledger The ledger to lookup
@return The number of entries in the trie for this ledger or a descendant
*/
func (lt *ledgerTrie) branchSupport(l Ledger) uint32 {
	loc, diffSeq := lt.find(l)
	// Check that ledger is is an exact match or proper
	// prefix of loc
	if loc != nil && diffSeq > l.Seq() &&
		l.Seq() < loc.span.end {
		return loc.branchSupport
	}
	return 0
}
func (lt *ledgerTrie) keys() []Seq {
	keys := make([]Seq, 0, len(lt.seqSupport))
	for k := range lt.seqSupport {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}

/** Return the preferred ledger ID

  The preferred ledger is used to determine the working ledger
  for consensus amongst competing alternatives.

  Recall that each validator is normally validating a chain of ledgers,
  e.g. A.B.C.D. However, if due to network connectivity or other
  issues, validators generate different chains

  @code
         /.C
     A.B
         \.D.E
  @endcode

  we need a way for validators to converge on the chain with the most
  support. We call this the preferred ledger.  Intuitively, the idea is to
  be conservative and only switch to a different branch when you see
  enough peer validations to *know* another branch won't have preferred
  support.

  The preferred ledger is found by walking this tree of validated ledgers
  starting from the common ancestor ledger.

  At each sequence number, we have

     - The prior sequence preferred ledger, e.g. B.
     - The (tip) support of ledgers with this sequence number,e.g. the
       number of validators whose last validation was for C or D.
     - The (branch) total support of all descendants of the current
       sequence number ledgers, e.g. the branch support of D is the
       tip support of D plus the tip support of E; the branch support of
       C is just the tip support of C.
     - The number of validators that have yet to validate a ledger
       with this sequence number (uncommitted support). Uncommitted
       includes all validators whose last sequence number is smaller than
       our last issued sequence number, since due to asynchrony, we may
       not have heard from those nodes yet.

  The preferred ledger for this sequence number is then the ledger
  with relative majority of support, where uncommitted support
  can be given to ANY ledger at that sequence number
  (including one not yet known). If no such preferred ledger exists, then
  the prior sequence preferred ledger is the overall preferred ledger.

  In this example, for D to be preferred, the number of validators
  supporting it or a descendant must exceed the number of validators
  supporting C _plus_ the current uncommitted support. This is because if
  all uncommitted validators end up validating C, that new support must
  be less than that for D to be preferred.

  If a preferred ledger does exist, then we continue with the next
  sequence using that ledger as the root.

  @param largestIssued The sequence number of the largest validation
                       issued by this node.
  @return Pair with the sequence number and ID of the preferred ledger
*/
func (lt *ledgerTrie) getPreferred(largestIssued Seq) *spanTip {
	curr := lt.root
	done := false
	uncommittedIt := lt.keys()
	var uncommited uint32
	it := 0

	for curr != nil && !done {
		// Within a single span, the preferred by branch strategy is simply
		// to continue along the span as long as the branch support of
		// the next ledger exceeds the uncommitted support for that ledger.
		{
			// Add any initial uncommitted support prior for ledgers
			// earlier than nextSeq or earlier than largestIssued
			nextSeq := curr.span.start + 1
			maxSeq := nextSeq
			if maxSeq < largestIssued {
				maxSeq = largestIssued
			}
			for ; it < len(uncommittedIt) && uncommittedIt[it] < maxSeq; it++ {
				uncommited += lt.seqSupport[uncommittedIt[it]]
			}
			// Advance nextSeq along the span
			for nextSeq < curr.span.end &&
				curr.branchSupport > uncommited {
				// Jump to the next seqSupport change
				if it != len(uncommittedIt) &&
					uncommittedIt[it] < curr.span.end {
					nextSeq = uncommittedIt[it] + 1
					uncommited += lt.seqSupport[uncommittedIt[it]]
					it++
				} else { // otherwise we jump to the end of the span
					nextSeq = curr.span.end
				}
			}
			// We did not consume the entire span, so we have found the
			// preferred ledger
			if nextSeq < curr.span.end {
				sp, err := curr.span.before(nextSeq)
				if err != nil {
					panic(err)
				}
				return sp.tip()
			}
		}

		// We have reached the end of the current span, so we need to
		// find the best child
		var best *node
		var margin uint32
		if len(curr.children) == 1 {
			best = curr.children[0]
			margin = best.branchSupport
		} else if len(curr.children) != 0 {
			// Sort placing children with largest branch support in the
			// front, breaking ties with the span's starting ID
			sort.Slice(curr.children, func(i, j int) bool {
				b := int(curr.children[i].branchSupport) - int(curr.children[j].branchSupport)
				if b != 0 {
					return b > 0
				}
				idi := curr.children[i].span.startID()
				idj := curr.children[j].span.startID()
				return bytes.Compare(idi[:], idj[:]) > 0
			})
			best = curr.children[0]
			margin = curr.children[0].branchSupport -
				curr.children[1].branchSupport
			// If best holds the tie-breaker, gets one larger margin
			// since the second best needs additional branchSupport
			// to overcome the tie
			idi := best.span.startID()
			idj := curr.children[1].span.startID()
			if bytes.Compare(idi[:], idj[:]) > 0 {
				margin++
			}
		}

		// If the best child has margin exceeding the uncommitted support,
		// continue from that child, otherwise we are done
		if best != nil && ((margin > uncommited) || (uncommited == 0)) {
			curr = best
		} else { // current is the best
			done = true
		}
	}
	return curr.span.tip()
}

/** Dump an ascii representation of the trie to the stream*/
func (lt *ledgerTrie) dump() string {
	return lt.dumpImpl(lt.root, 0)
}

/** Check the compressed trie and support invariants.
 */

func (lt *ledgerTrie) checkInvariants() error {

	expectedSeqSupport := make(map[Seq]uint32)

	nodes := []*node{lt.root}
	for len(nodes) > 0 {
		curr := nodes[0]
		nodes = nodes[1:]
		if curr == nil {
			continue
		}
		// Node with 0 tip support must have multiple children
		// unless it is the root node
		if curr != lt.root && curr.tipSupport == 0 &&
			len(curr.children) < 2 {
			return errors.New("invalid tipsupport")
		}
		// branchSupport = tipSupport + sum(child.branchSupport)
		support := curr.tipSupport
		if curr.tipSupport != 0 {
			expectedSeqSupport[curr.span.end-1] += curr.tipSupport
		}

		for _, child := range curr.children {
			if child == nil || child.parent != curr {
				return errors.New("invalid child ")
			}
			support += child.branchSupport
			nodes = append(nodes, child)
		}
		if support != curr.branchSupport {
			return errors.New("invalid support")
		}
	}
	if len(expectedSeqSupport) != len(lt.seqSupport) {
		return errors.New("invalid #seqsupport")
	}
	for k, v := range expectedSeqSupport {
		if vv, ok := lt.seqSupport[k]; !ok || vv != v {
			return errors.New("invalid seqsupport")
		}
	}
	return nil
}
