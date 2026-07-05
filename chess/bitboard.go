package chess

import (
	"fmt"
	"math"
	"math/bits"
)

type Bitboard uint64

const (
	BbEmpty Bitboard = 0
	BbFull  Bitboard = math.MaxUint64
)

func BitboardFromCoord(c Coord) Bitboard {
	return Bitboard(uint64(1) << c)
}

func (b Bitboard) With(c Coord) Bitboard {
	return b | Bitboard(uint64(1)<<c)
}

func (b Bitboard) With2(f File, r Rank) Bitboard {
	return b.With(CoordFromParts(f, r))
}

func (b Bitboard) Without(c Coord) Bitboard {
	return b & ^Bitboard(uint64(1)<<c)
}

func (b Bitboard) Without2(f File, r Rank) Bitboard {
	return b.Without(CoordFromParts(f, r))
}

func (b Bitboard) DepositBits(x uint64) Bitboard {
	res, msk := uint64(0), uint64(b)
	for msk != 0 {
		bit := msk & -msk
		if (x & 1) != 0 {
			res |= bit
		}
		msk ^= bit
		x >>= 1
	}
	return Bitboard(res)
}

func (b *Bitboard) Set(c Coord) {
	*b = b.With(c)
}

func (b *Bitboard) Unset(c Coord) {
	*b = b.Without(c)
}

func (b Bitboard) Has(c Coord) bool {
	return ((uint64(b) >> c) & 1) != 0
}

func (b Bitboard) Has2(f File, r Rank) bool {
	return b.Has(CoordFromParts(f, r))
}

func (b Bitboard) Len() int {
	return bits.OnesCount64(uint64(b))
}

func (b Bitboard) IsEmpty() bool {
	return b == BbEmpty
}

func (b Bitboard) FlippedRank() Bitboard {
	return Bitboard(bits.ReverseBytes64(uint64(b)))
}

func (b Bitboard) FlippedFile() Bitboard {
	return Bitboard(
		bits.ReverseBytes64(
			bits.Reverse64(uint64(b))))
}

func (b Bitboard) String() string {
	v := bits.Reverse64(uint64(b))
	return fmt.Sprintf(
		"%08b/%08b/%08b/%08b/%08b/%08b/%08b/%08b",
		(v>>56)&0xff,
		(v>>48)&0xff,
		(v>>40)&0xff,
		(v>>32)&0xff,
		(v>>24)&0xff,
		(v>>16)&0xff,
		(v>>8)&0xff,
		v&0xff,
	)
}

func (b Bitboard) GetFirst() Coord {
	return Coord(bits.TrailingZeros64(uint64(b)))
}

func (b *Bitboard) Next() Coord {
	res := b.GetFirst()
	*b &= *b - 1
	return res
}
