package proxy

import (
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

const chunkSize = 10 * 1024 * 1024

var transport = &http.Transport{
	MaxIdleConns:        10,
	IdleConnTimeout:     30 * time.Second,
	DisableCompression:  true,
	ForceAttemptHTTP2:   true,
	MaxIdleConnsPerHost: 5,
}

var client = &http.Client{
	Transport: transport,
	Timeout:   0,
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 64*1024)
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

	go http.Serve(l, s)

	go func() {
		if video != nil {
			warmUp(video.Url)
		}
		if audio != nil {
			warmUp(audio.Url)
		}
	}()

	port := l.Addr().(*net.TCPAddr).Port
	vUrl := fmt.Sprintf("http://127.0.0.1:%d/v", port)
	aUrl := fmt.Sprintf("http://127.0.0.1:%d/a", port)

	return s, vUrl, aUrl, nil
}

func warmUp(url string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	offset := startByte
	for offset < total {
		end := offset + chunkSize
		if end > total {
			end = total
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, stream.Url, nil)
		if err != nil {
			return
		}

		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end-1))

		resp, err := client.Do(req)
		if err != nil {
			return
		}

		bufPtr := bufPool.Get().(*[]byte)
		n, err := io.CopyBuffer(w, resp.Body, *bufPtr)
		bufPool.Put(bufPtr)

		resp.Body.Close()

		if n > 0 {
			offset += n
		}

		if err != nil {
			break
		}

		if offset >= total {
			break
		}
	}
}
