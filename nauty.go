// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nauty

/*
#include <nauty.h>
#cgo LDFLAGS: -lnauty

void setDefault(optionblk* o) {
	o->dispatch = &dispatch_graph;
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"unsafe"
)

const WordSize = int(8 * unsafe.Sizeof(uintptr(0)))
const MSB = uintptr(1) << uint(WordSize-1)

func init() {
	if C.WORDSIZE != WordSize {
		panic(fmt.Errorf("size of uintptr %d is not equal to C WORDSIZE %d.", WordSize, C.WORDSIZE))
	}
}

type DenseGraph struct {
	N   int
	D   []uintptr
	Lab []int
	Ptn []int
}

func NewDenseGraph(n int) *DenseGraph {
	g := &DenseGraph{N: n, Lab: make([]int, n), Ptn: make([]int, n)}
	g.D = make([]uintptr, g.M()*n)
	return g
}

func (g *DenseGraph) M() int {
	return (g.N-1)/WordSize + 1
}

func (g *DenseGraph) checkRange(v int) {
	if v >= g.N {
		panic(fmt.Errorf("Vertex %d out of range %d.", v, g.N))
	}
}

func (g *DenseGraph) AddEdge(v, w int) {
	g.checkRange(v)
	g.checkRange(w)
	g.D[g.M()*v+w/WordSize] |= MSB >> uint(w%WordSize)
}

func (g *DenseGraph) HasEdge(v, w int) bool {
	g.checkRange(v)
	g.checkRange(w)
	return 0 != (g.D[g.M()*v+w/WordSize] & (MSB >> uint(w%WordSize)))
}

func (g *DenseGraph) ToCanonical() *DenseGraph {
	_, _, cg := Nauty(
		g,
		&Option{
			Getcanon:      true,
			Digraph:       true,
			Defaultptn:    true,
			TcLevel:       100,
			Maxinvarlevel: 1,
		},
	)
	return cg
}

func (g *DenseGraph) graph() *C.graph {
	return (*C.graph)(unsafe.Pointer(&g.D[0]))
}

func (g *DenseGraph) AdjacentVertices(v int) (ws []int) {
	for w := range g.D {
		if g.HasEdge(v, w) {
			ws = append(ws, w)
		}
	}
	return
}

func (g *DenseGraph) String() string {
	var b bytes.Buffer
	for v := range g.D {
		b.WriteString(fmt.Sprintf("%d -> { ", v))
		for _, w := range g.AdjacentVertices(v) {
			b.WriteString(fmt.Sprintf("%d ", w))
		}
		b.WriteString("}\n")
	}
	return b.String()
}

type Option struct {
	Getcanon     bool // make canong and canonlab?
	Digraph      bool // multiple edges or loops?
	Writeautoms  bool // write automorphisms?
	Writemarkers bool // write stats on pts fixed, etc.?
	Defaultptn   bool // set lab,ptn,active for single cell?
	Cartesian    bool // use cartesian rep for writing automs?
	Linelength   int  // max chars/line (excl. '\n') for output
	/*
		Outfile      string // file for output, if any
		void (*userrefproc)       // replacement for usual refine procedure 
		     (graph*,int*,int*,int,int*,permutation*,set*,int*,int,int);
		void (*userautomproc)     // procedure called for each automorphism 
		     (int,permutation*,int*,int,int,int);
		void (*userlevelproc)     // procedure called for each level 
		     (int*,int*,int,int*,statsblk*,int,int,int,int,int,int);
		void (*usernodeproc)      // procedure called for each node 
		     (graph*,int*,int*,int,int,int,int,int,int);
		void (*invarproc)         // procedure to compute vertex-invariant 
		     (graph*,int*,int*,int,int,int,permutation*,int,boolean,int,int);
	*/
	TcLevel       int // max level for smart target cell choosing
	Mininvarlevel int // min level for invariant computation
	Maxinvarlevel int // max level for invariant computation
	Invararg      int // value passed to (*invarproc)()
	/*
	   dispatchvec *dispatch;    // vector of object-specific routines
	   void *extra_options;      // arbitrary extra options
	*/
}

func (o *Option) optionblk() *C.optionblk {
	b := &C.optionblk{
		getcanon:     C.int(boolean(o.Getcanon)),
		digraph:      boolean(o.Digraph),
		writeautoms:  boolean(o.Writeautoms),
		writemarkers: boolean(o.Writemarkers),
		defaultptn:   boolean(o.Defaultptn),
		cartesian:    boolean(o.Cartesian),
		linelength:   C.int(o.Linelength),

		tc_level:      C.int(o.TcLevel),
		mininvarlevel: C.int(o.Mininvarlevel),
		maxinvarlevel: C.int(o.Maxinvarlevel),
		invararg:      C.int(o.Invararg),
	}
	C.setDefault(b)
	return b
}

func boolean(b bool) C.boolean {
	if b {
		return 1
	}
	return 0
}

type Stats struct {
	Grpsize1      float64 // size of group is
	Grpsize2      int     //    grpsize1 * 10^grpsize2
	Numorbits     int     // number of orbits in group
	Numgenerators int     // number of generators found
	Errstatus     int     // if non-zero : an error code
	Numnodes      int     // total number of nodes
	Numbadleaves  int     // number of leaves of no use
	Maxlevel      int     // maximum depth of search
	Tctotal       int     // total size of all target cells
	Canupdates    int     // number of updates of best label
	Invapplics    int     // number of applications of invarproc
	Invsuccesses  int     // number of successful uses of invarproc()
	Invarsuclevel int     // least level where invarproc worked
}

func Nauty(
	g *DenseGraph,
	option *Option,
) (
	orbits []int,
	stats *Stats,
	canong *DenseGraph,
) {
	lab_ := make([]C.int, len(g.Lab))
	for i := range lab_ {
		lab_[i] = C.int(g.Lab[i])
	}
	ptn_ := make([]C.int, len(g.Ptn))
	for i := range ptn_ {
		ptn_[i] = C.int(g.Ptn[i])
	}
	var active *C.set = nil
	orbits, orbits_ := make([]int, g.N), make([]C.int, g.N)
	var stats_ C.statsblk
	worksize_ := 100 * g.M()
	workspace_ := make([]C.set, worksize_)
	canong = NewDenseGraph(g.N)
	C.nauty(g.graph(), &lab_[0], &ptn_[0], active, &orbits_[0], option.optionblk(), &stats_, &workspace_[0], C.int(worksize_), C.int(g.M()), C.int(g.N), canong.graph())
	for i := range orbits {
		orbits[i] = int(orbits_[i])
	}
	stats = &Stats{
		Grpsize1:      float64(stats_.grpsize1),
		Grpsize2:      int(stats_.grpsize2),
		Numorbits:     int(stats_.numorbits),
		Numgenerators: int(stats_.numgenerators),
		Errstatus:     int(stats_.errstatus),
		Numnodes:      int(stats_.numnodes),
		Numbadleaves:  int(stats_.numbadleaves),
		Maxlevel:      int(stats_.maxlevel),
		Tctotal:       int(stats_.tctotal),
		Canupdates:    int(stats_.canupdates),
		Invapplics:    int(stats_.invapplics),
		Invsuccesses:  int(stats_.invsuccesses),
		Invarsuclevel: int(stats_.invarsuclevel),
	}
	return
}

