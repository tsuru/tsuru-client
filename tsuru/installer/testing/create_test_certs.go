// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installertest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func GenerateKeyPair() (privateKey *rsa.PrivateKey, err error) {
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	return
}

func CertTemplateGenerator() (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"tsuru Inc."}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(3650 * 24 * time.Hour),
		BasicConstraintsValid: true,
	}
	return &tmpl, nil
}

func CreateCert(template, parent *x509.Certificate, pub interface{}, parentPrivateKey interface{}) (cert *x509.Certificate, certPEM []byte, err error) {
	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPrivateKey)
	if err != nil {
		return
	}
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	return
}

type CertsPath struct {
	ClientCert string
	ClientKey  string
	ServerCert string
	ServerKey  string
	RootKey    string
	RootCert   string
	RootDir    string
}

func CreateTestCerts() (CertsPath, error) {
	var path CertsPath
	rootKey, err := GenerateKeyPair()
	if err != nil {
		return path, err
	}
	rootCertTmpl, err := CertTemplateGenerator()
	if err != nil {
		return path, err
	}
	rootCertTmpl.IsCA = true
	rootCertTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	rootCertTmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	rootCert, rootCertPEM, err := CreateCert(rootCertTmpl, rootCertTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		return path, err
	}
	rootKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rootKey),
	})
	serverKey, err := GenerateKeyPair()
	if err != nil {
		return path, err
	}
	serverCertTmpl, err := CertTemplateGenerator()
	if err != nil {
		return path, err
	}
	serverCertTmpl.KeyUsage = x509.KeyUsageDigitalSignature
	serverCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	serverCertTmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	_, serverCertPEM, err := CreateCert(serverCertTmpl, rootCert, &serverKey.PublicKey, rootKey)
	if err != nil {
		return path, err
	}
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
	})
	clientKey, err := GenerateKeyPair()
	if err != nil {
		return path, err
	}
	clientCertTmpl, err := CertTemplateGenerator()
	if err != nil {
		return path, err
	}
	clientCertTmpl.KeyUsage = x509.KeyUsageDigitalSignature
	clientCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	_, clientCertPEM, err := CreateCert(clientCertTmpl, rootCert, &clientKey.PublicKey, rootKey)
	if err != nil {
		return path, err
	}
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey),
	})
	absPath, err := ioutil.TempDir("", "installer_test")
	if err != nil {
		return path, err
	}
	err = os.Mkdir(absPath+"/certs", 0700)
	if err != nil {
		return path, err
	}
	absPath = absPath + "/certs"
	path = CertsPath{
		RootDir:    absPath,
		RootCert:   filepath.Join(absPath, "ca.pem"),
		RootKey:    filepath.Join(absPath, "ca-key.pem"),
		ServerCert: filepath.Join(absPath, "server-cert.pem"),
		ServerKey:  filepath.Join(absPath, "server-key.pem"),
		ClientCert: filepath.Join(absPath, "cert.pem"),
		ClientKey:  filepath.Join(absPath, "key.pem"),
	}
	err = ioutil.WriteFile(path.RootCert, rootCertPEM, 0644)
	if err != nil {
		return path, err
	}
	err = ioutil.WriteFile(path.RootKey, rootKeyPEM, 0644)
	if err != nil {
		return path, err
	}
	err = ioutil.WriteFile(path.ServerCert, serverCertPEM, 0644)
	if err != nil {
		return path, err
	}
	err = ioutil.WriteFile(path.ServerKey, serverKeyPEM, 0644)
	if err != nil {
		return path, err
	}
	err = ioutil.WriteFile(path.ClientCert, clientCertPEM, 0644)
	if err != nil {
		return path, err
	}
	err = ioutil.WriteFile(path.ClientKey, clientKeyPEM, 0644)
	return path, err
}

func CleanCerts(path string) error {
	return os.Remove(path)
}
