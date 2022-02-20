package server

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

func ConfigTLS(caCrt, srvCrt, srvKey string) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(srvCrt, srvKey)
	if err != nil {
		return nil, err
	}
	caCertFile, err := ioutil.ReadFile(caCrt)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertFile)
	// Create the TLS Config with the CA pool and enable Client certificate validation
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{certificate},
		ClientCAs:                caCertPool,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}

	return tlsConfig, nil
}
