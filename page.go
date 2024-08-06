package chaselink

import "net/http"

type Page struct {
	Request         *http.Request  `json:"-"`
	RequestMethod   string         `json:"request_method"`
	RequestUrl      string         `json:"request_url"`
	RequestProtocol string         `json:"request_protocol"`
	RequestHeader   http.Header    `json:"request_header"`
	RequestCookies  []*http.Cookie `json:"request_cookies"`
	ResponseHeader  http.Header    `json:"response_header"`
	ResponseTrailer http.Header    `json:"response_trailer"`
	ResponseBody    []byte         `json:"response_body"`
	ResponseCookies []*http.Cookie `json:"response_cookies"`
	StatusCode      int            `json:"status_code"`
	StatusMessage   string         `json:"status_message"`
	Proto           string         `json:"proto"`
	TlsVersion      string         `json:"tls_version"`
	CipherSuite     string         `json:"cipher_suite"`
	TlsServerName   string         `json:"tls_server_name"`
	TlsCertificates []string       `json:"tls_certificates"`
	Err             error          `json:"error"`
	LocalAddr       string         `json:"local_addr"`
	RemoteAddr      string         `json:"remote_addr"`
	DnsHosts        []string       `json:"dns_hosts"`
	DnsAddresses    []string       `json:"dns_addresses"`
}
