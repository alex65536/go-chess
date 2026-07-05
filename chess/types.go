package chess

import (
	"fmt"
	"strings"
)

type File uint8

const (
	FileA File = iota
	FileB
	FileC
	FileD
	FileE
	FileF
	FileG
	FileH
	FileMax
)

func FileFromByte(b byte) (File, error) {
	if 'a' <= b && b <= 'h' {
		return File(b - 'a'), nil
	}
	return File(0), fmt.Errorf("unexpected file char %q", b)
}

func (f File) IsValid() bool {
	return f < FileMax
}

func (f File) ToByte() byte {
	return 'a' + byte(f)
}

func (f File) String() string {
	var b strings.Builder
	_ = b.WriteByte(f.ToByte())
	return b.String()
}

type Rank uint8

const (
	Rank8 Rank = iota
	Rank7
	Rank6
	Rank5
	Rank4
	Rank3
	Rank2
	Rank1
	RankMax
)

func RankFromByte(b byte) (Rank, error) {
	if '1' <= b && b <= '8' {
		return Rank('8' - b), nil
	}
	return Rank(0), fmt.Errorf("unexpected rank char %q", b)
}

func (r Rank) IsValid() bool {
	return r < RankMax
}

func (r Rank) ToByte() byte {
	return '8' - byte(r)
}

func (r Rank) String() string {
	var b strings.Builder
	_ = b.WriteByte(r.ToByte())
	return b.String()
}

type Coord uint8

const CoordMax Coord = 64

type CoordDelta struct {
	File int8
	Rank int8
}

func CoordFromParts(f File, r Rank) Coord {
	return Coord((uint8(r) << 3) | uint8(f))
}

func (c Coord) IsValid() bool {
	return c < CoordMax
}

func (c Coord) File() File {
	return File(c & 7)
}

func (c Coord) Rank() Rank {
	return Rank(c >> 3)
}

func (c Coord) Diag() int {
	return int(c.File()) + int(c.Rank())
}

func (c Coord) Antidiag() int {
	return 7 - int(c.Rank()) + int(c.File())
}

func (c Coord) Add(delta int8) Coord {
	return Coord(uint8(c) + uint8(delta))
}

func (c Coord) Shift(d CoordDelta) MaybeCoord {
	newFile, newRank := uint8(c.File())+uint8(d.File), uint8(c.Rank())+uint8(d.Rank)
	if newFile >= 8 || newRank >= 8 {
		return NoCoord
	}
	return SomeCoord(CoordFromParts(File(newFile), Rank(newRank)))
}

func (c Coord) FlippedRank() Coord {
	return Coord(c ^ 56)
}

func (c Coord) FlippedFile() Coord {
	return Coord(c ^ 7)
}

func (c Coord) String() string {
	var b strings.Builder
	_ = b.WriteByte(c.File().ToByte())
	_ = b.WriteByte(c.Rank().ToByte())
	return b.String()
}

func CoordFromString(s string) (Coord, error) {
	if len(s) != 2 {
		return Coord(0), fmt.Errorf("invalid string length")
	}
	f, err := FileFromByte(s[0])
	if err != nil {
		return Coord(0), err
	}
	r, err := RankFromByte(s[1])
	if err != nil {
		return Coord(0), err
	}
	return CoordFromParts(f, r), nil
}

type MaybeCoord uint8

const NoCoord MaybeCoord = 255

func SomeCoord(c Coord) MaybeCoord {
	return MaybeCoord(c)
}

func (c MaybeCoord) IsValid() bool {
	return c == NoCoord || Coord(c).IsValid()
}

func (c MaybeCoord) IsSome() bool {
	return c != NoCoord
}

func (c MaybeCoord) IsNone() bool {
	return c == NoCoord
}

func (c MaybeCoord) ForceGet() Coord {
	return Coord(c)
}

func (c MaybeCoord) TryGet() (Coord, bool) {
	if c == NoCoord {
		return Coord(0), false
	}
	return Coord(c), true
}

func (c MaybeCoord) String() string {
	if c == NoCoord {
		return "-"
	}
	return Coord(c).String()
}

func MaybeCoordFromString(s string) (MaybeCoord, error) {
	if s == "-" {
		return NoCoord, nil
	}
	c, err := CoordFromString(s)
	if err != nil {
		return MaybeCoord(0), err
	}
	return SomeCoord(c), nil
}

type Color uint8

const (
	ColorWhite Color = iota
	ColorBlack
	ColorMax
)

func (c Color) Inv() Color {
	return Color(1 - uint8(c))
}

func (c Color) ToByte() byte {
	switch c {
	case ColorWhite:
		return 'w'
	case ColorBlack:
		return 'b'
	default:
		return '?'
	}
}

func ColorFromByte(b byte) (Color, error) {
	switch b {
	case 'w':
		return ColorWhite, nil
	case 'b':
		return ColorBlack, nil
	default:
		return Color(0), fmt.Errorf("unexpected color char %q", b)
	}
}

func (c Color) IsValid() bool {
	return c < ColorMax
}

func (c Color) String() string {
	var b strings.Builder
	_ = b.WriteByte(c.ToByte())
	return b.String()
}

func (c Color) LongString() string {
	switch c {
	case ColorWhite:
		return "white"
	case ColorBlack:
		return "black"
	default:
		return "invalid"
	}
}

func ColorFromString(s string) (Color, error) {
	if len(s) != 1 {
		return Color(0), fmt.Errorf("invalid string length")
	}
	return ColorFromByte(s[0])
}

type Piece uint8

const (
	PiecePawn Piece = iota
	PieceKing
	PieceKnight
	PieceBishop
	PieceRook
	PieceQueen
	PieceMax
)

func (p Piece) IsValid() bool {
	return p < PieceMax
}

func (p Piece) ToByte() byte {
	const bytes = "pknbrq"
	return bytes[p]
}

func (p Piece) String() string {
	var b strings.Builder
	_ = b.WriteByte(p.ToByte())
	return b.String()
}

func doPieceFromByte(b byte) (Piece, bool) {
	switch b {
	case 'p':
		return PiecePawn, true
	case 'k':
		return PieceKing, true
	case 'n':
		return PieceKnight, true
	case 'b':
		return PieceBishop, true
	case 'r':
		return PieceRook, true
	case 'q':
		return PieceQueen, true
	default:
		return Piece(0), false
	}
}

func PieceFromByte(b byte) (Piece, error) {
	p, ok := doPieceFromByte(b)
	if !ok {
		return Piece(0), fmt.Errorf("unexpected piece char %q", b)
	}
	return p, nil
}

func PieceFromString(s string) (Piece, error) {
	if len(s) != 1 {
		return Piece(0), fmt.Errorf("invalid string length")
	}
	return PieceFromByte(s[0])
}

type Cell uint8

const (
	CellEmpty Cell = iota
	CellWhitePawn
	CellWhiteKing
	CellWhiteKnight
	CellWhiteBishop
	CellWhiteRook
	CellWhiteQueen
	CellBlackPawn
	CellBlackKing
	CellBlackKnight
	CellBlackBishop
	CellBlackRook
	CellBlackQueen
	CellMax
)

func (c Cell) IsValid() bool {
	return c < CellMax
}

func (c Cell) IsFree() bool {
	return c == CellEmpty
}

func (c Cell) IsOccupied() bool {
	return c != CellEmpty
}

func CellFromParts(c Color, p Piece) Cell {
	switch c {
	case ColorWhite:
		return Cell(1 + uint8(p))
	case ColorBlack:
		return Cell(7 + uint8(p))
	default:
		panic("bad color")
	}
}

func (c Cell) HasColor(co Color) bool {
	if c == 0 {
		return false
	} else if 1 <= c && c <= 6 {
		return co == ColorWhite
	} else {
		return co == ColorBlack
	}
}

func (c Cell) Color() (Color, bool) {
	if c == 0 {
		return Color(0), false
	} else if 1 <= c && c <= 6 {
		return ColorWhite, true
	} else {
		return ColorBlack, true
	}
}

func (c Cell) Piece() (Piece, bool) {
	switch c {
	case CellEmpty:
		return Piece(0), false
	case CellWhitePawn, CellBlackPawn:
		return PiecePawn, true
	case CellWhiteKing, CellBlackKing:
		return PieceKing, true
	case CellWhiteKnight, CellBlackKnight:
		return PieceKnight, true
	case CellWhiteBishop, CellBlackBishop:
		return PieceBishop, true
	case CellWhiteRook, CellBlackRook:
		return PieceRook, true
	case CellWhiteQueen, CellBlackQueen:
		return PieceQueen, true
	default:
		panic("bad cell")
	}
}

func (c Cell) ToByte() byte {
	const bytes = ".PKNBRQpknbrq"
	return bytes[c]
}

var cellRunes = [13]rune{
	'.', '♙', '♔', '♘', '♗', '♖', '♕', '♟', '♚', '♞', '♝', '♜', '♛',
}

func (c Cell) ToRune() rune {
	return cellRunes[c]
}

func CellFromByte(b byte) (Cell, error) {
	if b == '.' {
		return CellEmpty, nil
	}
	var (
		c Color
		p Piece
	)
	if 'A' <= b && b <= 'Z' {
		c = ColorWhite
		b ^= 32
	} else {
		c = ColorBlack
	}
	p, ok := doPieceFromByte(b)
	if !ok {
		if c == ColorWhite {
			b ^= 32
		}
		return Cell(0), fmt.Errorf("unexpected cell char %q", b)
	}
	return CellFromParts(c, p), nil
}

func (c Cell) String() string {
	var b strings.Builder
	_ = b.WriteByte(c.ToByte())
	return b.String()
}

func CellFromString(s string) (Cell, error) {
	if len(s) != 1 {
		return Cell(0), fmt.Errorf("invalid string length")
	}
	return CellFromByte(s[0])
}

type CastlingSide uint8

const (
	CastlingQueenside CastlingSide = iota
	CastlingKingside
	CastlingSideMax
)

func (s CastlingSide) IsValid() bool {
	return s < CastlingSideMax
}

type CastlingRights uint8

const (
	CastlingRightsEmpty CastlingRights = 0
	CastlingRightsFull  CastlingRights = 15
	CastlingRightsMax   CastlingRights = CastlingRightsFull + 1
)

func castlingRightsIdx(c Color, s CastlingSide) int {
	return (int(c) << 1) | int(s)
}

func castlingRightsColorMask(c Color) CastlingRights {
	return CastlingRights(3 << (uint8(c) << 1))
}

func (r CastlingRights) IsValid() bool {
	return r < CastlingRightsMax
}

func (r CastlingRights) Has(c Color, s CastlingSide) bool {
	return (uint8(r)>>castlingRightsIdx(c, s))&1 != 0
}

func (r CastlingRights) HasColor(c Color) bool {
	return (r & castlingRightsColorMask(c)) != 0
}

func (r CastlingRights) With(c Color, s CastlingSide) CastlingRights {
	return r | CastlingRights(uint8(1)<<castlingRightsIdx(c, s))
}

func (r CastlingRights) Without(c Color, s CastlingSide) CastlingRights {
	return r & ^CastlingRights(uint8(1)<<castlingRightsIdx(c, s))
}

func (r *CastlingRights) Set(c Color, s CastlingSide) {
	*r = r.With(c, s)
}

func (r *CastlingRights) Unset(c Color, s CastlingSide) {
	*r = r.Without(c, s)
}

func (r *CastlingRights) UnsetColor(c Color) {
	r.Unset(c, CastlingQueenside)
	r.Unset(c, CastlingKingside)
}

func (r CastlingRights) String() string {
	if r == CastlingRightsEmpty {
		return "-"
	}
	var b strings.Builder
	if r.Has(ColorWhite, CastlingKingside) {
		_ = b.WriteByte('K')
	}
	if r.Has(ColorWhite, CastlingQueenside) {
		_ = b.WriteByte('Q')
	}
	if r.Has(ColorBlack, CastlingKingside) {
		_ = b.WriteByte('k')
	}
	if r.Has(ColorBlack, CastlingQueenside) {
		_ = b.WriteByte('q')
	}
	return b.String()
}

func CastlingRightsFromString(s string) (CastlingRights, error) {
	if s == "-" {
		return CastlingRightsEmpty, nil
	}
	if s == "" {
		return CastlingRightsEmpty, fmt.Errorf("string is empty")
	}
	res := CastlingRightsEmpty
	for i := range len(s) {
		var nRes CastlingRights
		b := s[i]
		switch b {
		case 'K':
			nRes = res.With(ColorWhite, CastlingKingside)
		case 'Q':
			nRes = res.With(ColorWhite, CastlingQueenside)
		case 'k':
			nRes = res.With(ColorBlack, CastlingKingside)
		case 'q':
			nRes = res.With(ColorBlack, CastlingQueenside)
		default:
			return CastlingRightsEmpty, fmt.Errorf("unexpected castling rights char %q", b)
		}
		if nRes == res {
			return CastlingRightsEmpty, fmt.Errorf("duplicate castling rights char %q", b)
		}
		res = nRes
	}
	return res, nil
}

type Verdict uint8

const (
	// Running game verdicts
	VerdictRunning Verdict = 0

	// Draw game verdicts
	VerdictDrawUnknown          Verdict = 32
	VerdictStalemate            Verdict = 33
	VerdictInsufficientMaterial Verdict = 34
	VerdictMoves75              Verdict = 35
	VerdictRepeat5              Verdict = 36
	VerdictMoves50              Verdict = 37
	VerdictRepeat3              Verdict = 38
	VerdictDrawAgreement        Verdict = 39

	// Win game verdicts
	VerdictWinUnknown      Verdict = 64
	VerdictCheckmate       Verdict = 65
	VerdictTimeForfeit     Verdict = 66
	VerdictInvalidMove     Verdict = 67
	VerdictEngineError     Verdict = 68
	VerdictResign          Verdict = 69
	VerdictOpponentAbandon Verdict = 70
)

type VerdictKind uint8

const (
	VerdictKindRunning VerdictKind = 0
	VerdictKindDraw    VerdictKind = 1
	VerdictKindWin     VerdictKind = 2
)

type VerdictFilter uint8

const (
	VerdictFilterForce VerdictFilter = iota
	VerdictFilterStrict
	VerdictFilterRelaxed
)

func (v Verdict) Kind() VerdictKind {
	return VerdictKind(uint8(v) >> 5)
}

func (v Verdict) IsFinished() bool {
	return v.Kind() != VerdictKindRunning
}

func (v Verdict) Passes(filter VerdictFilter) bool {
	switch v {
	case VerdictCheckmate, VerdictStalemate, VerdictRunning:
		return filter >= VerdictFilterForce
	case VerdictInsufficientMaterial, VerdictMoves75, VerdictRepeat5:
		return filter >= VerdictFilterStrict
	case VerdictMoves50, VerdictRepeat3:
		return filter >= VerdictFilterRelaxed
	default:
		return false
	}
}

func (v Verdict) String() string {
	switch v {
	case VerdictRunning:
		return ""
	case VerdictDrawUnknown:
		return "draw by unknown reason"
	case VerdictStalemate:
		return "stalemate"
	case VerdictInsufficientMaterial:
		return "insufficient material"
	case VerdictMoves75:
		return "75 move rule"
	case VerdictRepeat5:
		return "fivefold repetition"
	case VerdictMoves50:
		return "50 move rule"
	case VerdictRepeat3:
		return "threefold repetition"
	case VerdictDrawAgreement:
		return "draw by agreement"
	case VerdictWinUnknown:
		return "win by unknown reason"
	case VerdictCheckmate:
		return "checkmate"
	case VerdictTimeForfeit:
		return "opponent forfeits on time"
	case VerdictInvalidMove:
		return "opponent made an invalid move"
	case VerdictEngineError:
		return "opponent is a buggy chess engine"
	case VerdictResign:
		return "opponent resigns"
	case VerdictOpponentAbandon:
		return "opponent abandons the game"
	default:
		return "invalid"
	}
}

type Status uint8

const (
	StatusRunning Status = iota
	StatusDraw
	StatusWhiteWins
	StatusBlackWins
)

func StatusFromString(s string) (Status, error) {
	switch s {
	case "*":
		return StatusRunning, nil
	case "1/2-1/2":
		return StatusDraw, nil
	case "1-0":
		return StatusWhiteWins, nil
	case "0-1":
		return StatusBlackWins, nil
	default:
		return Status(0), fmt.Errorf("bad status string")
	}
}

func StatusWin(c Color) Status {
	switch c {
	case ColorWhite:
		return StatusWhiteWins
	case ColorBlack:
		return StatusBlackWins
	default:
		panic("bad color")
	}
}

func (s Status) IsFinished() bool {
	return s != StatusRunning
}

func (s Status) Winner() (Color, bool) {
	switch s {
	case StatusWhiteWins:
		return ColorWhite, true
	case StatusBlackWins:
		return ColorBlack, true
	default:
		return Color(0), false
	}
}

func (s Status) String() string {
	switch s {
	case StatusRunning:
		return "*"
	case StatusDraw:
		return "1/2-1/2"
	case StatusWhiteWins:
		return "1-0"
	case StatusBlackWins:
		return "0-1"
	default:
		return "?"
	}
}

type Outcome struct {
	verdict Verdict
	side    Color
}

func RunningOutcome() Outcome {
	return Outcome{verdict: VerdictRunning}
}

func DrawOutcome(verdict Verdict) (Outcome, bool) {
	if verdict.Kind() == VerdictKindDraw {
		return Outcome{verdict: verdict}, true
	}
	return Outcome{}, false
}

func MustDrawOutcome(verdict Verdict) Outcome {
	res, ok := DrawOutcome(verdict)
	if !ok {
		panic("not a draw verdict")
	}
	return res
}

func WinOutcome(verdict Verdict, side Color) (Outcome, bool) {
	if verdict.Kind() == VerdictKindWin {
		return Outcome{verdict: verdict, side: side}, true
	}
	return Outcome{}, false
}

func MustWinOutcome(verdict Verdict, side Color) Outcome {
	res, ok := WinOutcome(verdict, side)
	if !ok {
		panic("not a win verdict")
	}
	return res
}

func NewOutcome(verdict Verdict, side Color) Outcome {
	if verdict.Kind() == VerdictKindWin {
		return Outcome{verdict: verdict, side: side}
	}
	return Outcome{verdict: verdict}
}

func (o Outcome) Verdict() Verdict {
	return o.verdict
}

func (o Outcome) Side() (Color, bool) {
	if o.verdict.Kind() == VerdictKindWin {
		return o.side, true
	}
	return Color(0), false
}

func (o Outcome) IsFinished() bool {
	return o.verdict.IsFinished()
}

func (o Outcome) Passes(filter VerdictFilter) bool {
	return o.verdict.Passes(filter)
}

func (o Outcome) Status() Status {
	switch o.verdict.Kind() {
	case VerdictKindRunning:
		return StatusRunning
	case VerdictKindDraw:
		return StatusDraw
	case VerdictKindWin:
		return StatusWin(o.side)
	default:
		panic("bad verdict kind")
	}
}

func (o Outcome) String() string {
	if o.verdict.Kind() != VerdictKindWin {
		return o.verdict.String()
	}
	s := o.side
	switch o.verdict {
	case VerdictWinUnknown:
		return fmt.Sprintf("%s wins by unknown reason", s.LongString())
	case VerdictCheckmate:
		return fmt.Sprintf("%s checkmates", s.LongString())
	case VerdictTimeForfeit:
		return fmt.Sprintf("%s forfeits on time", s.Inv().LongString())
	case VerdictInvalidMove:
		return fmt.Sprintf("%s made an invalid move", s.Inv().LongString())
	case VerdictEngineError:
		return fmt.Sprintf("%s is a buggy chess engine", s.Inv().LongString())
	case VerdictResign:
		return fmt.Sprintf("%s resigns", s.Inv().LongString())
	case VerdictOpponentAbandon:
		return fmt.Sprintf("%s abandons the game", s.Inv().LongString())
	default:
		return "invalid"
	}
}
