package chess

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUCIMoveFromString(t *testing.T) {
	for _, v := range []struct {
		src string
		res UCIMove
	}{
		{
			src: "0000",
			res: UCIMove{Kind: UCIMoveNull},
		},
		{
			src: "e2e4",
			res: UCIMove{
				Kind: UCIMoveSimple,
				Src:  CoordFromParts(FileE, Rank2),
				Dst:  CoordFromParts(FileE, Rank4),
			},
		},
		{
			src: "g7g8q",
			res: UCIMove{
				Kind:    UCIMovePromote,
				Src:     CoordFromParts(FileG, Rank7),
				Dst:     CoordFromParts(FileG, Rank8),
				Promote: PieceQueen,
			},
		},
	} {
		m, err := UCIMoveFromString(v.src)
		require.NoError(t, err)
		assert.Equal(t, v.res, m)
		assert.Equal(t, v.src, m.String())
	}
}

func TestUCIMoveToMove(t *testing.T) {
	for _, v := range []struct {
		src     UCIMove
		fen     string
		res     Move
		illegal bool
	}{
		{
			src:     UCIMove{Kind: UCIMoveNull},
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			res:     NullMove(),
			illegal: true,
		},
		{
			src: UCIMove{
				Kind: UCIMoveSimple,
				Src:  CoordFromParts(FileE, Rank2),
				Dst:  CoordFromParts(FileE, Rank4),
			},
			fen: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			res: NewMoveUnchecked(
				MovePawnDouble,
				CellFromParts(ColorWhite, PiecePawn),
				CoordFromParts(FileE, Rank2),
				CoordFromParts(FileE, Rank4),
			),
		},
		{
			src: UCIMove{
				Kind:    UCIMovePromote,
				Src:     CoordFromParts(FileG, Rank7),
				Dst:     CoordFromParts(FileG, Rank8),
				Promote: PieceQueen,
			},
			fen: "4K3/6P1/8/8/8/k7/8/8 w - - 0 1",
			res: NewMoveUnchecked(
				MovePromoteQueen,
				CellFromParts(ColorWhite, PiecePawn),
				CoordFromParts(FileG, Rank7),
				CoordFromParts(FileG, Rank8),
			),
		},
	} {
		b, err := BoardFromFEN(v.fen)
		require.NoError(t, err)

		mv, err := MoveFromUCIMove(v.src, b)
		require.NoError(t, err)
		assert.Equal(t, v.res, mv)

		mv, err = SemilegalMoveFromUCIMove(v.src, b)
		if v.illegal {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, v.res, mv)
		}

		mv, err = LegalMoveFromUCIMove(v.src, b)
		if v.illegal {
			assert.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, v.res, mv)
		}

		assert.Equal(t, v.src.String(), mv.UCI())
	}
}
