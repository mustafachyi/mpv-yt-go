package proxy

import (
	"context"
	"fmt"
	"io"
	"mpy-yt/internal/models"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	chunkSize   = 10 * 1024 * 1024
	dialTimeout = 10 * time.Second
)

var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 15 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ForceAttemptHTTP2:     true,
	MaxIdleConnsPerHost:   20,
	DisableCompression:    true,
}

var client = &http.Client{
	Transport: transport,
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 128*1024)
		return &b
	},
}

type Server struct {
	listener net.Listener
	video    *models.Stream
	audio    *models.Stream
}

func Start(video, audio *models.Stream) (*Server, string, string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", "", err
	}
	s := &Server{
		listener: l,
		video:    video,
		audio:    audio,
	}
	go func() {
		if video != nil {
			warmUp(video.Url)
		}
		if audio != nil {
			warmUp(audio.Url)
		}
	}()
	go s.serve()
	port := l.Addr().(*net.TCPAddr).Port
	return s, fmt.Sprintf("http://127.0.0.1:%d/v", port), fmt.Sprintf("http://127.0.0.1:%d/a", port), nil
}

func warmUp(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func (s *Server) serve() {
	srv := &http.Server{
		Handler:      s,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  90 * time.Second,
	}
	srv.Serve(s.listener)
}

func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	var stream *models.Stream
	switch r.URL.Path {
	case "/v":
		stream = s.video
	case "/a":
		stream = s.audio
	default:
		http.NotFound(w, r)
		return
	}
	if stream == nil {
		http.NotFound(w, r)
		return
	}
	startByte := int64(0)
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		if strings.HasPrefix(rangeHeader, "bytes=") {
			if ranges := strings.Split(rangeHeader[6:], "-"); len(ranges) > 0 {
				if val, err := strconv.ParseInt(ranges[0], 10, 64); err == nil {
					startByte = val
				}
			}
		}
	}
	total := stream.Size
	if total == 0 {
		total = 100 * 1024 * 1024 * 1024
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(total-startByte, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startByte, total-1, total))
	w.WriteHeader(http.StatusPartialContent)
	s.stream(r.Context(), w, stream.Url, startByte, total)
}

type result struct {
	resp *http.Response
	err  error
	end  int64
}

func (s *Server) stream(ctx context.Context, w io.Writer, url string, start, total int64) {
	offset := start
	nextCh := make(chan result, 1)
	fetch := func(off, size int64) {
		end := off + size
		if end > total {
			end = total
		}
		go func() {
			r, e := fetchChunk(ctx, url, off, end)
			select {
			case nextCh <- result{resp: r, err: e, end: end}:
			case <-ctx.Done():
				if r != nil && r.Body != nil {
					r.Body.Close()
				}
			}
		}()
	}
	fetch(offset, chunkSize)
	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)
	buf := *bufPtr
	for offset < total {
		var res result
		select {
		case res = <-nextCh:
		case <-ctx.Done():
			return
		}
		if res.err != nil {
			return
		}
		if res.end < total {
			fetch(res.end, chunkSize)
		}
		n, err := io.CopyBuffer(w, res.resp.Body, buf)
		res.resp.Body.Close()
		if n > 0 {
			offset += n
		}
		if err != nil {
			return
		}
	}
}

func fetchChunk(ctx context.Context, url string, start, end int64) (*http.Response, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			t := time.NewTimer(500 * time.Millisecond)
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
				return nil, ctx.Err()
			}
			t.Stop()
		}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end-1))
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			lastErr = fmt.Errorf("status: %d", resp.StatusCode)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}
