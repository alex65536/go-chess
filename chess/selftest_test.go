//go:build largetest

package chess

import (
	"bufio"
	"cmp"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type selftestOpts struct {
	BigDepth        bool
	DumpTraceChains bool
	RunSelfCheck    bool
	AttackHeatmaps  bool
}

func defaultSelftestOpts() selftestOpts {
	return selftestOpts{
		BigDepth:        true,
		DumpTraceChains: false,
		RunSelfCheck:    true,
		AttackHeatmaps:  true,
	}
}

type selftestDepthSpec struct {
	d        int
	heatmaps bool
}

func (s selftestDepthSpec) name() string {
	suf := ""
	if s.heatmaps {
		suf = "-heatmaps"
	}
	return strconv.FormatInt(int64(s.d), 10) + suf
}

type selftestDepthCtx struct {
	spec  selftestDepthSpec
	hash  uint64
	chain string
}

func (c *selftestDepthCtx) growHash(val uint64) {
	c.hash = c.hash*2579 + val
}

func selftestMoveStrs(moves []Move) []string {
	res := make([]string, len(moves))
	for i, m := range moves {
		res[i] = m.UCI()
	}
	slices.Sort(res)
	return res
}

func selftestMoveHash(m Move) uint64 {
	res := uint64(m.Src().File().ToByte()-'a')*512 +
		uint64(m.Src().Rank().ToByte()-'1')*64 +
		uint64(m.Dst().File().ToByte()-'a')*8 +
		uint64(m.Dst().Rank().ToByte()-'1')
	res *= 5
	if p, ok := m.Kind().Promote(); ok {
		switch p {
		case PieceKnight:
			res += 1
		case PieceBishop:
			res += 2
		case PieceRook:
			res += 3
		case PieceQueen:
			res += 4
		default:
			panic("must not happen")
		}
	}
	return res
}

func selftestDepthDump(b *Board, c *selftestDepthCtx, d int, o *selftestOpts, w io.Writer) error {
	if d == 0 {
		if o.DumpTraceChains {
			if _, err := fmt.Fprintf(w, "cur-chain: %v\n", c.chain); err != nil {
				return fmt.Errorf("format: %w", err)
			}
		}
		if !o.AttackHeatmaps && c.spec.heatmaps {
			return fmt.Errorf("attack heatmaps are disabled")
		}
		if c.spec.heatmaps {
			for color := range ColorMax {
				for r := range RankMax {
					data := uint64(0)
					for f := range FileMax {
						data *= 2
						if b.IsCellAttacked(CoordFromParts(f, r), color) {
							data++
						}
					}
					c.growHash(data)
				}
			}
		}
		isCheck := uint64(0)
		if b.IsCheck() {
			isCheck = 1
		}
		c.growHash(isCheck)
		return nil
	}

	var buf [256]Move
	moves := b.GenLegalMoves(MoveGenAll, buf[:0])
	type moveExt struct {
		m Move
		h uint64
	}
	movesExt := make([]moveExt, len(moves))
	for i, m := range moves {
		movesExt[i] = moveExt{
			m: m,
			h: selftestMoveHash(m),
		}
	}
	slices.SortFunc(movesExt, func(a, b moveExt) int {
		return cmp.Compare(a.h, b.h)
	})

	c.growHash(519365819)
	for _, m := range movesExt {
		oldChain := c.chain
		u := b.MakeLegalMove(m.m)
		if o.DumpTraceChains {
			c.chain += m.m.String() + " "
		}
		c.growHash(m.h)
		err := selftestDepthDump(b, c, d-1, o, w)
		b.UnmakeMove(u)
		c.chain = oldChain
		if err != nil {
			return err
		}
	}
	c.growHash(15967534195)

	return nil
}

func selftestRunOne(fen string, o *selftestOpts, w io.Writer) error {
	b, err := BoardFromFEN(fen)
	if err != nil {
		return fmt.Errorf("parse board: %w", err)
	}
	if _, err := fmt.Fprintf(w, "fen: %v\n", fen); err != nil {
		return fmt.Errorf("format: %w", err)
	}
	if o.RunSelfCheck {
		if err := selfCheck(b); err != nil {
			return fmt.Errorf("selfcheck %q: %w", fen, err)
		}
	}

	var buf [256]Move
	moves := b.GenLegalMoves(MoveGenAll, buf[:0])

	if _, err := io.WriteString(w, "moves: [\n"); err != nil {
		return fmt.Errorf("format: %w", err)
	}
	for _, s := range selftestMoveStrs(moves) {
		if _, err := fmt.Fprintf(w, "  %v\n", s); err != nil {
			return fmt.Errorf("format: %w", err)
		}
	}
	if _, err := io.WriteString(w, "]\n"); err != nil {
		return fmt.Errorf("format: %w", err)
	}

	if _, err := fmt.Fprintf(w, "check?: %v\n", b.IsCheck()); err != nil {
		return fmt.Errorf("format: %w", err)
	}

	if o.AttackHeatmaps {
		for color := range ColorMax {
			if _, err := fmt.Fprintf(w, "%v-heatmap: [\n", color.LongString()); err != nil {
				return fmt.Errorf("format: %w", err)
			}
			for r := range RankMax {
				if _, err := io.WriteString(w, "  "); err != nil {
					return fmt.Errorf("format: %w", err)
				}
				for f := range FileMax {
					ch := '.'
					if b.IsCellAttacked(CoordFromParts(f, r), color) {
						ch = '#'
					}
					if _, err := w.Write([]byte{byte(ch)}); err != nil {
						return fmt.Errorf("format: %w", err)
					}
				}
				if _, err := io.WriteString(w, "\n"); err != nil {
					return fmt.Errorf("format: %w", err)
				}
			}
			if _, err := io.WriteString(w, "]\n"); err != nil {
				return fmt.Errorf("format: %w", err)
			}
		}
	}

	if o.RunSelfCheck {
		for _, m := range moves {
			u := b.MakeLegalMove(m)
			if err := selfCheck(b); err != nil {
				return fmt.Errorf("selfcheck %q: %w", b.FEN(), err)
			}
			b.UnmakeMove(u)
		}
	}

	specs := []selftestDepthSpec{
		{d: 1, heatmaps: o.AttackHeatmaps},
		{d: 2, heatmaps: false},
	}
	if o.BigDepth {
		if o.AttackHeatmaps {
			specs = append(specs, selftestDepthSpec{d: 2, heatmaps: true})
		}
		specs = append(specs, selftestDepthSpec{d: 3, heatmaps: false})
	}
	for _, s := range specs {
		c := selftestDepthCtx{
			spec:  s,
			hash:  0,
			chain: "",
		}
		if err := selftestDepthDump(b, &c, s.d, o, w); err != nil {
			return fmt.Errorf("depth dump %v: %w", s.name(), err)
		}
		if _, err := fmt.Fprintf(w, "depth-dump-at-%v: %v\n", s.name(), c.hash); err != nil {
			return fmt.Errorf("format: %w", err)
		}
	}

	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("format: %w", err)
	}
	return nil
}

func selftestRunMany(o *selftestOpts, r io.Reader, w io.Writer, batch int) error {
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	doRun := func(fens []string) error {
		var gErrMu sync.Mutex
		gErr := error(nil)
		data := make([]*string, len(fens))

		var wg sync.WaitGroup
		for i, fen := range fens {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var b strings.Builder
				if err := selftestRunOne(fen, o, &b); err != nil {
					gErrMu.Lock()
					gErr = err
					gErrMu.Unlock()
					return
				}
				s := b.String()
				data[i] = &s
			}()
		}
		wg.Wait()

		for _, d := range data {
			if d == nil {
				continue
			}
			if _, err := io.WriteString(w, *d); err != nil {
				return fmt.Errorf("format: %w", err)
			}
		}
		return gErr
	}

	fens := make([]string, 0, batch)
	for {
		ln, err := br.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read: %w", err)
		}
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		fens = append(fens, ln)
		if len(fens) == batch {
			if err := doRun(fens); err != nil {
				return err
			}
			fens = fens[:0]
		}
	}
	if len(fens) != 0 {
		if err := doRun(fens); err != nil {
			return err
		}
	}

	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	return nil
}

func selfCheck(b *Board) error {
	boardValidate := func(b *Board) error {
		b2, err := NewBoard(b.r)
		if err != nil {
			return fmt.Errorf("rebuild board: %w", err)
		}
		if *b2 != *b {
			return fmt.Errorf("board is different after rebuild")
		}
		return nil
	}

	sortMoves := func(ms []Move) {
		slices.SortFunc(ms, func(a, b Move) int {
			return cmp.Or(
				cmp.Compare(a.kind, b.kind),
				cmp.Compare(a.srcCell, b.srcCell),
				cmp.Compare(a.src, b.src),
				cmp.Compare(a.dst, b.dst),
			)
		})
	}

	// Check that the board itself is valid
	if err := boardValidate(b); err != nil {
		return err
	}

	// Check that FEN() and BoardFromFEN() are symmetrical
	fen := b.FEN()
	b2, err := BoardFromFEN(fen)
	if err != nil {
		return fmt.Errorf("rebuild board from fen: %w", err)
	}
	if *b2 != *b {
		return fmt.Errorf("board is different after rebuild from fen")
	}

	// Try to generate moves in total and compare the result if we generate simple moves and captures separately
	moves := b.GenSemilegalMoves(MoveGenAll, make([]Move, 0, 256))
	sortMoves(moves)
	movesSimple := b.GenSemilegalMoves(MoveGenSimple, make([]Move, 0, 256))
	sortMoves(movesSimple)

	moves2 := b.GenSemilegalMoves(MoveGenSimpleNoPromote, make([]Move, 0, 256))
	moves2 = b.GenSemilegalMoves(MoveGenSimplePromote, moves2)
	sortMoves(moves2)
	if !slices.Equal(movesSimple, moves2) {
		return fmt.Errorf("movesSimple != moves2")
	}

	moves2 = b.GenSemilegalMoves(MoveGenCapture, moves2)
	sortMoves(moves2)
	if !slices.Equal(moves, moves2) {
		return fmt.Errorf("moves != moves2")
	}

	// Check that all the generated moves are well-formed
	for _, mv := range moves {
		if !mv.IsWellFormed() {
			return fmt.Errorf("move %v:%v:%v:%v is not well-formed", mv.kind, mv.srcCell, mv.src, mv.dst)
		}
	}

	// Check that converting semilegal moves from/to UCI yields the same results
	for _, mv := range moves {
		m2, err := MoveFromUCI(mv.UCI(), b)
		if err != nil {
			return fmt.Errorf("converting %v:%v:%v:%v from UCI: %w", mv.kind, mv.srcCell, mv.src, mv.dst, err)
		}
		if mv != m2 {
			return fmt.Errorf("move %v:%v:%v:%v different after converting from UCI and back", mv.kind, mv.srcCell, mv.src, mv.dst)
		}
	}

	// Check that a well-formed move is generated by GenSemilegalMoves() iff mv.SemiValidate()
	// succeeds. Note that we consider only non-null moves with srcCell.Side() == b.Side().
	semilegals := make([]Move, 0, 256)
	for kind := range MoveKindMax {
		if kind == MoveNull {
			continue
		}
		for p := range PieceMax {
			if !kind.MatchesPiece(p) {
				continue
			}
			for src := range CoordMax {
				for dst := range CoordMax {
					mv, err := NewMove(kind, CellFromParts(b.r.Side, p), src, dst)
					if err != nil {
						continue
					}
					if err := mv.SemiValidate(b); err == nil {
						semilegals = append(semilegals, mv)
					}
				}
			}
		}
	}
	sortMoves(semilegals)
	if !slices.Equal(moves, semilegals) {
		return fmt.Errorf("moves != semilegals")
	}

	// Check that making and unmaking move doesn't change anything
	legals1 := make([]Move, 0, 256)
	b2 = b.Clone()
	for _, mv := range moves {
		u, ok := b.MakeSemilegalMove(mv)
		if ok {
			if err := boardValidate(b); err != nil {
				return err
			}
			legals1 = append(legals1, mv)
			b.UnmakeMove(u)
		}
		if *b2 != *b {
			return fmt.Errorf("b2 and b diverged on move %v", mv)
		}
	}
	sortMoves(legals1)

	// Check that legal moves are determined correctly in three ways
	// (MakeSemilegalMove(), mv.IsLegalWhenSemilegal() and legal movegen).
	legals2 := make([]Move, 0, 256)
	for _, mv := range moves {
		if mv.IsLegalWhenSemilegal(b) {
			legals2 = append(legals2, mv)
		}
	}
	sortMoves(legals2)
	if !slices.Equal(legals1, legals2) {
		return fmt.Errorf("legals1 != legals2")
	}

	legals3 := b.GenLegalMoves(MoveGenAll, make([]Move, 0, 256))
	sortMoves(legals3)
	if !slices.Equal(legals1, legals3) {
		return fmt.Errorf("legals1 != legals3")
	}

	// Check that converting legal moves from/to SAN yields the same results
	for _, m := range legals1 {
		san, err := m.Styled(b, MoveStyleSAN)
		if err != nil {
			return fmt.Errorf("convert %v to SAN: %w", m, err)
		}
		m2, err := LegalMoveFromSAN(san, b)
		if err != nil {
			return fmt.Errorf("convert %v from SAN %q: %w", m, san, err)
		}
		if m != m2 {
			return fmt.Errorf("move %v different after converting from SAN and back", m)
		}
	}

	// Check that srcCell works correctly
	for _, mv := range moves {
		if mv.srcCell != b.Get(mv.src) {
			return fmt.Errorf("move %v has bad src cell %v", mv, mv.srcCell)
		}
	}

	// Check that HasLegalMoves() returns true iff there are legal moves
	hasLegals := b.HasLegalMoves()
	if hasLegals != (len(legals1) != 0) {
		return fmt.Errorf("hasLegals != (len(legals1) != 0)")
	}

	return nil
}

func TestSelftest(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "boards.fen"))
	require.NoError(t, err)
	defer f.Close()

	of, err := os.CreateTemp("", "chess_selftest")
	require.NoError(t, err)
	defer of.Close()

	hsh := sha256.New()

	w := io.MultiWriter(of, hsh)
	opts := defaultSelftestOpts()
	const batch = 512
	err = selftestRunMany(&opts, f, w, batch)
	require.NoError(t, err)

	sum := fmt.Sprintf("%x", hsh.Sum(nil))
	require.Equalf(
		t,
		"e0d9b6799e2bd0ae9e69f9e2757acbb3e5072fa6c667ee3e5745d90cfd2ed18d",
		sum,
		"inspect %q for details",
		of.Name(),
	)

	err = os.Remove(of.Name())
	require.NoError(t, err)
}
