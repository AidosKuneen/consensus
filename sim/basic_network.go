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

/** Peer to peer network simulator.

The network is formed from a set of Peer objects representing
vertices and configurable connections representing edges.
The caller is responsible for creating the Peer objects ahead
of time.

Peer objects cannot be destroyed once the BasicNetwork is
constructed. To handle peers going online and offline,
callers can simply disconnect all links and reconnect them
later. Connections are directed, one end is the inbound
Peer and the other is the outbound Peer.

Peers may send messages along their connections. To simulate
the effects of latency, these messages can be delayed by a
configurable duration set when the link is established.
Messages always arrive in the order they were sent on a
particular connection.

A message is modeled using a lambda function. The caller
provides the code to execute upon delivery of the message.
If a Peer is disconnected, all messages pending delivery
at either end of the connection will not be delivered.

When creating the Peer set, the caller needs to provide a
Scheduler object for managing the the timing and delivery
of messages. After constructing the network, and establishing
connections, the caller uses the scheduler's step_* functions
to drive messages through the network.

The graph of peers and connections is internally represented
using Digraph<Peer,BasicNetwork::link_type>. Clients have
const access to that graph to perform additional operations not
directly provided by BasicNetwork.

Peer Requirements:

	Peer should be a lightweight type, cheap to copy
	and/or move. A good candidate is a simple pointer to
	the underlying user defined type in the simulation.

	Expression      Type        Requirements
	----------      ----        ------------
	P               Peer
	u, v                        Values of type P
	P u(v)                      CopyConstructible
	u.~P()                      Destructible
	u == v          bool        EqualityComparable
	u < v           bool        LessThanComparable
	std::hash<P>    class       std::hash is defined for P
	! u             bool        true if u is not-a-peer

*/

type basicNetwork struct {
	sch   *scheduler
	links *digraph
}

func (n *basicNetwork) link(from *peer) []*edge {
	return n.links.outEdges(from)
}
func (n *basicNetwork) connect(from, to *peer, delay time.Duration) bool {
	if to == from {
		return false
	}
	now := n.sch.clock.Now()
	if !n.links.connect(to, from, &linkType{
		inbound:     true,
		delay:       delay,
		established: now,
	}) {
		return false
	}
	result := n.links.connect(from, to, &linkType{
		inbound:     true,
		delay:       delay,
		established: now,
	})
	if !result {
		panic("should be true")
	}
	return true
}

func (n *basicNetwork) disconnect(peer1, peer2 *peer) bool {
	if !n.links.disconnect(peer1, peer2) {
		return false
	}
	r := n.links.disconnect(peer2, peer1)
	if !r {
		panic("should be true")
	}
	return true
}
func (n *basicNetwork) send(from, to *peer, f func()) {
	links := n.links.edge(from, to)
	if links == nil {
		return
	}
	sent := n.sch.clock.Now()
	n.sch.in(links.delay, func() {
		if links != nil && (links.established.Before(sent) || links.established.Equal(sent)) {
			f()
		}
	})
}
