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
	ledger ledger
}

/** Lookup the ID of an ancestor of the tip ledger

  @param s The sequence number of the ancestor
  @return The ID of the ancestor with that sequence number

  @note s must be less than or equal to the sequence number of the
        tip ledger
*/
func (st *spanTip) ancestor(s Seq) (LedgerID, error) {
	if s > st.seq {
		return zeroID, errors.New("invalid sequence")
	}
	return st.ledger.IndexOf(s), nil
}

type span struct {
	start  Seq
	End    Seq
	ledger ledger
}

func newSpan(l ledger) *span {
	return &span{
		ledger: l,
		End:    l.Seq() + 1,
	}
}

//From Return the Span from [spot,end_) or none if no such valid span
func (s *span) From(spot Seq) (*span, error) {
	return s.sub(spot, s.End)
}

// Before Return the Span from [start_,spot) or none if no such valid span
func (s *span) Before(spot Seq) (*span, error) {
	return s.sub(s.start, spot)
}

// Return the ID of the ledger that starts this span
func (s *span) startID() LedgerID {
	return s.ledger.IndexOf(s.start)
}

// Return the ledger sequence number of the first possible difference
// between this span and a given ledger.
func (s *span) Diff(o ledger) Seq {
	length := s.ledger.Seq()
	if length > o.Seq() {
		length = o.Seq()
	}
	var i Seq
	for i = 0; i < length; i++ {
		if s.ledger.ID() != o.ID() {
			break
		}
	}
	return i
}

//  The tip of this span
func (s *span) tip() *spanTip {
	return &spanTip{
		seq:    s.End - 1,
		id:     s.ledger.IndexOf(s.End - 1),
		ledger: s.ledger,
	}
}

func (s *span) clamp(val Seq) Seq {
	tmp := s.start
	if tmp < val {
		tmp = val
	}
	if tmp > s.End {
		tmp = s.End
	}
	return tmp
}

// Return a span of this over the half-open interval [from,to)
func (s *span) sub(from, to Seq) (*span, error) {
	newFrom := s.clamp(from)
	newTo := s.clamp(to)
	if newFrom < newTo {
		return &span{
			start:  newFrom,
			End:    newTo,
			ledger: s.ledger,
		}, nil
	}
	return nil, errors.New("invalid from or to")
}
func (s *span) String() string {
	return fmt.Sprint(s.tip(), "[", s.start, ",", s.End, ")")
}

// Return combined span, using ledger_ from higher sequence span
func merge(a, b *span) *span {
	start := a.start
	if start > b.start {
		start = b.start
	}
	if a.End < b.End {
		return &span{
			start:  start,
			End:    b.End,
			ledger: b.ledger,
		}
	}
	return &span{
		start:  start,
		End:    a.End,
		ledger: a.ledger,
	}
}

// A node in the trie
type node struct {
	Span          *span
	TipSupport    uint32
	BranchSupport uint32
	Children      []*node
	Parent        *node
}

func newNode(l ledger) *node {
	return &node{
		Span:          newSpan(l),
		TipSupport:    1,
		BranchSupport: 1,
	}
}
func newNode2(s *span) *node {
	return &node{
		Span: s,
	}
}

/** Remove the given node from this Node's children

  @param child The address of the child node to remove
  @note The child must be a member of the vector. The passed pointer
        will be dangling as a result of this call
*/
func (n *node) erase(child *node) error {
	for i, c := range n.Children {
		if child == c {
			copy(n.Children[:i], n.Children[i+1:])
			n.Children[i-1] = nil
			return nil
		}
	}
	return errors.New("child not found")
}

func (n *node) String() string {
	return fmt.Sprint(n.Span.String(), "(T", n.TipSupport, ",B:", n.BranchSupport, ")")
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
func (lt *ledgerTrie) find(l ledger) (*node, Seq, error) {
	curr := lt.root
	// Root is always defined and is in common with all ledgers
	if curr == nil {
		return nil, 0, errors.New("root is nil")
	}
	pos := curr.Span.Diff(l)

	done := false

	// Continue searching for a better span as long as the current position
	// matches the entire span
	for !done && pos == curr.Span.End {
		done = true
		// Find the child with the longest ancestry match
		for _, child := range curr.Children {
			childPos := child.Span.Diff(l)
			if childPos > pos {
				done = false
				pos = childPos
				curr = child
				break
			}
		}
	}
	return curr, pos, nil
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
		for _, child := range curr.Children {
			str.WriteString(lt.dumpImpl(child, offset+1+len(curr.String())+2))
		}
	}
	return str.String()
}

func newLedgerTrie() *ledgerTrie {
	return &ledgerTrie{
		root: &node{},
	}
}

/** Insert and/or increment the support for the given ledger.

  @param ledger A ledger and its ancestry
  @param count The count of support for this ledger
*/
func (lt *ledgerTrie) insert(l ledger, count uint32) error {
	loc, diffSeq, err := lt.find(l)
	if err != nil {
		return err
	}
	// There is always a place to insert
	if loc == nil {
		return errors.New("ledger not found")
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

	prefix, err := loc.Span.Before(diffSeq)
	if err != nil {
		return err
	}
	oldSuffix, err := loc.Span.From(diffSeq)
	if err != nil {
		return err
	}
	newSuffix, err := newSpan(l).From(diffSeq)
	if err != nil {
		return err
	}

	if oldSuffix != nil {
		// Have
		//   abcdef . ....
		// Inserting
		//   abc
		// Becomes
		//   abc . def . ...

		// Create oldSuffix node that takes over loc
		nNode := newNode2(oldSuffix)
		nNode.TipSupport = loc.TipSupport
		nNode.BranchSupport = loc.BranchSupport
		copy(nNode.Children, loc.Children)
		loc.Children = nil
		for _, child := range nNode.Children {
			child.Parent = nNode
		}

		// Loc truncates to prefix and newNode is its child
		if prefix == nil {
			return errors.New("prefix is null")
		}
		loc.Span = prefix
		nNode.Parent = loc
		loc.Children = append(loc.Children, nNode)
		loc.TipSupport = 0
	}
	if newSuffix != nil {
		// Have
		//  abc . ...
		// Inserting
		//  abcdef. ...
		// Becomes
		//  abc . ...
		//     \. def

		nNode := newNode2(newSuffix)
		nNode.Parent = loc
		// increment support starting from the new node
		incNode = nNode
		loc.Children = append(loc.Children, nNode)
	}

	incNode.TipSupport += count
	for incNode != nil {
		incNode.BranchSupport += count
		incNode = incNode.Parent
	}

	lt.seqSupport[l.Seq()] += count
	return nil
}

/** Decrease support for a ledger, removing and compressing if possible.

  @param ledger The ledger history to remove
  @param count The amount of tip support to remove

  @return Whether a matching node was decremented and possibly removed.
*/
func (lt *ledgerTrie) remove(l ledger, count uint32) (bool, error) {
	loc, diffSeq, err := lt.find(l)
	if err != nil {
		return false, err
	}

	// Cannot erase root
	if loc != nil && loc != lt.root {
		// Must be exact match with tip support
		if diffSeq == loc.Span.End && diffSeq > l.Seq() &&
			loc.TipSupport > 0 {
			if count > loc.TipSupport {
				count = loc.TipSupport
			}
			loc.TipSupport -= count

			it, exist := lt.seqSupport[l.Seq()]
			if !exist {
				return false, errors.New("ledger not found")
			}
			it -= count
			if it == 0 {
				delete(lt.seqSupport, l.Seq())
			}
			decNode := loc
			for decNode != nil {
				decNode.BranchSupport -= count
				decNode = decNode.Parent
			}

			for loc.TipSupport == 0 && loc != lt.root {
				parent := loc.Parent
				if len(loc.Children) == 0 {
					// this node can be erased
					parent.erase(loc)
				} else if len(loc.Children) == 1 {
					// This node can be combined with its child
					child := loc.Children[0]
					child.Span = merge(loc.Span, child.Span)
					child.Parent = parent
					parent.Children = append(parent.Children, child)
					parent.erase(loc)
				} else {
					break
				}
				loc = parent
			}
			return false, nil
		}
	}
	return false, nil
}

/** Return count of tip support for the specific ledger.

  @param ledger The ledger to lookup
  @return The number of entries in the trie for this *exact* ledger
*/
func (lt *ledgerTrie) tipSupport(l ledger) (uint32, error) {
	loc, diffSeq, err := lt.find(l)
	if err != nil {
		return 0, err
	}

	// Exact match
	if loc != nil && diffSeq == loc.Span.End && diffSeq > l.Seq() {
		return loc.TipSupport, nil
	}
	return 0, nil
}

/** Return the count of branch support for the specific ledger

@param ledger The ledger to lookup
@return The number of entries in the trie for this ledger or a descendant
*/
func (lt *ledgerTrie) branchSupport(l ledger) (uint32, error) {
	loc, diffSeq, err := lt.find(l)
	if err != nil {
		return 0, err
	}

	// Check that ledger is is an exact match or proper
	// prefix of loc
	if loc != nil && diffSeq > l.Seq() &&
		l.Seq() < loc.Span.End {
		return loc.BranchSupport, nil
	}
	return 0, nil
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
func (lt *ledgerTrie) getPreferred(largestIssued Seq) (*spanTip, error) {
	curr := lt.root
	done := false
	uncommittedIt := lt.keys()
	it := 0
	var uncommited uint32

	for curr != nil && !done {
		// Within a single span, the preferred by branch strategy is simply
		// to continue along the span as long as the branch support of
		// the next ledger exceeds the uncommitted support for that ledger.
		{
			// Add any initial uncommitted support prior for ledgers
			// earlier than nextSeq or earlier than largestIssued
			nextSeq := curr.Span.start + 1
			seq := nextSeq
			if seq < largestIssued {
				seq = largestIssued
			}
			for it != len(uncommittedIt) && uncommittedIt[it] < seq {
				uncommited += lt.seqSupport[uncommittedIt[it]]
				it++
			}

			// Advance nextSeq along the span
			for nextSeq < curr.Span.End &&
				curr.BranchSupport > uncommited {
				// Jump to the next seqSupport change
				if it != len(uncommittedIt) &&
					uncommittedIt[it] < curr.Span.End {
					nextSeq = uncommittedIt[it] + 1
					uncommited += lt.seqSupport[uncommittedIt[it]]
					it++
				} else { // otherwise we jump to the end of the span
					nextSeq = curr.Span.End
				}
			}
			// We did not consume the entire span, so we have found the
			// preferred ledger
			if nextSeq < curr.Span.End {
				sp, err := curr.Span.Before(nextSeq)
				if err != nil {
					return nil, err
				}
				return sp.tip(), nil
			}
		}

		// We have reached the end of the current span, so we need to
		// find the best child
		var best *node
		var margin uint32
		if len(curr.Children) == 1 {
			best = curr.Children[0]
			margin = best.BranchSupport
		} else if len(curr.Children) != 0 {
			// Sort placing children with largest branch support in the
			// front, breaking ties with the span's starting ID
			sort.Slice(curr.Children, func(i, j int) bool {
				b := curr.Children[i].BranchSupport - curr.Children[j].BranchSupport
				if b != 0 {
					return b > 0
				}
				idi := curr.Children[i].Span.startID()
				idj := curr.Children[j].Span.startID()
				return bytes.Compare(idi[:], idj[:]) > 0
			})
			best = curr.Children[0]
			margin = curr.Children[0].BranchSupport -
				curr.Children[1].BranchSupport

			// If best holds the tie-breaker, gets one larger margin
			// since the second best needs additional branchSupport
			// to overcome the tie
			idi := best.Span.startID()
			idj := curr.Children[1].Span.startID()
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
	return curr.Span.tip(), nil
}

/** Dump an ascii representation of the trie to the stream*/
func (lt *ledgerTrie) dump() string {
	return lt.dumpImpl(lt.root, 0)
}

/** Check the compressed trie and support invariants.
 */

func (lt *ledgerTrie) checkInvariants() bool {

	expectedSeqSupport := make(map[Seq]uint32)

	nodes := []*node{lt.root}
	for len(nodes) > 0 {
		curr := nodes[0]
		nodes = nodes[:len(nodes)-1]
		if curr == nil {
			continue
		}
		// Node with 0 tip support must have multiple children
		// unless it is the root node
		if curr != lt.root && curr.TipSupport == 0 &&
			len(curr.Children) < 2 {
			return false
		}
		// branchSupport = tipSupport + sum(child.branchSupport)
		support := curr.TipSupport
		if curr.TipSupport != 0 {
			expectedSeqSupport[curr.Span.End-1] += curr.TipSupport
		}

		for _, child := range curr.Children {
			if child.Parent != curr {
				return false
			}
			support += child.BranchSupport
			nodes = append(nodes, child)
		}
		if support != curr.BranchSupport {
			return false
		}
	}
	if len(expectedSeqSupport) != len(lt.seqSupport) {
		return false
	}
	for k, v := range expectedSeqSupport {
		if vv, ok := lt.seqSupport[k]; !ok || vv != v {
			return false
		}
	}
	return true
}
