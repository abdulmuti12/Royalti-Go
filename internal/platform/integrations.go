package platform

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

type SMTPMailer struct{ Config Config }

func (m SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	c := m.Config
	if c.MailHost == "" {
		return fmt.Errorf("MAIL_HOST is not configured")
	}
	if _, e := mail.ParseAddress(to); e != nil {
		return e
	}
	if _, e := mail.ParseAddress(c.MailFrom); e != nil {
		return e
	}
	if strings.ContainsAny(subject+to+c.MailFrom, "\r\n") {
		return fmt.Errorf("invalid mail headers")
	}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	var e error
	implicitTLS := c.MailEncryption == "ssl" || c.MailPort == "465"
	if implicitTLS {
		tlsDialer := tls.Dialer{NetDialer: &dialer, Config: &tls.Config{ServerName: c.MailHost, MinVersion: tls.VersionTLS12}}
		conn, e = tlsDialer.DialContext(ctx, "tcp", net.JoinHostPort(c.MailHost, c.MailPort))
	} else {
		conn, e = dialer.DialContext(ctx, "tcp", net.JoinHostPort(c.MailHost, c.MailPort))
	}
	if e != nil {
		return e
	}
	defer conn.Close()
	deadline := time.Now().Add(15 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if e = conn.SetDeadline(deadline); e != nil {
		return e
	}
	client, e := smtp.NewClient(conn, c.MailHost)
	if e != nil {
		return e
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok && !implicitTLS {
		if e = client.StartTLS(&tls.Config{ServerName: c.MailHost, MinVersion: tls.VersionTLS12}); e != nil {
			return e
		}
	} else if c.MailUser != "" && !implicitTLS {
		return fmt.Errorf("SMTP authentication requires STARTTLS")
	}
	if c.MailUser != "" {
		if e = client.Auth(smtp.PlainAuth("", c.MailUser, c.MailPassword, c.MailHost)); e != nil {
			return e
		}
	}
	if e = client.Mail(c.MailFrom); e != nil {
		return e
	}
	if e = client.Rcpt(to); e != nil {
		return e
	}
	w, e := client.Data()
	if e != nil {
		return e
	}
	_, e = io.WriteString(w, "From: "+c.MailFrom+"\r\nTo: "+to+"\r\nSubject: "+subject+"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n"+body)
	closeErr := w.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return client.Quit()
}

type Google struct {
	ClientID, ClientSecret string
	Client                 *http.Client
}

func (g Google) request(ctx context.Context, method, endpoint string, form url.Values) (domain.Record, error) {
	var body io.Reader
	if method == "GET" {
		endpoint += "?" + form.Encode()
	} else {
		body = strings.NewReader(form.Encode())
	}
	req, e := http.NewRequestWithContext(ctx, method, endpoint, body)
	if e != nil {
		return nil, e
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	res, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Google authentication rejected")
	}
	var data domain.Record
	e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&data)
	return data, e
}
func (g Google) Verify(ctx context.Context, idToken, code string) (domain.GoogleIdentity, error) {
	var id domain.GoogleIdentity
	if g.ClientID == "" {
		return id, fmt.Errorf("Google client is not configured")
	}
	if code != "" {
		r, e := g.request(ctx, "POST", "https://oauth2.googleapis.com/token", url.Values{"code": {code}, "client_id": {g.ClientID}, "client_secret": {g.ClientSecret}, "redirect_uri": {"postmessage"}, "grant_type": {"authorization_code"}})
		if e != nil {
			return id, e
		}
		idToken = domain.Text(r["id_token"])
	}
	if idToken == "" {
		return id, fmt.Errorf("missing Google token")
	}
	r, e := g.request(ctx, "GET", "https://oauth2.googleapis.com/tokeninfo", url.Values{"id_token": {idToken}})
	if e != nil {
		return id, e
	}
	iss := domain.Text(r["iss"])
	if domain.Text(r["aud"]) != g.ClientID || (iss != "https://accounts.google.com" && iss != "accounts.google.com") || domain.Text(r["sub"]) == "" || !domain.Bool(r["email_verified"]) || domain.Integer(r["exp"]) <= time.Now().Unix() {
		return id, fmt.Errorf("invalid Google claims")
	}
	id = domain.GoogleIdentity{Subject: domain.Text(r["sub"]), Email: domain.Text(r["email"]), Name: domain.Text(r["name"]), Avatar: domain.Text(r["picture"])}
	return id, nil
}
