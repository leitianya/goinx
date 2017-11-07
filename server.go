package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/biezhi/goinx/log"
)

type Server struct {
	Name      string   `yaml:"name"`
	Listen    string   `yaml:"listen"`
	Domains   []string `yaml:"domains"`
	Root      *string  `yaml:"root"`
	ProxyPass *string  `yaml:"proxy_pass"`
	KeyFile   string   `yaml:"key_file"`
	CertFile  string   `yaml:"cert_file"`
}

func (s *Server) Start() {
	if s.ProxyPass != nil {
		log.Info("%s listen %s, proxy to %s", s.Name, s.Listen, *s.ProxyPass)
	} else {
		if s.Root != nil {
			log.Info("%s listen %s, static dir %s", s.Name, s.Listen, *s.Root)
		}
	}

	http.HandleFunc("/", s.Handler)
	err := http.ListenAndServe(s.Listen, nil)
	if err != nil {
		log.Error("%v", err)
	}
}

func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", ProjectName+"/"+Version)
	if !Contains(s.Domains, strings.Replace(r.Host, s.Listen, "", -1)) {
		log.Error("Request [%s] RemoteAddr: %s, Header: %v", r.Host, r.RemoteAddr, r.Header)
		w.Write([]byte("Bad Request."))
		return
	}
	log.Info("Request [%s] RemoteAddr: %s, Header: %v", r.Host, r.RemoteAddr, r.Header)

	if s.ProxyPass == nil {
		s.Static(w, r)
	} else {
		s.Proxy(w, r)
	}

}

// static server
func (s *Server) Static(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir(*s.Root))
}

// proxy server
func (s *Server) Proxy(w http.ResponseWriter, r *http.Request) {
	o := new(http.Request)

	*o = *r
	targetURL, err := url.Parse(*s.ProxyPass)

	o.Host = targetURL.Host
	o.URL.Scheme = targetURL.Scheme
	o.URL.Host = targetURL.Host
	o.URL.Path = singleJoiningSlash(targetURL.Path, o.URL.Path)

	if q := o.URL.RawQuery; q != "" {
		o.URL.RawPath = o.URL.Path + "?" + q
	} else {
		o.URL.RawPath = o.URL.Path
	}

	o.URL.RawQuery = targetURL.RawQuery

	o.Proto = "HTTP/1.1"
	o.ProtoMajor = 1
	o.ProtoMinor = 1
	o.Close = false

	transport := http.DefaultTransport

	res, err := transport.RoundTrip(o)

	if err != nil {
		log.Error("http: proxy error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hdr := w.Header()

	for k, vv := range res.Header {
		for _, v := range vv {
			hdr.Add(k, v)
		}
	}

	// for _, c := range res.SetCookie {
	// w.Header().Add("Set-Cookie", c.Raw)
	// }

	w.WriteHeader(res.StatusCode)

	if res.Body != nil {
		io.Copy(w, res.Body)
	}

}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
