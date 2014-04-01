// Copyright 2014 Garrett D'Amore
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sp

import (
	"sync"
)

// xreq is an implementation of the XREQ protocol.
type xreq struct {
	sock    ProtocolSocket
	rcvmsg  *Message
	sndmsg  *Message
	rcvlock sync.Mutex
	sndlock sync.Mutex
}

func (p *xreq) Init(socket ProtocolSocket) {
	p.sock = socket
}

func (x *xreq) Process() {
	x.ProcessRecv()
	x.ProcessSend()
}

func (x *xreq) ProcessRecv() {
	var msg *Message
	x.rcvlock.Lock()
	defer x.rcvlock.Unlock()
	for {
		msg = x.rcvmsg
		if msg == nil {
			if msg, _, _ = x.sock.RecvAnyPipe(); msg != nil {
				// Move the requestID into the header
				if msg.trimUint32() != nil {
					// XXX: FreeMsg() - error
					continue
				}
			}
		}
		if msg == nil {
			return
		}
		if !x.sock.PushUp(msg) {
			x.rcvmsg = msg
			return
		}
		x.rcvmsg = nil
	}
}

func (x *xreq) ProcessSend() {
	for {
		msg := x.sndmsg
		x.sndmsg = nil
		if msg == nil {
			msg = x.sock.PullDown()
		}
		if msg == nil {
			return
		}
		// Send sends unmolested.  If we can't due to lack of a
		// connected peer, we drop it.  (Req protocol resends, but
		// we don't in xreq.)  Note that it is expected that the
		// application will have written the request ID into the
		// header at minimum, but possibly a full backtrace.  We
		// don't bother to check.  (XXX: Perhaps we should, and
		// drop any message that lacks at least a minimal header?)
		//
		if _, err := x.sock.SendAnyPipe(msg); err != nil {
			x.sndmsg = msg
			if err == ErrPipeFull {
				// No available pipes, come back later
				return
			}
			// Other errors, (ErrClosed) look for another possible pipe
			continue
		}
	}
}

func (*xreq) Name() string {
	return XReqName
}

func (*xreq) Number() uint16 {
	return ProtoReq
}

func (*xreq) IsRaw() bool {
	return true
}

func (*xreq) ValidPeer(peer uint16) bool {
	if peer == ProtoRep {
		return true
	}
	return false
}

func (*xreq) AddEndpoint(Endpoint) {}
func (*xreq) RemEndpoint(Endpoint) {}

type xreqFactory int

func (xreqFactory) NewProtocol() Protocol {
	return new(xreq)
}

// XReqFactory implements the Protocol Factory for the XREQ protocol.
// The XREQ Protocol is the raw form of the REQ (Request) protocol.
var XReqFactory xreqFactory
