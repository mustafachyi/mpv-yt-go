package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"mpy-yt/internal/models"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
		MaxIdleConnsPerHost: 10,
	},
}

var itagQualityMap = [702]string{
	133: "240p", 134: "360p", 135: "480p", 136: "720p", 137: "1080p", 138: "4320p",
	160: "144p", 242: "240p", 243: "360p", 244: "480p", 247: "720p", 248: "1080p",
	264: "1440p", 266: "2160p", 271: "1440p", 272: "4320p", 278: "144p",
	298: "720p", 299: "1080p", 302: "720p", 303: "1080p", 304: "1440p", 305: "2160p",
	308: "1440p", 313: "2160p", 315: "2160p",
	330: "144p", 331: "240p", 332: "360p", 333: "480p", 334: "720p", 335: "1080p", 336: "1440p", 337: "2160p",
	394: "144p", 395: "240p", 396: "360p", 397: "480p", 398: "720p", 399: "1080p", 400: "1440p", 401: "2160p", 402: "4320p",
	571: "4320p",
	694: "144p", 695: "240p", 696: "360p", 697: "480p", 698: "720p", 699: "1080p", 700: "1440p", 701: "2160p",
}

const (
	apiEndpoint      = "https://www.youtube.com/youtubei/v1/player"
	thumbnailBaseUrl = "https://img.youtube.com/vi/"
)

type clientConfig struct {
	name        string
	version     string
	id          string
	deviceModel string
}

var (
	clientAndroid = clientConfig{"ANDROID", "19.50.42", "3", ""}
	clientIos     = clientConfig{"IOS", "21.03.2", "5", "iPhone14,3"}
)

type adaptiveFormat struct {
	Url           string `json:"url"`
	Bitrate       int64  `json:"bitrate"`
	MimeType      string `json:"mimeType"`
	Itag          int    `json:"itag"`
	ContentLength string `json:"contentLength"`
	AudioTrack    *struct {
		DisplayName    string `json:"displayName"`
		Id             string `json:"id"`
		AudioIsDefault bool   `json:"audioIsDefault"`
	} `json:"audioTrack"`
}

type playerApiResponse struct {
	PlayabilityStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"playabilityStatus"`
	VideoDetails struct {
		Title         string `json:"title"`
		IsLiveContent bool   `json:"isLiveContent"`
		Thumbnail     struct {
			Thumbnails []struct {
				Url string `json:"url"`
			} `json:"thumbnails"`
		} `json:"thumbnail"`
	} `json:"videoDetails"`
	StreamingData *struct {
		AdaptiveFormats []adaptiveFormat `json:"adaptiveFormats"`
	} `json:"streamingData"`
}

func ExtractVideoId(input string) string {
	n := len(input)
	if n == 0 {
		return ""
	}

	start, end := 0, n
	for start < end && input[start] <= ' ' {
		start++
	}
	for end > start && input[end-1] <= ' ' {
		end--
	}
	s := input[start:end]
	n = len(s)

	if n < 11 {
		return ""
	}

	if n == 11 && isValidId(s) {
		return s
	}

	if n > 11 && isValidId(s[:11]) {
		c := s[11]
		if c == '&' || c == '?' || c == '/' || c <= ' ' {
			return s[:11]
		}
	}

	patterns := [...]struct {
		prefix string
		offset int
	}{
		{"v=", 2},
		{"youtu.be/", 9},
		{"/shorts/", 8},
		{"/embed/", 7},
		{"/live/", 6},
		{"/v/", 3},
	}

	for _, p := range patterns {
		if idx := strings.Index(s, p.prefix); idx != -1 {
			sub := s[idx+p.offset:]
			if len(sub) >= 11 && isValidId(sub[:11]) {
				return sub[:11]
			}
		}
	}

	return ""
}

func isValidId(s string) bool {
	if len(s) != 11 {
		return false
	}
	for i := 0; i < 11; i++ {
		c := s[i]
		valid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
		if !valid {
			return false
		}
	}
	return true
}

func GetPlayerData(videoId string) (*models.PlayerData, error) {
	data, err := fetchPlayerData(videoId, clientAndroid)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "login_required") || strings.Contains(errLower, "age") {
			return fetchPlayerData(videoId, clientIos)
		}
		return nil, err
	}
	return data, nil
}

func fetchPlayerData(videoId string, cfg clientConfig) (*models.PlayerData, error) {
	var b strings.Builder
	b.Grow(400)

	b.WriteString(`{"context":{"client":{"clientName":"`)
	b.WriteString(cfg.name)
	b.WriteString(`","clientVersion":"`)
	b.WriteString(cfg.version)
	if cfg.deviceModel != "" {
		b.WriteString(`","deviceModel":"`)
		b.WriteString(cfg.deviceModel)
	}
	b.WriteString(`","hl":"en","gl":"US"},"user":{"lockedSafetyMode":false}},"videoId":"`)
	b.WriteString(videoId)
	b.WriteString(`","contentCheckOk":true,"racyCheckOk":true}`)

	req, _ := http.NewRequest(http.MethodPost, apiEndpoint, strings.NewReader(b.String()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Youtube-Client-Name", cfg.id)
	req.Header.Set("X-Youtube-Client-Version", cfg.version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api request failed: %d", resp.StatusCode)
	}

	var apiResp playerApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if apiResp.PlayabilityStatus.Status != "OK" {
		msg := apiResp.PlayabilityStatus.Reason
		if msg == "" {
			msg = apiResp.PlayabilityStatus.Status
		}
		if msg == "" {
			msg = "video is unplayable"
		}
		return nil, errors.New(msg)
	}

	if apiResp.StreamingData == nil || apiResp.VideoDetails.Title == "" {
		return nil, errors.New("incomplete video data received from API")
	}

	if apiResp.VideoDetails.IsLiveContent {
		return nil, errors.New("live streams are not supported")
	}

	videos, audios := parseStreams(apiResp.StreamingData.AdaptiveFormats)
	if len(audios) == 0 {
		return nil, errors.New("no audio streams available for this video")
	}

	thumbUrl := ""
	if thumbs := apiResp.VideoDetails.Thumbnail.Thumbnails; len(thumbs) > 0 {
		thumbUrl = thumbs[len(thumbs)-1].Url
	} else {
		thumbUrl = thumbnailBaseUrl + videoId + "/maxresdefault.jpg"
	}

	return &models.PlayerData{
		Title:        strings.TrimSpace(apiResp.VideoDetails.Title),
		ThumbnailUrl: thumbUrl,
		Videos:       videos,
		Audios:       audios,
	}, nil
}

func parseStreams(formats []adaptiveFormat) ([]models.VideoStream, []models.AudioStream) {
	videos := make([]models.VideoStream, 0, 8)
	audios := make([]models.AudioStream, 0, 6)

	for i := range formats {
		f := &formats[i]
		if f.Url == "" || f.Bitrate == 0 {
			continue
		}

		mime := f.MimeType
		if len(mime) < 6 {
			continue
		}

		size, _ := strconv.ParseInt(f.ContentLength, 10, 64)

		if mime[0] == 'v' && mime[4] == 'o' {
			if f.Itag < 0 || f.Itag >= len(itagQualityMap) {
				continue
			}
			quality := itagQualityMap[f.Itag]
			if quality == "" {
				continue
			}

			found := -1
			for j := range videos {
				if videos[j].Quality == quality {
					found = j
					break
				}
			}

			if found != -1 {
				if f.Bitrate > videos[found].Bitrate {
					videos[found].Url = f.Url
					videos[found].Bitrate = f.Bitrate
					videos[found].Size = size
				}
			} else {
				videos = append(videos, models.VideoStream{
					Stream:  models.Stream{Url: f.Url, Bitrate: f.Bitrate, Size: size},
					Quality: quality,
				})
			}

		} else if mime[0] == 'a' && mime[4] == 'o' {
			langCode := "und"
			displayName := "Original"
			isDefault := false

			if f.AudioTrack != nil {
				if f.AudioTrack.DisplayName != "" {
					displayName = f.AudioTrack.DisplayName
				} else {
					displayName = "Unknown"
				}
				if id := f.AudioTrack.Id; id != "" {
					if dotIdx := strings.IndexByte(id, '.'); dotIdx > 0 {
						langCode = id[:dotIdx]
					} else {
						langCode = id
					}
				}
				isDefault = f.AudioTrack.AudioIsDefault
			}

			found := -1
			for j := range audios {
				if audios[j].Language == langCode {
					found = j
					break
				}
			}

			if found != -1 {
				if f.Bitrate > audios[found].Bitrate {
					audios[found].Url = f.Url
					audios[found].Bitrate = f.Bitrate
					audios[found].Size = size
					audios[found].Name = displayName
					audios[found].IsDefault = isDefault
				}
			} else {
				audios = append(audios, models.AudioStream{
					Stream:    models.Stream{Url: f.Url, Bitrate: f.Bitrate, Size: size},
					Language:  langCode,
					Name:      displayName,
					IsDefault: isDefault,
				})
			}
		}
	}

	slices.SortFunc(videos, func(a, b models.VideoStream) int {
		return int(b.Bitrate - a.Bitrate)
	})

	slices.SortFunc(audios, func(a, b models.AudioStream) int {
		if a.IsDefault != b.IsDefault {
			if a.IsDefault {
				return -1
			}
			return 1
		}
		return int(b.Bitrate - a.Bitrate)
	})

	return videos, audios
}
