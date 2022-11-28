package qry

import (
	"fmt"
	"sort"

	"xelf.org/daql/dom"
	"xelf.org/daql/mig"
	"xelf.org/xelf/exp"
	"xelf.org/xelf/lib"
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// LitBackend is a query backend that operates on literal values from the program environment.
type LitBackend struct{}

func (LitBackend) Proj() *dom.Project { return nil }
func (LitBackend) Exec(p *exp.Prog, j *Job) (*exp.Lit, error) {
	a, err := p.Eval(j.Env, &exp.Sym{Sym: j.Ref})
	if err != nil {
		return nil, fmt.Errorf("lit backend: %w", err)
	}
	var vals lit.Vals
	switch v := a.Value().(type) {
	case *lit.Vals:
		vals = *v
	case *lit.List:
		vals = v.Vals
	case lit.Idxr:
		vals = make(lit.Vals, 0, v.Len())
		v.IterIdx(func(idx int, el lit.Val) error {
			vals = append(vals, el)
			return nil
		})
	default:
		return nil, fmt.Errorf("literal query expects list got %T", a)
	}
	return execListQry(p, j, vals)
}

// MemBackend is a query backend that evaluates queries using in-memory literal values.
type MemBackend struct {
	*dom.Project
	*mig.Version
	Data map[string]*lit.List
}

// NewMemBackend returns a new memory backend for the given project.
func NewMemBackend(pr *dom.Project, v *mig.Version) *MemBackend {
	return &MemBackend{pr, v, make(map[string]*lit.List)}
}

func (b *MemBackend) Proj() *dom.Project { return b.Project }
func (b *MemBackend) Vers() *mig.Version { return b.Version }
func (b *MemBackend) Keys() (res []string) {
	for key := range b.Data {
		res = append(res, key)
	}
	return res
}
func (b *MemBackend) Close() error { return nil }
func (b *MemBackend) Stream(key string) (mig.Stream, error) {
	m := b.Project.Model(key)
	if m == nil {
		return nil, fmt.Errorf("stream %s not found", key)
	}
	return mig.NewLitStream(b.list(m)), nil
}
func (b *MemBackend) Exec(p *exp.Prog, j *Job) (*exp.Lit, error) {
	return execListQry(p, j, b.list(j.Model).Vals)
}
func (b *MemBackend) list(m *dom.Model) (list *lit.List) {
	if list = b.Data[m.Qualified()]; list == nil {
		list = &lit.List{El: m.Type()}
	}
	return list
}

// Add converts and adds a nested list of values to this backend.
func (b *MemBackend) Add(m *dom.Model, list *lit.Vals) error {
	if b.Data == nil {
		b.Data = make(map[string]*lit.List)
	}
	mt := m.Type()
	for i, v := range *list {
		l := v.(*lit.Vals)
		s := &lit.Obj{Typ: mt, Vals: *l}
		(*list)[i] = s
	}
	b.Data[m.Qualified()] = &lit.List{El: mt, Vals: *list}
	return nil
}

var _ mig.Dataset = (*MemBackend)(nil)

func execListQry(p *exp.Prog, j *Job, vals lit.Vals) (*exp.Lit, error) {
	var whr exp.Exp
	if len(j.Whr) > 0 {
		whr = &exp.Call{Args: append([]exp.Exp{exp.LitVal(exp.NewSpecRef(lib.And))}, j.Whr...)}
	}
	if j.Kind == KindCount {
		return collectCount(p, j, vals, whr)
	}
	res, err := collectList(p, j, vals, whr)
	if err != nil {
		return nil, err
	}
	l := &exp.Lit{Res: j.Res}
	switch j.Kind {
	case KindOne:
		if len(res) == 0 {
			l.Val = lit.Null{}
		} else {
			l.Val = res[0]
		}
	case KindMany:
		l.Val = &lit.List{El: typ.El(j.Res), Vals: res}
	default:
		return nil, fmt.Errorf("exec unknown query kind %s", j.Ref)
	}
	return l, nil
}

func collectList(p *exp.Prog, j *Job, vals lit.Vals, whr exp.Exp) (res lit.Vals, _ error) {
	res = make([]lit.Val, 0, len(vals))
	org := vals
	if whr != nil {
		org = make([]lit.Val, 0, len(vals))
	}
	for _, l := range vals {
		j.Cur = l
		if whr != nil {
			ok, err := filter(p, j, l, whr)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			org = append(org, l)
		}
		if len(j.Sel.Fields) == 0 {
			res = append(res, l)
			continue
		}
		rec := l.(lit.Keyr)
		px := p.Reg.Zero(j.Sel.Type)
		z, ok := px.(lit.Keyr)
		for _, f := range j.Sel.Fields {
			var val lit.Val
			var err error
			if f.Exp != nil {
				el, err := p.Eval(j, f.Exp)
				if err != nil {
					return nil, err
				}
				val = el.Val
			} else {
				val, err = rec.Key(f.Key)
				if err != nil {
					return nil, err
				}
			}
			if ok {
				err = z.SetKey(f.Key, val)
			} else {
				err = px.Assign(val)
			}
			if err != nil {
				return nil, err
			}
		}
		res = append(res, px)
	}
	if len(j.Ord) != 0 {
		err := orderResult(res, org, j.Ord)
		if err != nil {
			return nil, err
		}
	}
	if j.Off > 0 {
		if len(res) > int(j.Off) {
			res = res[j.Off:]
		} else {
			res = nil
		}
	}
	if j.Lim > 0 && len(res) > int(j.Lim) {
		res = res[:j.Lim]
	}
	return res, nil
}

func collectCount(p *exp.Prog, j *Job, vals lit.Vals, whr exp.Exp) (*exp.Lit, error) {
	// we can ignore order and selection completely
	var res int64
	if whr == nil {
		res = int64(len(vals))
	} else {
		for _, l := range vals {
			j.Cur = l
			ok, err := filter(p, j, l, whr)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			res++
		}
	}
	if j.Off > 0 {
		if res > j.Off {
			res -= j.Off
		} else {
			res = 0
		}
	}
	if j.Lim > 0 && res > j.Lim {
		res = j.Lim
	}
	return exp.LitVal(lit.Int(res)), nil
}

func filter(p *exp.Prog, env exp.Env, v lit.Val, whr exp.Exp) (bool, error) {
	whr, err := p.Resl(env, whr, typ.Bool)
	if err != nil {
		return false, err
	}
	res, err := p.Eval(env, whr)
	if err != nil {
		return false, err
	}
	b, err := lit.ToBool(res.Val)
	if err != nil {
		return false, err
	}
	return b == true, nil
}

func orderResult(sel, subj lit.Vals, ords []Ord) (res error) {
	sort.Stable(orderer{sel, subj, func(i, j int) bool {
		less, err := orderFunc(sel, subj, i, j, ords)
		if err != nil && res == nil {
			res = err
		}
		return less
	}})
	return res
}

func orderFunc(sel, subj lit.Vals, i, j int, ords []Ord) (bool, error) {
	ord := ords[0]
	list := sel
	if ord.Subj {
		list = subj
	}
	a, err := lit.Select(list[i], ord.Key)
	if err != nil {
		return true, err
	}
	b, err := lit.Select(list[j], ord.Key)
	if err != nil {
		return true, err
	}
	cmp, err := lit.Compare(a, b)
	if err != nil {
		return true, err
	}
	if cmp == 0 && len(ords) > 1 {
		return orderFunc(sel, subj, i, j, ords[1:])
	}
	if ord.Desc {
		return cmp > 0, nil
	}
	return cmp < 0, nil
}

type orderer struct {
	a, b lit.Vals
	less func(i, j int) bool
}

func (o orderer) Len() int { return len(o.a) }
func (o orderer) Swap(i, j int) {
	o.a[i], o.a[j] = o.a[j], o.a[i]
	o.b[i], o.b[j] = o.b[j], o.b[i]
}
func (o orderer) Less(i, j int) bool {
	return o.less(i, j)
}
