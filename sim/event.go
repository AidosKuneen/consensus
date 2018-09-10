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

import "github.com/AidosKuneen/consensus"

// Events are emitted by peers at a variety of points during the simulation.
// Each event is emitted by a particlar peer at a particular time. Collectors
// process these events, perhaps calculating statistics or storing events to
// a log for post-processing.
//
// The Event types can be arbitrary, but should be copyable and lightweight.
//
// Example collectors can be found in collectors.h, but have the general
// interface:
//
// @code
//     template <class T>
//     struct Collector
//     {
//        template <class Event>
//        void
//        on(peerID who, SimTime when, Event e);
//     };
// @endcode
//
// CollectorRef.f defines a type-erased holder for arbitrary Collectors.  If
// any new events are added, the interface there needs to be updated.

/** A value to be flooded to all other peers starting from this peer.
 */
type share struct {
	//! Event that is shared
	V interface{}
}

/** A value relayed to another peer as part of flooding
 */
type relay struct {
	//! Peer relaying to
	to consensus.NodeID

	//! The value to relay
	V interface{}
}

/** A value received from another peer as part of flooding
 */
type receive struct {
	//! Peer that sent the value
	from consensus.NodeID

	//! The received value
	V interface{}
}

/** A transaction submitted to a peer */
type submitTx struct {
	//! The submitted transaction
	tx *tx
}

/** Peer starts a new consensus round
 */
type startRound struct {
	//! The preferred ledger for the start of consensus
	bestLedger consensus.LedgerID

	//! The prior ledger on hand
	prevLedger consensus.Ledger
}

/** Peer closed the open ledger
 */
type closeLedger struct {
	// The ledger closed on
	prevLedger consensus.Ledger

	// Initial txs for including in ledger
	txSetType txSetType
}

//! Peer accepted consensus results
type acceptLedger struct {
	// The newly created ledger
	ledger consensus.Ledger

	// The prior ledger (this is a jump if prior.id() != ledger.parentID())
	prior consensus.Ledger
}

//! Peer detected a wrong prior ledger during consensus
type wrongPrevLedger struct {
	// ID of wrong ledger we had
	wrong consensus.LedgerID
	// ID of what we think is the correct ledger
	right consensus.LedgerID
}

//! Peer fully validated a new ledger
type fullyValidateLedger struct {
	//! The new fully validated ledger
	ledger consensus.Ledger

	//! The prior fully validated ledger
	//! This is a jump if prior.id() != ledger.parentID()
	prior consensus.Ledger
}
