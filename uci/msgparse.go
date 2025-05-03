package uci

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/alex65536/go-chess/chess"
	"github.com/alex65536/go-chess/util/maybe"
)

var (
	optionKeywords = map[string]struct{}{
		"name":    {},
		"type":    {},
		"default": {},
		"min":     {},
		"max":     {},
		"var":     {},
	}

	moveRe = regexp.MustCompile("^[a-h][1-8][a-h][1-8][nbrq]?$")
)

func parseOption(tok *tokenizer, l Logger) (optPair, error) {
	var (
		name   *string
		typ    *string
		def    *string
		minV   *string
		maxV   *string
		choice []string
	)
	visited := make(map[string]struct{})

	isKeyword := func(s string) bool {
		_, ok := optionKeywords[s]
		return ok
	}

	for {
		sub, ok := tok.Next()
		if !ok {
			break
		}
		if sub != "var" {
			if _, ok := visited[sub]; ok {
				l.Printf("parse \"option\": duplicate %q", sub)
			}
			visited[sub] = struct{}{}
		}
		switch sub {
		case "name":
			s := tok.NextUntil(isKeyword)
			name = &s
		case "type":
			s, ok := tok.Next()
			if !ok {
				return optPair{}, fmt.Errorf("no type after \"type\"")
			}
			typ = &s
		case "default":
			s := tok.NextUntil(isKeyword)
			def = &s
		case "min":
			s, ok := tok.Next()
			if !ok {
				return optPair{}, fmt.Errorf("no value after \"min\"")
			}
			minV = &s
		case "max":
			s, ok := tok.Next()
			if !ok {
				return optPair{}, fmt.Errorf("no value after \"max\"")
			}
			maxV = &s
		case "var":
			choice = append(choice, tok.NextUntil(isKeyword))
		default:
			l.Printf("parse \"option\": bad token %q", sub)
		}
	}

	if name == nil {
		return optPair{}, fmt.Errorf("no name")
	}
	if len(*name) == 0 {
		return optPair{}, fmt.Errorf("empty name")
	}
	if typ == nil {
		return optPair{}, fmt.Errorf("no type")
	}

	o := optPair{name: *name}

	switch *typ {
	case "check":
		if minV != nil || maxV != nil || choice != nil {
			l.Printf("parse \"option\": extra data for \"check\"")
		}
		if def == nil {
			return optPair{}, fmt.Errorf("no default for \"check\"")
		}
		var val bool
		switch *def {
		case "true":
			val = true
		case "false":
			val = false
		default:
			return optPair{}, fmt.Errorf("bad default for \"check\": %q", *def)
		}
		o.value = &OptionCheck{val: val}
	case "spin":
		if choice != nil {
			l.Printf("parse \"option\": extra data for \"spin\"")
		}
		if def == nil {
			return optPair{}, fmt.Errorf("no default for \"spin\"")
		}
		minI, maxI := int64(math.MinInt64), int64(math.MaxInt64)
		var err error
		if minV != nil {
			minI, err = strconv.ParseInt(*minV, 10, 64)
			if err != nil {
				l.Printf("parse \"option\": bad min: %q", *minV)
				minI = int64(math.MinInt64)
			}
		}
		if maxV != nil {
			maxI, err = strconv.ParseInt(*maxV, 10, 64)
			if err != nil {
				l.Printf("parse \"option\": bad max: %q", *minV)
				maxI = int64(math.MaxInt64)
			}
		}
		defI, err := strconv.ParseInt(*def, 10, 64)
		if err != nil {
			return optPair{}, fmt.Errorf("bad default for \"spin\": %q", *def)
		}
		if !(minI <= defI && defI <= maxI) {
			return optPair{}, fmt.Errorf("default %v out of range [%v; %v]", defI, minI, maxI)
		}
		o.value = &OptionSpin{
			val:    defI,
			minVal: minI,
			maxVal: maxI,
		}
	case "combo":
		if minV != nil || maxV != nil {
			l.Printf("parse \"option\": extra data for \"combo\"")
		}
		if def == nil {
			return optPair{}, fmt.Errorf("no default for \"combo\"")
		}
		realChoices := make([]string, 0, len(choice))
		choiceMap := make(map[string]string)
		for _, s := range choice {
			if s == "" {
				l.Printf("parse \"option\": empty choice")
				continue
			}
			folded := caseFold(s)
			if _, ok := choiceMap[folded]; ok {
				l.Printf("parse \"option\": duplicate choice %q", s)
				continue
			}
			choiceMap[folded] = s
			realChoices = append(realChoices, s)
		}
		val, ok := choiceMap[caseFold(*def)]
		if !ok {
			return optPair{}, fmt.Errorf("default for \"combo\" is not in choices")
		}
		o.value = &OptionCombo{
			val:       val,
			choices:   realChoices,
			choiceMap: choiceMap,
		}
	case "button":
		if minV != nil || maxV != nil || def != nil || choice != nil {
			l.Printf("parse \"option\": extra data for \"button\"")
		}
		o.value = &OptionButton{}
	case "string":
		if minV != nil || maxV != nil || choice != nil {
			l.Printf("parse \"option\": extra data for \"string\"")
		}
		if def == nil {
			return optPair{}, fmt.Errorf("no default for \"string\"")
		}
		val := *def
		if val == "<empty>" {
			val = ""
		}
		o.value = &OptionString{
			val: val,
		}
	default:
		return optPair{}, fmt.Errorf("unknown type %v", *typ)
	}

	return o, nil
}

func parseInfo(tok *tokenizer, l Logger) (Info, error) {
	info := Info{}
	parsed := make(map[string]struct{})
	for tok.More() {
		kw, _ := tok.Next()
		if _, ok := parsed[kw]; ok {
			l.Printf("parse \"info\": duplicate %q", kw)
		}
		parsed[kw] = struct{}{}

		doParseInt64 := func(bits int) (int64, error) {
			t, ok := tok.Next()
			if !ok {
				return 0, fmt.Errorf("end of line")
			}
			n, err := strconv.ParseInt(t, 10, bits)
			if err != nil {
				return 0, fmt.Errorf("bad value %q", t)
			}
			return n, nil
		}

		parseInt := func(target *maybe.Maybe[int]) error {
			n, err := doParseInt64(0)
			if err != nil {
				return err
			}
			*target = maybe.Some(int(n))
			return nil
		}

		parseInt32 := func(target *maybe.Maybe[int32]) error {
			n, err := doParseInt64(32)
			if err != nil {
				return err
			}
			*target = maybe.Some(int32(n))
			return nil
		}

		parseInt64 := func(target *maybe.Maybe[int64]) error {
			n, err := doParseInt64(64)
			if err != nil {
				return err
			}
			*target = maybe.Some(n)
			return nil
		}

		parseTime := func(target *maybe.Maybe[time.Duration]) error {
			n, err := doParseInt64(64)
			if err != nil {
				return err
			}
			if n > math.MaxInt64/int64(time.Millisecond) {
				return fmt.Errorf("time too large: %v", n)
			}
			*target = maybe.Some(time.Duration(n) * time.Millisecond)
			return nil
		}

		doParsePermille := func() (float64, error) {
			n, err := doParseInt64(64)
			if err != nil {
				return 0.0, err
			}
			if n > 1000 {
				return 0.0, fmt.Errorf("permille value too large: %v", n)
			}
			if n < 0 {
				return 0.0, fmt.Errorf("permille value too small: %v", n)
			}
			return float64(n) / 1000.0, nil
		}

		parsePermille := func(target *maybe.Maybe[float64]) error {
			p, err := doParsePermille()
			if err != nil {
				return err
			}
			*target = maybe.Some(p)
			return nil
		}

		parseMove := func(target *maybe.Maybe[chess.UCIMove]) error {
			t, ok := tok.Next()
			if !ok {
				return fmt.Errorf("end of line")
			}
			mv, err := chess.UCIMoveFromString(t)
			if err != nil {
				return fmt.Errorf("bad uci move %q: %w", mv, err)
			}
			*target = maybe.Some(mv)
			return nil
		}

		parseMoves := func(target *[]chess.UCIMove) error {
			*target = make([]chess.UCIMove, 0)
			for {
				t, ok := tok.Next()
				if !ok {
					return nil
				}
				if !moveRe.MatchString(t) {
					tok.Undo()
					return nil
				}
				mv, err := chess.UCIMoveFromString(t)
				if err != nil {
					return fmt.Errorf("bad uci move %q: %w", mv, err)
				}
				*target = append(*target, mv)
			}
		}

		parseScore := func(target *maybe.Maybe[BoundedScore]) error {
			bound := maybe.None[ScoreBound]()
			cp := maybe.None[int32]()
			mate := maybe.None[int32]()
		loop:
			for {
				kw, ok := tok.Next()
				if !ok {
					break
				}
				var err error
				switch kw {
				case "cp":
					if cp.IsSome() {
						l.Printf("parse \"info\": parse \"score\": duplicate cp")
					}
					err = parseInt32(&cp)
				case "mate":
					if mate.IsSome() {
						l.Printf("parse \"info\": parse \"score\": duplicate mate")
					}
					err = parseInt32(&mate)
				case "lowerbound", "upperbound":
					if bound.IsSome() {
						l.Printf("parse \"info\": parse \"score\": duplicate bound")
					}
					if kw[0] == 'l' {
						bound = maybe.Some(ScoreLower)
					} else {
						bound = maybe.Some(ScoreUpper)
					}
				default:
					tok.Undo()
					break loop
				}
				if err != nil {
					return fmt.Errorf("parse %q: %w", kw, err)
				}
			}
			if cp.IsNone() && mate.IsNone() {
				return fmt.Errorf("no score")
			}
			if cp.IsSome() && mate.IsSome() {
				return fmt.Errorf("ambiguous score")
			}
			var s BoundedScore
			s.Bound = bound.GetOr(ScoreExact)
			if cp.IsSome() {
				s.Score = ScoreCentipawns(cp.Get())
			} else {
				s.Score = ScoreMate(mate.Get())
			}
			*target = maybe.Some(s)
			return nil
		}

		parseCurrLine := func(t1 *[]chess.UCIMove, t2 *maybe.Maybe[int]) error {
			t, ok := tok.Next()
			*t1 = nil
			*t2 = maybe.None[int]()
			if ok {
				n, err := strconv.ParseInt(t, 10, 0)
				if err != nil {
					tok.Undo()
				} else {
					*t2 = maybe.Some(int(n))
				}
			}
			if err := parseMoves(t1); err != nil {
				return fmt.Errorf("parse currline: %w", err)
			}
			return nil
		}

		parseWDL := func(target *maybe.Maybe[WDL]) error {
			var (
				wdl WDL
				err error
			)
			if wdl.Win, err = doParsePermille(); err != nil {
				return fmt.Errorf("parse win: %w", err)
			}
			if wdl.Draw, err = doParsePermille(); err != nil {
				return fmt.Errorf("parse draw: %w", err)
			}
			if wdl.Loss, err = doParsePermille(); err != nil {
				return fmt.Errorf("parse loss: %w", err)
			}
			*target = maybe.Some(wdl)
			return nil
		}

		var err error
		switch kw {
		case "depth":
			err = parseInt(&info.Depth)
		case "seldepth":
			err = parseInt(&info.Seldepth)
		case "time":
			err = parseTime(&info.Time)
		case "nodes":
			err = parseInt64(&info.Nodes)
		case "pv":
			err = parseMoves(&info.PV)
		case "multipv":
			err = parseInt(&info.MultiPV)
		case "score":
			err = parseScore(&info.Score)
		case "currmove":
			err = parseMove(&info.CurMove)
		case "currmovenumber":
			err = parseInt(&info.CurMoveNumber)
		case "hashfull":
			err = parsePermille(&info.HashFull)
		case "nps":
			err = parseInt64(&info.NPS)
		case "tbhits":
			err = parseInt64(&info.TBHits)
		case "sbhits":
			err = parseInt64(&info.SBHits)
		case "cpuload":
			err = parsePermille(&info.CPULoad)
		case "string":
			info.String = maybe.Some(tok.NextUntilEnd())
			err = nil
		case "refutation":
			err = parseMoves(&info.Refutation)
		case "currline":
			err = parseCurrLine(&info.CurLine, &info.CurLineCPU)
		case "wdl":
			err = parseWDL(&info.WDL)
		default:
			err = fmt.Errorf("bad keyword")
		}
		if err != nil {
			l.Printf("parse \"info\": parse %q: %v", kw, err)
		}
	}

	return info, nil
}

func parseBestMove(tok *tokenizer, l Logger) ([]chess.UCIMove, error) {
	t, ok := tok.Next()
	if !ok {
		return nil, fmt.Errorf("end of line")
	}
	mv, err := chess.UCIMoveFromString(t)
	if !ok {
		return nil, fmt.Errorf("parse best move: %w", err)
	}
	buf := [2]chess.UCIMove{mv, {}}
	res := buf[:1]
	t, ok = tok.Next()
	if !ok {
		return res, nil
	}
	if t != "ponder" {
		l.Printf("parse \"bestmove\": bad token %q", t)
		return res, nil
	}
	t, ok = tok.Next()
	if !ok {
		l.Printf("parse \"bestmove\": missing ponder move", t)
		return res, nil
	}
	mv, err = chess.UCIMoveFromString(t)
	if err != nil {
		l.Printf("parse \"bestmove\": bad ponder move %w", t, err)
		return res, nil
	}
	res = append(res, mv)
	if tok.More() {
		l.Printf("parse \"bestmove\": extra data")
	}
	return res, nil
}
