package log

type TB interface {
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
	Helper()
}

// Test is a logger using the testing package T or B types for logging.
type Test struct {
	TB
	Default
}

func (l *Test) Debug(m string, s ...interface{}) { l.Helper(); l.Logf(line("DEB ", m, s, l.Tags)) }
func (l *Test) Error(m string, s ...interface{}) { l.Helper(); l.Errorf(line("ERR ", m, s, l.Tags)) }
func (l *Test) Crit(m string, s ...interface{})  { l.Helper(); l.Fatalf(line("CRI", m, s, l.Tags)) }
func (l *Test) With(tags ...interface{}) Logger  { return &Test{l.TB, *l.Default.with(tags)} }
