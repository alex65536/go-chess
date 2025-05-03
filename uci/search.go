package uci

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/alex65536/go-chess/chess"
	"github.com/alex65536/go-chess/clock"
	"github.com/alex65536/go-chess/util/maybe"
)

type searchInfoConsumer func(*searchState, Info)

type GoOptions struct {
	SearchMoves []chess.Move
	Ponder      bool
	Infinite    bool
	TimeSpec    maybe.Maybe[clock.UCITimeSpec]
	Depth       maybe.Maybe[int64]
	Nodes       maybe.Maybe[int64]
	Mate        maybe.Maybe[int64]
	Movetime    maybe.Maybe[time.Duration]
}

func (g GoOptions) Clone() GoOptions {
	g.SearchMoves = slices.Clone(g.SearchMoves)
	return g
}

func (g GoOptions) Validate(b *chess.Board) error {
	var used map[chess.Move]struct{}
	if len(g.SearchMoves) != 0 {
		used = make(map[chess.Move]struct{})
	}
	for _, m := range g.SearchMoves {
		if _, ok := used[m]; ok {
			return fmt.Errorf("move %v is in searchmoves twice", m)
		}
		used[m] = struct{}{}
		if err := m.Validate(b); err != nil {
			return fmt.Errorf("bad move %v", m)
		}
	}

	if g.Ponder &&
		(g.Infinite ||
			g.Depth.IsSome() || g.Nodes.IsSome() || g.Mate.IsSome() || g.Movetime.IsSome()) {
		return fmt.Errorf("conflicting options with ponder")
	}

	if g.Infinite &&
		(g.TimeSpec.IsSome() || g.Depth.IsSome() || g.Nodes.IsSome() || g.Mate.IsSome() || g.Movetime.IsSome()) {
		return fmt.Errorf("conflicting options with infinite")
	}

	if s, ok := g.TimeSpec.TryGet(); ok {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("invalid time spec: %w", err)
		}
	}

	if g.Depth.IsSome() && g.Depth.Get() <= 0 {
		return fmt.Errorf("non-positive depth")
	}

	if g.Nodes.IsSome() && g.Nodes.Get() <= 0 {
		return fmt.Errorf("non-positive nodes")
	}

	if g.Mate.IsSome() && g.Mate.Get() <= 0 {
		return fmt.Errorf("non-positive mate")
	}

	if g.Movetime.IsSome() && g.Movetime.Get() <= 0 {
		return fmt.Errorf("non-positive movetime")
	}

	return nil
}

type WDL struct {
	Win  float64
	Draw float64
	Loss float64
}

type Info struct {
	Depth         maybe.Maybe[int]
	Seldepth      maybe.Maybe[int]
	Time          maybe.Maybe[time.Duration]
	Nodes         maybe.Maybe[int64]
	PV            []chess.UCIMove
	MultiPV       maybe.Maybe[int]
	Score         maybe.Maybe[BoundedScore]
	CurMove       maybe.Maybe[chess.UCIMove]
	CurMoveNumber maybe.Maybe[int]
	HashFull      maybe.Maybe[float64]
	NPS           maybe.Maybe[int64]
	TBHits        maybe.Maybe[int64]
	SBHits        maybe.Maybe[int64]
	CPULoad       maybe.Maybe[float64]
	String        maybe.Maybe[string]
	Refutation    []chess.UCIMove
	CurLine       []chess.UCIMove
	CurLineCPU    maybe.Maybe[int]
	WDL           maybe.Maybe[WDL]
}

type SearchStatus struct {
	Depth    int
	Time     time.Duration
	Nodes    int64
	PV       []chess.UCIMove
	Score    maybe.Maybe[Score]
	HashFull maybe.Maybe[float64]
	NPS      int64
	WDL      maybe.Maybe[WDL]
}

func (s SearchStatus) Clone() SearchStatus {
	s.PV = slices.Clone(s.PV)
	return s
}

type searchState struct {
	c searchInfoConsumer
	l Logger

	mu       sync.RWMutex
	done     chan struct{}
	err      error
	s        SearchStatus
	ponder   bool
	stopped  bool
	stopping bool
	best     []chess.Move
	start    time.Time
	b        *chess.Board
}

func newSearchState(c searchInfoConsumer, l Logger, b *chess.Board, ponder bool) *searchState {
	return &searchState{
		c: c,
		l: l,

		done:     make(chan struct{}),
		err:      nil,
		s:        SearchStatus{},
		ponder:   ponder,
		stopped:  false,
		stopping: false,
		best:     nil,
		start:    time.Now(),
		b:        b.Clone(),
	}
}

func (s *searchState) OnInfo(info Info, strOnly bool) error {
	defer func() { s.c(s, info) }()

	if strOnly {
		// Info contains only string, nothing to handle.
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		panic("must not happen")
	}
	s.s.Time = time.Since(s.start)
	if d, ok := info.Depth.TryGet(); ok {
		s.s.Depth = d
	}
	if n, ok := info.Nodes.TryGet(); ok {
		s.s.Nodes = n
		if s.s.Time.Nanoseconds() <= 0 {
			s.s.NPS = 0
		} else {
			nps := float64(s.s.Nodes) / float64(s.s.Time.Nanoseconds()) * 1e9
			if nps >= math.MaxInt64 {
				s.s.NPS = math.MaxInt64
			} else {
				s.s.NPS = int64(nps)
			}
		}
	}
	if p := info.PV; p != nil && info.MultiPV.GetOr(1) == 1 {
		s.s.PV = slices.Clone(p)
	}
	if sc, ok := info.Score.TryGet(); ok && sc.Bound == ScoreExact {
		s.s.Score = maybe.Some(sc.Score)
	}
	if h, ok := info.HashFull.TryGet(); ok {
		s.s.HashFull = maybe.Some(h)
	}
	if wdl, ok := info.WDL.TryGet(); ok {
		s.s.WDL = maybe.Some(wdl)
	}
	return nil
}

func (s *searchState) OnPonderHit() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		panic("must not happen")
	}
	if s.stopping {
		return fmt.Errorf("cannot do \"ponderhit\" after \"stop\"")
	}
	if !s.ponder {
		return fmt.Errorf("not pondering at the moment")
	}
	s.ponder = false
	return nil
}

func (s *searchState) OnStop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		panic("must not happen")
	}
	if s.stopping {
		return nil
	}
	s.stopping = true
	return nil
}

func (s *searchState) doStop(err error) {
	if s.stopped {
		panic("must not happen")
	}
	s.stopping = false
	s.stopped = true
	s.err = err
	close(s.done)
}

func (s *searchState) OnBestMove(best chess.UCIMove, ponder maybe.Maybe[chess.UCIMove]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		panic("must not happen")
	}

	var retErr error
	defer func() {
		if s.ponder {
			retErr = errors.Join(retErr, fmt.Errorf("search stopped by engine during ponder"))
		}
		if retErr != nil {
			s.l.Printf("process \"bestmove\": %v", retErr)
		}
		s.doStop(retErr)
	}()

	var buf [2]chess.Move
	s.best = buf[:0]

	m, err := chess.LegalMoveFromUCIMove(best, s.b)
	if err != nil {
		retErr = fmt.Errorf("convert best move: %w", err)
		return
	}
	s.best = append(s.best, m)

	if ponder.IsSome() && ponder.Get().Kind() != chess.UCIMoveNull {
		u := s.b.MakeLegalMove(m)
		m, err = chess.LegalMoveFromUCIMove(ponder.Get(), s.b)
		s.b.UnmakeMove(u)
		if err != nil {
			retErr = fmt.Errorf("convert ponder move: %w", err)
			return
		}
		s.best = append(s.best, m)
	}
}

func (s *searchState) Cancel(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doStop(err)
}

func (s *searchState) Done() <-chan struct{} {
	return s.done
}

func (s *searchState) Err() error {
	select {
	case <-s.done:
		return s.err
	default:
		return nil
	}
}

func (s *searchState) Status() SearchStatus {
	t := time.Since(s.start)
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := s.s.Clone()
	res.Time = t
	return res
}

func (s *searchState) Ponder() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ponder
}

func (s *searchState) Stopping() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stopping
}

func (s *searchState) Stopped() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stopped
}

func (s *searchState) BestMove() (chess.Move, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.stopped {
		return chess.Move{}, fmt.Errorf("search still running")
	}
	if s.err != nil {
		return chess.Move{}, fmt.Errorf("search errored: %w", s.err)
	}
	switch len(s.best) {
	case 0:
		return chess.Move{}, fmt.Errorf("no best move")
	default:
		return s.best[0], nil
	}
}

func (s *searchState) PonderMove() (chess.Move, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.stopped {
		return chess.Move{}, false, fmt.Errorf("search still running")
	}
	if s.err != nil {
		return chess.Move{}, false, fmt.Errorf("search errored: %w", s.err)
	}
	switch len(s.best) {
	case 0:
		return chess.Move{}, false, fmt.Errorf("no best move")
	case 1:
		return chess.Move{}, false, nil
	default:
		return s.best[1], true, nil
	}
}
