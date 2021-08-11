// generated code

package mig

import (
	"time"
)

// Migration contains migration information of a data source.
type Migration struct {
	ID   int64     `json:"id"`
	Vers string    `json:"vers"`
	Date time.Time `json:"date"`
	Note string    `json:"note,omitempty"`
}

// Version contains essential details for a node to derive a new version number.
//
// The name is the node's qualified name, and date is an optional recording time. Vers is a positive
// integer for known versions or zero if unknown. The hash is a lowercase hex string of an sha256
// hash of the node's qualified name and its contents. For models the default string representation
// is used as content, for schemas each model hash and for projects each schema hash.
type Version struct {
	Name  string    `json:"name"`
	Vers  string    `json:"vers"`
	Date  time.Time `json:"date,omitempty"`
	Minor string    `json:"minor"`
	Patch string    `json:"patch"`
}
