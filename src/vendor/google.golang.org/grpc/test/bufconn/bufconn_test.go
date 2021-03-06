/*
 *
 * Copyright 2017 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package bufconn

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"testing"
	"time"
)

func testRW(r io.Reader, w io.Writer) error {
	for i := 0; i < 20; i++ {
		d := make([]byte, i)
		for j := 0; j < i; j++ {
			d[j] = byte(i - j)
		}
		var rn int
		var rerr error
		b := make([]byte, i)
		done := make(chan struct{})
		go func() {
			for rn < len(b) && rerr == nil {
				var x int
				x, rerr = r.Read(b[rn:])
				rn += x
			}
			close(done)
		}()
		wn, werr := w.Write(d)
		if wn != i || werr != nil {
			return fmt.Errorf("%v: w.Write(%v) = %v, %v; want %v, nil", i, d, wn, werr, i)
		}
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			return fmt.Errorf("%v: r.Read never returned", i)
		}
		if rn != i || rerr != nil {
			return fmt.Errorf("%v: r.Read = %v, %v; want %v, nil", i, rn, rerr, i)
		}
		if !reflect.DeepEqual(b, d) {
			return fmt.Errorf("%v: r.Read read %v; want %v", i, b, d)
		}
	}
	return nil
}

func TestPipe(t *testing.T) {
	p := newPipe(10)
	if err := testRW(p, p); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestPipeClose(t *testing.T) {
	p := newPipe(10)
	p.Close()
	if _, err := p.Write(nil); err != io.ErrClosedPipe {
		t.Fatalf("p.Write = _, %v; want _, %v", err, io.ErrClosedPipe)
	}
	if _, err := p.Read(nil); err != io.ErrClosedPipe {
		t.Fatalf("p.Read = _, %v; want _, %v", err, io.ErrClosedPipe)
	}
}

func TestConn(t *testing.T) {
	p1, p2 := newPipe(10), newPipe(10)
	c1, c2 := &conn{p1, p2}, &conn{p2, p1}

	if err := testRW(c1, c2); err != nil {
		t.Fatalf(err.Error())
	}
	if err := testRW(c2, c1); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestListener(t *testing.T) {
	l := Listen(7)
	var s net.Conn
	var serr error
	done := make(chan struct{})
	go func() {
		s, serr = l.Accept()
		close(done)
	}()
	c, cerr := l.Dial()
	<-done
	if cerr != nil || serr != nil {
		t.Fatalf("cerr = %v, serr = %v; want nil, nil", cerr, serr)
	}
	if err := testRW(c, s); err != nil {
		t.Fatalf(err.Error())
	}
	if err := testRW(s, c); err != nil {
		t.Fatalf(err.Error())
	}
}
