package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	configPath = flag.String("config", "config.json", "path to config.json file with link definitions")
	listenAddr = flag.String("listen", "[::1]:8080", "address to listen on")
	baseURL    = flag.String("base-url", "", "base URL, used by opensearchdescription")
)

const (
	defaultCacheControl = "public, max-age=3600, must-revalidate"
	linkCacheControl    = "public, max-age=600, stale-while-revalidate=2592000"
	faviconCacheControl = "public, max-age=86400, immutable"
)

type (
	Name  = string
	links map[Name]Target
	Links interface {
		Links() *links
	}
)

type Target struct {
	*url.URL
}

func (j *Target) UnmarshalJSON(buf []byte) (err error) {
	var u string
	err = json.Unmarshal(buf, &u)
	if err != nil {
		return err
	}
	j.URL, err = url.Parse(u)
	if err != nil {
		return err
	}
	return nil
}

func (l links) Links() *links { return &l }

// ServeHTTP performs a link redirect for the given request.
//
// All of these are valid:
//
//	/name
//	/name?query=value
//	/name/suffix
//	/name/suffix?query=value
func (l links) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	name, suffix, _ := strings.Cut(r.URL.Path[1:], "/")
	if t, ok := l[name]; ok {
		// http://target/foo -> http://target/foo/suffix?query=value
		tnew := t.JoinPath(suffix)
		tquery := tnew.Query()
		for k, v := range r.URL.Query() {
			tquery.Del(k)
			for _, vv := range v {
				tquery.Add(k, vv)
			}
		}
		tnew.RawQuery = tquery.Encode()
		w.Header().Set("Cache-Control", linkCacheControl)
		http.Redirect(w, r, tnew.String(), http.StatusTemporaryRedirect)
		return
	}
	http.NotFound(w, r)
}

func Config(file string) (links links, err error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &links)
	if err != nil {
		return nil, err
	}
	return links, nil
}

type Reloader struct {
	file          string
	cfg           atomic.Pointer[links]
	checking      sync.Mutex
	mtime         int64
	nextCheck     int64
	checkInterval time.Duration
}

var ErrLocked = errors.New("mutex locked")

func (re *Reloader) reload() error {
	if !re.checking.TryLock() {
		return ErrLocked
	}
	defer re.checking.Unlock()
	if nc := time.Now().Round(re.checkInterval).Unix(); re.nextCheck != nc {
		re.nextCheck = nc
		st, err := os.Stat(re.file)
		if err != nil {
			return err
		}
		if mtime := st.ModTime().Unix(); re.mtime != mtime {
			newcfg, err := Config(re.file)
			if err != nil {
				return err
			}
			re.cfg.Store(&newcfg)
			re.mtime = mtime
		}
	}
	return nil
}

func (re *Reloader) Links() *links {
	if err := re.reload(); err != nil {
		if errors.Is(err, ErrLocked) {
			// silently continue
		} else {
			log.Print(err)
		}
		// continue (non-fatal)
	}
	return re.cfg.Load()
}

func (re *Reloader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	re.Links().ServeHTTP(w, r)
}

func NewReloader(file string, checkInterval time.Duration) (handler *Reloader, err error) {
	var re Reloader
	re.file = file
	re.checkInterval = checkInterval
	if err := re.reload(); err != nil {
		return nil, err
	}
	return &re, nil
}

func listen(addr string) (net.Listener, error) {
	if addr == "-" {
		return net.FileListener(os.NewFile(0, "stdin"))
	}
	if path, ok := strings.CutPrefix(addr, "unix:"); ok {
		return net.Listen("unix", path)
	}
	return net.Listen("tcp", addr)
}

func main() {
	flag.Parse()
	handler, err := NewReloader(*configPath, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	if sock, err := listen(*listenAddr); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Listening on %s", sock.Addr())
		log.Fatal(http.Serve(sock, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.RequestURI)
			w.Header().Set("Cache-Control", defaultCacheControl)
			switch r.URL.Path {
			case "/":
				ServeIndex(w, r, handler)
			case "/opensearch.xml":
				ServeOpenSearchXML(w, r)
			case "/favicon.ico":
				w.Header().Set("Cache-Control", faviconCacheControl)
				http.ServeFileFS(w, r, files, "/favicon.ico")
			default:
				handler.ServeHTTP(w, r)
			}
		})))
	}
}
