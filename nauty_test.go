// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nauty

import (
	"testing"
)

func Test_AgainstTriple(t *testing.T) {
	m := make(map[Triple]Triple)
	for i := byte(0); i < 64; i++ {
		t := Triple{i}
		tc := t.ToCanonical()
		gc := FromDenseGraph(t.ToDenseGraph().ToCanonical())
		if val, ok := m[tc]; ok {
			if val != gc {
				p(t, tc, gc, val)
			}
		} else {
			m[tc] = gc
		}
	}
	for k, v := range m {
		if k != v {
			p(k, v)
		}
	}
}


