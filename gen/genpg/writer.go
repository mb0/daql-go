package genpg

import (
	"fmt"

	"xelf.org/daql/gen"
	"xelf.org/xelf/bfr"
	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

var External = fmt.Errorf("external symbol")

type Writer struct {
	gen.Gen
	Prog *exp.Prog
	Translator
	Params []Param
}

type Param struct {
	Name  string
	Type  typ.Type
	Value lit.Val
}

func NewWriter(b bfr.Writer, p *exp.Prog, t Translator) *Writer {
	return &Writer{gen.Gen{
		P:      bfr.P{Writer: b, Tab: "\t"},
		Header: "-- generated code\n\n",
	}, p, t, nil}
}
func (w *Writer) Translate(p *exp.Prog, env exp.Env, s *exp.Sym) (string, lit.Val, error) {
	for i, p := range w.Params {
		// TODO better way to idetify a reference, maybe in another env
		if p.Name == s.Sym {
			return fmt.Sprintf("$%d", i+1), nil, nil
		}
	}
	if w.Translator == nil {
		return "", nil, exp.ErrDefer
	}
	n, l, err := w.Translator.Translate(p, env, s)
	if err == External {
		if n == "" {
			n = s.Sym
		}
		w.Params = append(w.Params, Param{n, s.Type, l})
		return fmt.Sprintf("$%d", len(w.Params)), nil, nil
	}
	return n, l, err
}

type Translator interface {
	Translate(*exp.Prog, exp.Env, *exp.Sym) (string, lit.Val, error)
}

type ExpEnv struct{}

func (ee ExpEnv) Translate(p *exp.Prog, env exp.Env, s *exp.Sym) (string, lit.Val, error) {
	_, err := env.Resl(p, s, s.Sym)
	if err != nil {
		return "", nil, err
	}
	n := s.Sym
	if n[0] == '.' {
		n = n[1:]
	}
	if cor.IsKey(n) {
		return n, nil, nil
	}
	return s.Sym, nil, External
}
