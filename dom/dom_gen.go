// generated code

package dom

import (
	"xelf.org/xelf/lit"
	"xelf.org/xelf/typ"
)

// Bit is a bit set used for a number of field options.
type Bit uint32

const (
	BitOpt Bit = 1 << iota
	BitPK
	BitIdx
	BitUniq
	BitOrdr
	BitAuto
	BitRO
)

// Elem holds additional information for either constants or type parameters.
type Elem struct {
	Name  string    `json:"name,omitempty"`
	Type  typ.Type  `json:"type,omitempty"`
	Val   int64     `json:"val,omitempty"`
	Bits  Bit       `json:"bits,omitempty"`
	Ref   string    `json:"ref,omitempty"`
	Extra *lit.Dict `json:"extra,omitempty"`
}

// Index represents a record model index, mainly used for databases.
type Index struct {
	Name   string   `json:"name,omitempty"`
	Keys   []string `json:"keys"`
	Unique bool     `json:"unique,omitempty"`
}

// Object holds data specific to object types for grouping.
type Object struct {
	Indices []*Index `json:"indices,omitempty"`
	OrderBy []string `json:"orderby,omitempty"`
}

// Model represents either a bits, enum or obj type and has extra domain information.
type Model struct {
	Kind   typ.Type  `json:"kind"`
	Name   string    `json:"name"`
	Schema string    `json:"schema,omitempty"`
	Extra  *lit.Dict `json:"extra,omitempty"`
	Elems  []*Elem   `json:"elems,omitempty"`
	Object *Object   `json:"object,omitempty"`
}

// Schema is a namespace for models.
type Schema struct {
	Name   string    `json:"name"`
	Extra  *lit.Dict `json:"extra,omitempty"`
	Path   string    `json:"path,omitempty"`
	Use    []string  `json:"use,omitempty"`
	Models []*Model  `json:"models"`
}

// Project is a collection of schemas and project specific extra configuration.
//
// The schema definition can either be declared as part of the project file, or included from an
// external schema file. Includes should have syntax to filtering the included schema definition.
//
// Extra setting, usually include, but are not limited to, targets and output paths for code
// generation, paths to look for the project's manifest and history.
type Project struct {
	Name    string    `json:"name,omitempty"`
	Extra   *lit.Dict `json:"extra,omitempty"`
	Schemas []*Schema `json:"schemas"`
}
