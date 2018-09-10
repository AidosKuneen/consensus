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

// This is a rewrite of https://github.com/ripple/rippled/src/test/csf
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

package sim

type trustGraph struct {
	graph digraph
}

func (t *trustGraph) trust(from, to *peer) {
	t.graph.connect2(from, to)
}
func (t *trustGraph) untrust(from, to *peer) {
	t.graph.disconnect(from, to)
}
func (t *trustGraph) trusts(from, to *peer) {
	t.graph.connected(from, to)
}
func (t *trustGraph) trustedPeers(a *peer) []*peer {
	return t.graph.outVerticesw(a)
}

type forkInfo struct {
	unlA     map[*peer]struct{}
	unlB     map[*peer]struct{}
	overlap  int
	required float64
}

func (t *trustGraph) forkablePairs(quorum float64) []*forkInfo {
	// Check the forking condition by looking at intersection
	// of UNL between all pairs of nodes.

	// TODO: Use the improved bound instead of the whitepaper bound.

	unique := make([]map[*peer]struct{}, len(t.graph.outVertices()))
	for i, p1 := range t.graph.outVertices() {
		unique[i] = make(map[*peer]struct{})
		for _, p2 := range t.trustedPeers(p1) {
			unique[i][p2] = struct{}{}
		}
	}
	var res []*forkInfo
	for i := 0; i < len(unique); i++ {
		for j := i + 1; j < len(unique); j++ {
			unlA := unique[i]
			unlB := unique[j]
			s := len(unlA)
			if s < len(unlB) {
				s = len(unlB)
			}
			rhs := 2.0 * (1 - quorum) * float64(s)
			intersec := 0
			for p := range unlA {
				if _, ok := unlB[p]; ok {
					intersec++
				}
			}
			if float64(intersec) < rhs {
				res = append(res, &forkInfo{
					unlA:     unlA,
					unlB:     unlB,
					overlap:  intersec,
					required: rhs,
				})
			}
		}
	}
	return res
}
func (t *trustGraph) canfork(quorum float64) bool {
	return len(t.forkablePairs(quorum)) > 0
}
