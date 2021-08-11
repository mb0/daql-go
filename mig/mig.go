package mig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"xelf.org/daql/log"
)

// ReadVersion returns a version read from r or and error.
func ReadVersion(r io.Reader) (v Version, err error) {
	err = json.NewDecoder(r).Decode(&v)
	return v, err
}

// WriteTo writes the version to w and returns the written bytes or an error.
func (v Version) WriteTo(w io.Writer) (int64, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(v)
	if err != nil {
		return 0, err
	}
	return b.WriteTo(w)
}

type Vers struct {
	Major, Minor, Patch int
}

func ParseVers(str string) (v Vers, err error) {
	if str[0] == 'v' {
		str = str[1:]
	}
	for i, part := range strings.SplitN(str, ".", 4) {
		num, err := strconv.Atoi(part)
		if err != nil {
			return v, err
		}
		switch i {
		case 0:
			v.Major = num
		case 1:
			v.Minor = num
		case 2:
			v.Patch = num
		default:
			log.Debug("unexpected version rest %s", str)
		}
	}
	return v, nil
}

func (v Vers) String() string { return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch) }
