package uci

import (
	"fmt"
	"strings"

	"github.com/alex65536/go-chess/chess"
)

var (
	_ command = cmdEmpty{}
	_ command = cmdDebug{}
	_ command = cmdIsReady{}
	_ command = cmdSetOption{}
	_ command = cmdUCINewGame{}
	_ command = cmdQuit{}
	_ command = cmdPosition{}
	_ command = cmdGo{}
	_ command = cmdStop{}
	_ command = cmdPonderHit{}
)

type cmdEmpty struct{}

func (c cmdEmpty) uciCommandMarker() {}
func (c cmdEmpty) Serialize() string { panic("must not happen") }

type cmdDebug struct {
	val bool
}

func (c cmdDebug) uciCommandMarker() {}
func (c cmdDebug) Serialize() string {
	if c.val {
		return "debug on"
	} else {
		return "debug off"
	}
}

type cmdIsReady struct{}
type cmdIsReadyRes <-chan error

func (c cmdIsReady) uciCommandMarker() {}
func (c cmdIsReady) Serialize() string { return "isready" }

type cmdSetOption struct {
	name  string
	value OptValue
}

func (c cmdSetOption) uciCommandMarker() {}
func (c cmdSetOption) Serialize() string {
	if _, ok := c.value.(OptValueButton); ok {
		return fmt.Sprintf("setoption name %v", c.name)
	}
	val := c.value.serialize()
	if len(val) == 0 {
		return fmt.Sprintf("setoption name %v value", c.name)
	}
	return fmt.Sprintf("setoption name %v value %v", c.name, val)
}

// Register is not supported.
// type cmdRegister struct{}

type cmdUCINewGame struct{}

func (c cmdUCINewGame) uciCommandMarker() {}
func (c cmdUCINewGame) Serialize() string { return "ucinewgame" }

type cmdQuit struct{}

func (c cmdQuit) uciCommandMarker() {}
func (c cmdQuit) Serialize() string { return "quit" }

type cmdPosition struct {
	start chess.RawBoard
	moves []chess.Move
	board *chess.Board
}

func (c cmdPosition) uciCommandMarker() {}
func (c cmdPosition) Serialize() string {
	var b strings.Builder
	_, _ = b.WriteString("position")
	if c.start == chess.InitialRawBoard() {
		_, _ = b.WriteString(" startpos")
	} else {
		_, _ = fmt.Fprintf(&b, " fen %v", c.start.FEN())
	}
	_, _ = b.WriteString(" moves")
	for _, m := range c.moves {
		_ = b.WriteByte(' ')
		_, _ = b.WriteString(m.UCI())
	}
	return b.String()
}

type cmdGo struct {
	opts GoOptions
	c    searchInfoConsumer
}
type cmdGoRes *searchState

func (c cmdGo) uciCommandMarker() {}
func (c cmdGo) Serialize() string {
	var b strings.Builder
	_, _ = b.WriteString("go")
	if c.opts.Ponder {
		_, _ = b.WriteString(" ponder")
	}
	if s, ok := c.opts.TimeSpec.TryGet(); ok {
		_, _ = fmt.Fprintf(&b, " wtime %v", s.Wtime.Milliseconds())
		_, _ = fmt.Fprintf(&b, " btime %v", s.Btime.Milliseconds())
		if ms := s.Winc.Milliseconds(); ms != 0 {
			_, _ = fmt.Fprintf(&b, " winc %v", ms)
		}
		if ms := s.Binc.Milliseconds(); ms != 0 {
			_, _ = fmt.Fprintf(&b, " binc %v", ms)
		}
		if s.MovesToGo != 0 {
			_, _ = fmt.Fprintf(&b, " movestogo %v", s.MovesToGo)
		}
	}
	if v, ok := c.opts.Depth.TryGet(); ok {
		_, _ = fmt.Fprintf(&b, " depth %v", v)
	}
	if v, ok := c.opts.Nodes.TryGet(); ok {
		_, _ = fmt.Fprintf(&b, " nodes %v", v)
	}
	if v, ok := c.opts.Mate.TryGet(); ok {
		_, _ = fmt.Fprintf(&b, " mate %v", v)
	}
	if v, ok := c.opts.Movetime.TryGet(); ok {
		_, _ = fmt.Fprintf(&b, " movetime %v", v.Milliseconds())
	}
	if c.opts.Infinite {
		_, _ = b.WriteString(" infinite")
	}
	if len(c.opts.SearchMoves) != 0 {
		_, _ = b.WriteString(" searchmoves")
		for _, m := range c.opts.SearchMoves {
			_ = b.WriteByte(' ')
			_, _ = b.WriteString(m.UCI())
		}
	}
	return b.String()
}

type cmdStop struct {
	s *searchState
}

func (c cmdStop) uciCommandMarker() {}
func (c cmdStop) Serialize() string { return "stop" }

type cmdPonderHit struct {
	s *searchState
}

func (c cmdPonderHit) uciCommandMarker() {}
func (c cmdPonderHit) Serialize() string { return "ponderhit" }
