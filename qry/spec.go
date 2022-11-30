package qry

import (
	"fmt"

	"xelf.org/xelf/cor"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/knd"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// Spec is the implementation to resolve calls on query subjects. It sets up the job environment,
// resolves arguments to form the task and calls the backend to execute the job when evaluated.
type Spec struct {
	exp.SpecBase
	Doc  *Doc
	Task Task
}

func (s *Spec) Resl(p *exp.Prog, par exp.Env, c *exp.Call, h typ.Type) (exp.Exp, error) {
	t := &s.Task
	j, ok := c.Env.(*Job)
	if !ok {
		j = &Job{Doc: s.Doc, Env: par, Task: t}
		c.Env = j
		s.Doc.Add(j)
	}
	if t.Subj.Type == typ.Void {
		if j.Model != nil {
			t.Subj.Type = j.Model.Type()
			t.Subj.Fields = subjFields(t.Subj.Type)
		} else {
			res, err := p.Resl(par, &exp.Sym{Sym: t.Ref}, typ.Void)
			if err != nil {
				return c, err
			}
			rt := typ.Res(res.Type())
			if rt.Kind&knd.List == 0 {
				return nil, fmt.Errorf("resl subj want list got %s", rt)
			}
			t.Subj.Type = typ.ContEl(rt)
			t.Subj.Fields = subjFields(t.Subj.Type)
		}
	}
	whr, args := splitPlain(c.Args[0].(*exp.Tupl).Els)
	tags, decl := splitDecls(args)
	if t.Sel == nil {
		// resolve selection
		sel, err := reslSel(p, j, decl)
		if err != nil {
			return c, fmt.Errorf("resl sel %v: %v", decl, err)
		}
		t.Sel = sel
	}
	if t.Res == typ.Void {
		switch t.Kind {
		case KindCount:
			t.Res = typ.Int
		case KindOne:
			t.Res = typ.Opt(t.Sel.Type)
		case KindMany:
			t.Res = typ.ListOf(t.Sel.Type)
		}
	}
	t.Ord = nil
	// resolve arguments for whr ord lim and off
	for _, tag := range tags {
		var err error
		switch tag.Tag {
		case "whr":
			whr = append(whr, tag.Exp)
		case "lim", "off":
			el, err := p.Eval(j, tag.Exp)
			if err != nil {
				return c, err
			}
			n, err := lit.ToInt(el.Val)
			if err != nil {
				return c, err
			}
			if tag.Tag == "lim" {
				t.Lim = int64(n)
			} else {
				t.Off = int64(n)
			}
		case "ord", "asc", "desc":
			// takes one or more field references
			// can be used multiple times to append to order
			err = evalOrd(p, par, t, tag.Tag == "desc", tag.Exp)
		default:
			return c, fmt.Errorf("unexpected query tag %q", tag.Tag)
		}
		if err != nil {
			return c, err
		}
	}
	for i, w := range whr {
		res, err := p.Resl(j, w, typ.Bool)
		if err != nil {
			return c, err
		}
		whr[i] = res
	}
	t.Whr = whr
	return c, nil
}

func (s *Spec) Eval(p *exp.Prog, c *exp.Call) (*exp.Lit, error) {
	// query arguments must all be evaluated or else skip until next run
	j := c.Env.(*Job)
	v, err := j.Bend.Exec(p, j)
	if err != nil {
		return nil, err
	}
	v.Src = c.Src
	j.Val = v
	return j.Val, nil
}

func splitPlain(args []exp.Exp) (plain, rest []exp.Exp) {
	for i, arg := range args {
		switch t := arg.(type) {
		case *exp.Tag:
			return args[:i], args[i:]
		case *exp.Sym:
			switch t.Sym {
			case "_", "+", "-":
				return args[:i], args[i:]
			}
		}
	}
	return args, nil
}

func splitDecls(args []exp.Exp) (tags, decl []*exp.Tag) {
	tags = make([]*exp.Tag, 0, len(args))
	decl = make([]*exp.Tag, 0, len(args))
	for _, arg := range args {
		switch t := arg.(type) {
		case *exp.Tag:
			if len(decl) == 0 && cor.IsKey(t.Tag) && t.Tag[0] != '_' {
				tags = append(tags, t)
			} else {
				decl = append(decl, t)
			}
		case *exp.Sym:
			decl = append(decl, &exp.Tag{Tag: t.Sym, Src: t.Src})
		default:
			decl = append(decl, &exp.Tag{Exp: arg, Src: arg.Source()})
		}
	}
	return tags, decl
}

func evalOrd(p *exp.Prog, env exp.Env, t *Task, desc bool, arg exp.Exp) error {
	sym, ok := arg.(*exp.Sym)
	if !ok || sym.Sym == "" {
		return fmt.Errorf("order want sym got %s", arg)
	}
	f := t.Sel.Field(sym.Sym)
	ord := Ord{sym.Sym, desc, f == nil}
	if ord.Subj {
		f = t.Subj.Field(sym.Sym)
	}
	if f == nil {
		return fmt.Errorf("field %s not found in %s", sym.Sym, t.Subj.Type)
	}
	t.Ord = append(t.Ord, ord)
	return nil
}
