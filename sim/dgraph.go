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

import (
	"time"
)

type linkType struct {
	inbound     bool
	delay       time.Duration
	established time.Time
}

type linksT map[*peer]*linkType
type graphT map[*peer]linksT

type digraph struct {
	graph graphT
}

func (d *digraph) connect(s, target *peer, e *linkType) bool {
	g, ok1 := d.graph[s]
	if !ok1 {
		g = make(linksT)
		d.graph[s] = g
	}
	_, ok2 := g[target]
	if !ok2 {
		g[target] = e
	}
	return !ok1 || !ok2 //true if not exists
}

func (d *digraph) connect2(s, v *peer) bool {
	return d.connect(s, v, &linkType{})
}
func (d *digraph) disconnect(s, v *peer) bool {
	g, ok := d.graph[s]
	if ok {
		_, ok := d.graph[s][v]
		delete(g, v)
		return ok
	}
	return false
}
func (d *digraph) edge(s, t *peer) *linkType {
	g, ok := d.graph[s]
	if ok {
		e, ok := g[t]
		if ok {
			return e
		}
	}
	return nil
}

func (d *digraph) outVertices() []*peer {
	r := make([]*peer, 0, len(d.graph))
	for k := range d.graph {
		r = append(r, k)
	}
	return r
}
func (d *digraph) outVerticesw(s *peer) []*peer {
	r := make([]*peer, 0, len(d.graph))
	for k := range d.graph[s] {
		r = append(r, k)
	}
	return r
}

type edge struct {
	source *peer
	target *peer
	data   *linkType
}

func (d *digraph) outEdges(s *peer) []*edge {
	r := make([]*edge, 0, len(d.graph[s]))
	for k, v := range d.graph[s] {
		r = append(r, &edge{
			source: s,
			target: k,
			data:   v,
		})
	}
	return r
}
func (d *digraph) outDegree(s *peer) int {
	return len(d.graph[s])
}
