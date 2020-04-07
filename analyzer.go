package analyzer

import (
	"bytes"
	"errors"
	"io"

	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-smtp"
	kitlog "github.com/go-kit/kit/log"
	report "github.com/oliverpool/go-dmarc-report"
)

// The Backend implements SMTP server methods.
type Backend struct {
	Logger      kitlog.Logger
	FailedEmail func(io.Reader)
}

// Login rejects all attempts
func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return nil, smtp.ErrAuthUnsupported
}

// AnonymousLogin accept all anonymous login
func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &session{backend: bkd}, nil
}

type session struct {
	backend *Backend
	from    string
}

// Reset discards currently processed message.
func (s *session) Reset() {
	s.from = ""
}

func (s *session) Mail(from string, opts smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *session) Rcpt(to string) error {
	return nil
}

func (s *session) Data(r io.Reader) error {
	buf := &bytes.Buffer{}
	tr := io.TeeReader(r, buf)

	err := parseReport(tr, kitlog.With(s.backend.Logger, "from", s.from))

	if err != nil && s.backend.FailedEmail != nil {
		s.backend.FailedEmail(io.MultiReader(buf, r))
		return nil
	}

	return nil
}

func parseReport(r io.Reader, logger kitlog.Logger) error {
	name, r, err := firstAttachment(r)
	if err != nil {
		if logger != nil {
			logger.Log("step", "firstAttachment", "err", err)
		}
		return err
	}

	agg, err := report.DecodeFile(name, r)
	if err != nil {
		if logger != nil {
			logger.Log("step", "DecodeFile", "name", name, "err", err)
		}
		return err
	}

	if logger != nil {
		for _, r := range agg.Records {
			err := r.Err()
			if err == nil {
				logger.Log("org", agg.Metadata.OrgName, "source_ip", r.Row.SourceIP, "success", r.Row.Count)
			} else {
				logger.Log("org", agg.Metadata.OrgName, "source_ip", r.Row.SourceIP, "failure", r.Row.Count, "err", err)
			}
		}
	}

	return agg.Err()
}

func firstAttachment(r io.Reader) (string, io.Reader, error) {
	mr, err := mail.CreateReader(r)
	if err != nil {
		return "", nil, err
	}

	// Read each mail's part
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", nil, err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			_, params, err := h.ContentDisposition()
			if params["filename"] != "" && err == nil {
				filename := params["filename"]
				return filename, p.Body, nil
			}
		case *mail.AttachmentHeader:
			filename, err := h.Filename()
			if err == nil {
				return filename, p.Body, nil
			}
		}
	}

	return "", nil, errors.New("no attachement found")
}

func (s *session) Logout() error {
	return nil
}
