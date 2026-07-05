package chess

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	for f := range FileMax {
		assert.True(t, f.IsValid())
		f2, err := FileFromByte(f.ToByte())
		require.NoError(t, err)
		assert.Equal(t, f, f2)
	}
}

func TestRank(t *testing.T) {
	for r := range RankMax {
		assert.True(t, r.IsValid())
		r2, err := RankFromByte(r.ToByte())
		require.NoError(t, err)
		assert.Equal(t, r, r2)
	}
}

func TestPiece(t *testing.T) {
	for p := range PieceMax {
		assert.True(t, p.IsValid())
		p2, err := PieceFromString(p.String())
		require.NoError(t, err)
		assert.Equal(t, p, p2)
	}
}

func TestCoord(t *testing.T) {
	var used [CoordMax]bool
	for f := range FileMax {
		for r := range RankMax {
			c := CoordFromParts(f, r)
			assert.False(t, used[c])
			used[c] = true
			assert.Equal(t, f, c.File())
			assert.Equal(t, r, c.Rank())
			c2, err := CoordFromString(c.String())
			require.NoError(t, err)
			assert.Equal(t, c, c2)
		}
	}
	for c := range CoordMax {
		assert.True(t, used[c])
	}
}

func TestCoordFlip(t *testing.T) {
	c2 := CoordFromParts(FileC, Rank2)
	c7 := CoordFromParts(FileC, Rank7)
	f2 := CoordFromParts(FileF, Rank2)
	f7 := CoordFromParts(FileF, Rank7)

	assert.Equal(t, c7, c2.FlippedRank())
	assert.Equal(t, c2, c7.FlippedRank())
	assert.Equal(t, f7, f2.FlippedRank())
	assert.Equal(t, f2, f7.FlippedRank())

	assert.Equal(t, f2, c2.FlippedFile())
	assert.Equal(t, f7, c7.FlippedFile())
	assert.Equal(t, c2, f2.FlippedFile())
	assert.Equal(t, c7, f7.FlippedFile())
}

func TestCell(t *testing.T) {
	var used [CellMax]bool

	assert.True(t, CellEmpty.IsValid())
	used[CellEmpty] = true

	assert.False(t, CellEmpty.HasColor(ColorWhite))
	assert.False(t, CellEmpty.HasColor(ColorBlack))
	_, ok := CellEmpty.Color()
	assert.False(t, ok)

	_, ok = CellEmpty.Piece()
	assert.False(t, ok)

	for co := range ColorMax {
		for p := range PieceMax {
			c := CellFromParts(co, p)
			assert.True(t, c.IsValid())
			assert.False(t, used[c])
			used[c] = true

			co2, ok := c.Color()
			assert.True(t, ok)
			assert.Equal(t, co2, co)
			assert.True(t, c.HasColor(co2))
			assert.False(t, c.HasColor(co2.Inv()))

			p2, ok := c.Piece()
			assert.True(t, ok)
			assert.Equal(t, p2, p)

			c2, err := CellFromString(c.String())
			require.NoError(t, err)
			assert.Equal(t, c, c2)
		}
	}

	for c := range CellMax {
		assert.True(t, used[c])
	}
}

func TestCastling(t *testing.T) {
	for r := range CastlingRightsMax {
		assert.True(t, r.IsValid())

		for c := range ColorMax {
			assert.Equal(t, r.Has(c, CastlingKingside) || r.Has(c, CastlingQueenside), r.HasColor(c))
		}

		r2, err := CastlingRightsFromString(r.String())
		require.NoError(t, err)
		assert.Equal(t, r2, r)
	}
}

func TestOutcome(t *testing.T) {
	o1 := NewOutcome(VerdictCheckmate, ColorWhite)
	o2 := NewOutcome(VerdictCheckmate, ColorBlack)
	o3 := NewOutcome(VerdictStalemate, ColorWhite)
	o4 := NewOutcome(VerdictStalemate, ColorBlack)
	o5 := RunningOutcome()
	assert.NotEqual(t, o1, o2)
	assert.NotEqual(t, o2, o3)
	assert.Equal(t, o3, o4)
	assert.NotEqual(t, o4, o5)

	assert.Equal(t, VerdictCheckmate, o1.Verdict())
	assert.Equal(t, VerdictStalemate, o3.Verdict())
	assert.Equal(t, VerdictRunning, o5.Verdict())

	s, ok := o1.Side()
	assert.True(t, ok)
	assert.Equal(t, ColorWhite, s)
	_, ok = o3.Side()
	assert.False(t, ok)
	_, ok = o5.Side()
	assert.False(t, ok)

	assert.Equal(t, StatusWhiteWins, o1.Status())
	assert.Equal(t, StatusBlackWins, o2.Status())
	assert.Equal(t, StatusDraw, o3.Status())
	assert.Equal(t, StatusDraw, o4.Status())
	assert.Equal(t, StatusRunning, o5.Status())

	w, ok := o1.Status().Winner()
	assert.True(t, ok)
	assert.Equal(t, ColorWhite, w)

	w, ok = o2.Status().Winner()
	assert.True(t, ok)
	assert.Equal(t, ColorBlack, w)

	_, ok = o3.Status().Winner()
	assert.False(t, ok)

	_, ok = o4.Status().Winner()
	assert.False(t, ok)

	_, ok = o5.Status().Winner()
	assert.False(t, ok)
}
