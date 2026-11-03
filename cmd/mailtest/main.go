package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"time"
)

func main() {
	host := "srv161.niagahoster.com"
	port := "587"
	user := "support@theluxuryasialiving.com"
	pass := "deFaris251123@"
	from := "support@theluxuryasialiving.com"
	to := "abdulmffmuti@gmail.com"

	// Force IPv4 to avoid IPv6 connection refused
	addr := net.JoinHostPort("185.229.118.151", port)
	fmt.Printf("Connecting to %s (%s)...\n", addr, host)

	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.Dial("tcp4", addr)
	if err != nil {
		log.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()
	fmt.Println("TCP connected OK")

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Fatalf("smtp.NewClient: %v", err)
	}
	defer client.Close()
	fmt.Println("SMTP banner received")

	// STARTTLS for port 587
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			log.Fatalf("STARTTLS: %v", err)
		}
		fmt.Println("STARTTLS OK")
	}

	// Auth
	if err = client.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
		log.Fatalf("auth: %v", err)
	}
	fmt.Println("Auth OK")

	// Mail
	if err = client.Mail(from); err != nil {
		log.Fatalf("mail: %v", err)
	}
	if err = client.Rcpt(to); err != nil {
		log.Fatalf("rcpt: %v", err)
	}
	fmt.Println("Recipients OK")

	w, err := client.Data()
	if err != nil {
		log.Fatalf("data: %v", err)
	}
	subject := "Test Email dari Royalti-Go (srv161.niagahoster.com:587)"
	body := "Ini email test dari Royalti-Go API.\n\nKirim oleh: " + user + "\nTimestamp: " + time.Now().Format(time.RFC3339)
	content := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, to, subject, body)
	if _, err = w.Write([]byte(content)); err != nil {
		log.Fatalf("write: %v", err)
	}
	if err = w.Close(); err != nil {
		log.Fatalf("close: %v", err)
	}
	if err = client.Quit(); err != nil {
		log.Fatalf("quit: %v", err)
	}

	fmt.Printf("SUCCESS: email sent to %s\n", to)
}
