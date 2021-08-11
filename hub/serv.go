package hub

// A service is a common interface for the last message processor in line.
// It usually is used by wrappers, that handles request parsing and delegate.
type Service interface {
	// Serve handles the message and returns the response or nil, or an error.
	Serve(*Msg) (*Msg, error)
}

// Services maps message subjects to services.
type Services map[string]Service

// Handle calls the service with m's subject or returns an error.
// If the service returns data and c is not nil, a reply is sent to the sender.
func (s Services) Handle(m *Msg) bool {
	f := s[m.Subj]
	if f == nil {
		return false
	}
	res, err := f.Serve(m)
	if err != nil {
		res = m.ReplyErr(err)
	}
	if res != nil {
		m.From.Chan() <- res
	}
	return true
}
