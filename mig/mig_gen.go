// generated code

package mig

import (
	_ "embed"

	"time"
)

//go:embed mig.daql
var rawSchema string

func RawSchema() string { return rawSchema }

// Migration contains migration information of a data source.
type Migration struct {
	ID   int64     `json:"id"`
	Vers string    `json:"vers"`
	Date time.Time `json:"date"`
	Note string    `json:"note,omitempty"`
}

// Version contains essential details for a node to derive a new version number.
//
// The name is the node's qualified name, and date is an optional recording time. Vers is version
// string v1.23.4 for known versions or empty. The minor and patch are a lowercase hex sha256 hash
// strings of the node's details and its children.
type Version struct {
	Name  string    `json:"name"`
	Vers  string    `json:"vers"`
	Date  time.Time `json:"date,omitempty"`
	Minor string    `json:"minor"`
	Patch string    `json:"patch"`
}
