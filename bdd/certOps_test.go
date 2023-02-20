package bdd_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

func TestCertificateEncryptionAndDecryption(t *testing.T) {
	assert.Equal(t, 1, 1)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Mashery API Authentication Backend"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 5),

		KeyUsage:              x509.KeyUsageDataEncipherment,
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	assert.Nil(t, err)

	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	// Read the certificate from file.
	block, _ := pem.Decode(out.Bytes())
	assert.Equal(t, "CERTIFICATE", block.Type)

	readCert, err := x509.ParseCertificate(block.Bytes)
	assert.Nil(t, err)

	// Encrypt piece of text for receipt.

	plainText := "plain-text"
	//
	dat, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, readCert.PublicKey.(*rsa.PublicKey), []byte(plainText), []byte("OAEP Encrypted"))
	assert.Nil(t, err)

	out.Reset()
	pem.Encode(out, &pem.Block{Type: "MASHERY DATA", Bytes: dat, Headers: map[string]string{
		"A": "B",
		"C": "D",
	}})
	//fmt.Println(out)

	str := base64.StdEncoding.EncodeToString(dat)
	//fmt.Println(str)

	// Decode back
	recData, err := base64.StdEncoding.DecodeString(str)
	assert.Nil(t, err)

	textOut, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, recData, []byte("OAEP Encrypted"))
	assert.Nil(t, err)

	fmt.Println(string(textOut))
}
