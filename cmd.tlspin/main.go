package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"
)

//go:embed render.tmpl
var templateFile string

func dumpCertificate(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
	funcMap := template.FuncMap{
		"add": func(a int, b int) int {
			return a + b
		},

		"sub": func(a int, b int) int {
			return a - b
		},

		"hex": func(a []byte) string {
			return hex.EncodeToString(a)
		},

		"sha256": func(a []byte) []byte {
			hash := sha256.New()
			hash.Write(a)
			return hash.Sum(nil)
		},
	}

	if t, err := template.New("renderer").Funcs(funcMap).Parse(templateFile); err != nil {
		fmt.Println(err)
	} else {

		if err := t.Execute(os.Stdout, verifiedChains); err != nil {
			fmt.Println(err)
		}
	}

	return errors.New("connection should not be attempted")
}

func main() {
	cl := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				VerifyPeerCertificate: dumpCertificate,
			},
		},
		Timeout: time.Second * time.Duration(10),
	}

	_, _ = cl.Get("https://api.mashery.com/")
}
