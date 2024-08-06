package chaselink

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/smallstep/certinfo"
	"golang.org/x/net/html"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"regexp"
	"strings"
	"time"
)

const defaultLimit = 10

// User is a http(s) user agent used to fetch pages
type User struct {
	Client    *http.Client
	Trace     *httptrace.ClientTrace
	Current   Page
	Pages     []Page
	UserAgent string
	Limit     int
	Timeout   time.Duration
	Callback  ProgressCallback
}

func (u *User) CheckRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// NewUser creates a new user agent
func NewUser(opts ...Option) (*User, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("could not create cookie jar: %w", err)
	}
	u := &User{
		Limit: defaultLimit,
	}
	u.Client = &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return http.DefaultTransport.(*http.Transport).DialContext(ctx, network, addr)
			},
		},
		CheckRedirect: u.CheckRedirect,
		Jar:           jar,
		Timeout:       0,
	}
	u.Trace = &httptrace.ClientTrace{
		DNSDone: u.DnsDone,
	}

	for _, opt := range opts {
		if err := opt(u); err != nil {
			return nil, err
		}
	}

	return u, nil
}

func (u *User) Do(req *http.Request) error {
	ctx := req.Context()
	var cancel context.CancelFunc
	if u.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, u.Timeout)
		defer cancel()
	}
	u.Pages = []Page{}
	var resp *http.Response
RETRIES:
	for {
		req = req.WithContext(httptrace.WithClientTrace(ctx, u.Trace))
		if u.UserAgent != "" {
			req.Header.Set("User-Agent", u.UserAgent)
		}
		u.Current = Page{
			RequestMethod:   req.Method,
			RequestUrl:      req.URL.String(),
			RequestProtocol: req.Proto,
			RequestHeader:   req.Header,
			RequestCookies:  req.Cookies(),
		}
		u.Current.Request = req.Clone(ctx)
		var err error
		resp, err = u.Client.Do(req)
		if err != nil {
			u.Current.Err = err
		} else {
			u.Current.ResponseBody, _ = io.ReadAll(resp.Body)
			u.Current.ResponseHeader = resp.Header.Clone()
			u.Current.ResponseTrailer = resp.Trailer.Clone()
			u.Current.StatusCode = resp.StatusCode
			u.Current.StatusMessage = resp.Status
			u.Current.Proto = resp.Proto
			u.Current.ResponseCookies = resp.Cookies()

			if resp.TLS != nil {
				thisTls := resp.TLS
				u.Current.TlsVersion = tls.VersionName(thisTls.Version)
				u.Current.CipherSuite = tls.CipherSuiteName(thisTls.CipherSuite)
				u.Current.TlsServerName = thisTls.ServerName
				for _, cert := range thisTls.PeerCertificates {
					txt, err := certinfo.CertificateText(cert)
					if err != nil {
						txt = fmt.Sprintf("certificate error: %v", err)
					}
					u.Current.TlsCertificates = append(u.Current.TlsCertificates, txt)
				}
			}
		}
		u.Pages = append(u.Pages, u.Current)
		if u.Callback != nil {
			err = u.Callback(u.Current)
			if err != nil {
				return err
			}
		}
		if u.Limit != 0 && len(u.Pages) >= u.Limit {
			return fmt.Errorf("too many redirects (%d)", len(u.Pages))
		}
		switch resp.StatusCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			loc := resp.Header.Get("Location")
			if loc != "" {
				req, err = http.NewRequest(http.MethodGet, loc, nil)
				if err != nil {
					break RETRIES
				}
				continue RETRIES
			}
		case http.StatusOK:
			contentType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
			if err != nil {
				break RETRIES
			}
			if contentType != "text/html" {
				break RETRIES
			}
			loc := htmlRefresh(u.Current.ResponseBody)
			if loc != "" {
				req, err = http.NewRequest(http.MethodGet, loc, nil)
				if err != nil {
					break RETRIES
				}
				continue RETRIES
			}
			break RETRIES
		default:
			break RETRIES
		}
	}
	return nil
}

func htmlRefresh(body []byte) string {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			return ""
		}
		if tt == html.SelfClosingTagToken {
			tn, _ := tokenizer.TagName()
			if string(tn) == "meta" {
				var httpEquiv, content string
				for {
					key, val, more := tokenizer.TagAttr()
					if string(key) == "http-equiv" {
						httpEquiv = string(val)
					}
					if string(key) == "content" {
						content = string(val)
					}
					if !more {
						break
					}
				}
				if strings.ToLower(httpEquiv) == "refresh" {
					matches := regexp.MustCompile(`^\s*\d+\s*;\s*(\S+)`).FindStringSubmatch(content)
					if matches != nil {
						return matches[1]
					}
				}
			}
		}
	}
}

func (u *User) DnsDone(dd httptrace.DNSDoneInfo) {
	for _, addr := range dd.Addrs {
		u.Current.DnsAddresses = append(u.Current.DnsAddresses, addr.String())
	}
}
