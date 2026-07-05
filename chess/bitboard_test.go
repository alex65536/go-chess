package chess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitboardIter(t *testing.T) {
	bb := BbEmpty.
		With2(FileA, Rank4).
		With2(FileE, Rank2).
		With2(FileF, Rank3)

	var res []Coord
	for !bb.IsEmpty() {
		res = append(res, bb.Next())
	}

	exp := []Coord{
		CoordFromParts(FileA, Rank4),
		CoordFromParts(FileF, Rank3),
		CoordFromParts(FileE, Rank2),
	}
	assert.Equal(t, exp, res)
}

func TestBitboardOps(t *testing.T) {
	ca := CoordFromParts(FileA, Rank4)
	cb := CoordFromParts(FileE, Rank2)
	cc := CoordFromParts(FileF, Rank3)

	bb1 := BbEmpty.With(ca).With(cb)
	bb2 := BbEmpty.With(cb).With(cc)
	assert.Equal(t, BbEmpty.With(cb), bb1&bb2)
	assert.Equal(t, BbEmpty.With(ca).With(cb).With(cc), bb1|bb2)
	assert.Equal(t, BbEmpty.With(ca).With(cc), bb1^bb2)

	assert.Equal(t, 62, (^bb1).Len())
	assert.Equal(t, 2, bb1.Len())
}

func TestBitboardFlip(t *testing.T) {
	bb1 := BbEmpty.
		With2(FileA, Rank1).
		With2(FileB, Rank1).
		With2(FileC, Rank1).
		With2(FileA, Rank2)
	bb2 := BbEmpty.
		With2(FileA, Rank8).
		With2(FileB, Rank8).
		With2(FileC, Rank8).
		With2(FileA, Rank7)
	bb3 := BbEmpty.
		With2(FileH, Rank1).
		With2(FileG, Rank1).
		With2(FileF, Rank1).
		With2(FileH, Rank2)
	bb4 := BbEmpty.
		With2(FileH, Rank8).
		With2(FileG, Rank8).
		With2(FileF, Rank8).
		With2(FileH, Rank7)

	assert.Equal(t, bb2, bb1.FlippedRank())
	assert.Equal(t, bb1, bb2.FlippedRank())
	assert.Equal(t, bb4, bb3.FlippedRank())
	assert.Equal(t, bb3, bb4.FlippedRank())

	assert.Equal(t, bb3, bb1.FlippedFile())
	assert.Equal(t, bb4, bb2.FlippedFile())
	assert.Equal(t, bb1, bb3.FlippedFile())
	assert.Equal(t, bb2, bb4.FlippedFile())
}

func TestBitboardFormat(t *testing.T) {
	bb := BbEmpty.
		With2(FileA, Rank4).
		With2(FileE, Rank2).
		With2(FileF, Rank3).
		With2(FileH, Rank8)
	assert.Equal(t, "00000001/00000000/00000000/00000000/10000000/00000100/00001000/00000000", bb.String())
}
