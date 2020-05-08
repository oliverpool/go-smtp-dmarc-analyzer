package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	analyzer "github.com/oliverpool/go-smtp-dmarc-analyzer"

	"github.com/kelseyhightower/envconfig"

	kitlog "github.com/go-kit/kit/log"
)

type forwarder struct {
	Addr     string `required:"true"`
	Username string
	Password string
	From     string   `required:"true"`
	To       []string `required:"true"`
}

type config struct {
	Listener struct {
		Addr   string `default:":25"`
		Domain string `required:"true"`
	} `required:"true"`

	Forwarder forwarder `required:"true"`
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var cfg config
	err := envconfig.Process("", &cfg)
	if err != nil {
		fmt.Println(err)
		envconfig.Usage("", &cfg)
		return err
	}

	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestamp)

	be := &analyzer.Backend{
		Logger:      kitlog.With(logger, "module", "dmarc-checker"),
		FailedEmail: emailForwarder(kitlog.With(logger, "module", "forwarder"), cfg.Forwarder),
	}

	s := smtp.NewServer(be)

	s.Addr = cfg.Listener.Addr
	s.Domain = cfg.Listener.Domain
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AuthDisabled = true

	logger.Log("listening", s.Addr)
	return s.ListenAndServe()
}

func emailForwarder(logger kitlog.Logger, cfg forwarder) func(io.Reader) {
	var auth sasl.Client
	if cfg.Username != "" || cfg.Password != "" {
		auth = sasl.NewPlainClient("", cfg.Username, cfg.Password)
	}
	return func(r io.Reader) {
		err := sendMail(cfg.Addr, auth, cfg.From, cfg.To, r)
		if err != nil && logger != nil {
			logger.Log("err", err)
		}
	}
}

// Function belows are a copy from the github.com/emersion/go-smtp repo
// with a DialTLS
// And a timeout

func dialTLS(addr string, tlsConfig *tls.Config) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}
	host, _, _ := net.SplitHostPort(addr)
	return smtp.NewClient(conn, host)
}

func sendMail(addr string, a sasl.Client, from string, to []string, r io.Reader) error {
	c, err := dialTLS(addr, nil)
	if err != nil {
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err = c.StartTLS(nil); err != nil {
			return err
		}
	}
	if a != nil {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err = c.Auth(a); err != nil {
			return err
		}
	}
	if err = c.Mail(from, nil); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
