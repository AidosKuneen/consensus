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
	"sort"
	"time"
)

/** Simulated discrete-event scheduler.

  Simulates the behavior of events using a single common clock.

  An event is modeled using a lambda function and is scheduled to occur at a
  specific time. Events may be canceled using a token returned when the
  event is scheduled.

  The caller uses one or more of the step, step_one, step_for, step_until and
  step_while functions to process scheduled events.
*/

type byWhen struct {
	when time.Time
	h    func()
}

type scheduler struct {
	que   []*byWhen
	clock *mclock
}

func (s *scheduler) at(w time.Time, h func()) *byWhen {
	b := &byWhen{
		when: w,
		h:    h,
	}
	s.que = append(s.que, b)
	sort.Slice(s.que, func(i, j int) bool {
		return s.que[i].when.Before(s.que[j].when)
	})
	return b
}
func (s *scheduler) cancel(b *byWhen) {
	for i, bb := range s.que {
		if bb != b {
			continue
		}
		copy(s.que[i:], s.que[i+1:])
		s.que[len(s.que)-1] = nil
		s.que = s.que[:len(s.que)-1]
		return
	}
	panic("not found")
}

func (s *scheduler) in(delay time.Duration, h func()) *byWhen {
	return s.at(s.clock.Now().Add(delay), h)
}

func (s *scheduler) stepOne() bool {
	if len(s.que) == 0 {
		return false
	}
	q := s.que[0]
	s.cancel(s.que[0])
	s.clock.now = q.when
	// log.Println("run a step at", s.clock.now)
	q.h()
	return true
}

func (s *scheduler) step() bool {
	if !s.stepOne() {
		return false
	}
	for {
		if !s.stepOne() {
			break
		}
	}
	return true
}

func (s *scheduler) stepUntil(until time.Time) bool {
	for len(s.que) == 0 {
		s.clock.now = until
		return false
	}
	for len(s.que) > 0 {
		if s.que[0].when.After(until) {
			s.clock.now = until
			return true
		}
		s.stepOne()
	}
	return true
}

func (s *scheduler) stepFor(amount time.Duration) bool {
	return s.stepUntil(s.clock.Now().Add(amount))
}
