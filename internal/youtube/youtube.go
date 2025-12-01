package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"mpy-yt/internal/models"
	"net/http"
	"slices"
	"strings"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
			DisableKeepAlives:  false,
			MaxConnsPerHost:    10,
		},
	}
	itagQualityMap = map[int]string{
		160: "144p", 278: "144p", 330: "144p", 394: "144p", 694: "144p",
		133: "240p", 242: "240p", 331: "240p", 395: "240p", 695: "240p",
		134: "360p", 243: "360p", 332: "360p", 396: "360p", 696: "360p",
		135: "480p", 244: "480p", 333: "480p", 397: "480p", 697: "480p",
		136: "720p", 247: "720p", 298: "720p", 302: "720p", 334: "720p", 398: "720p", 698: "720p",
		137: "1080p", 299: "1080p", 248: "1080p", 303: "1080p", 335: "1080p", 399: "1080p", 699: "1080p",
		264: "1440p", 271: "1440p", 304: "1440p", 308: "1440p", 336: "1440p", 400: "1440p", 700: "1440p",
		266: "2160p", 305: "2160p", 313: "2160p", 315: "2160p", 337: "2160p", 401: "2160p", 701: "2160p",
		138: "4320p", 272: "4320p", 402: "4320p", 571: "4320p",
	}
)

const (
	apiEndpoint      = "https://www.youtube.com/youtubei/v1/player"
	thumbnailBaseUrl = "https://img.youtube.com/vi/"
)

type clientConfig struct {
	Name        string
	Version     string
	Id          string
	DeviceModel string
}

var (
	clientAndroid = clientConfig{Name: "ANDROID", Version: "19.50.42", Id: "3"}
	clientIos     = clientConfig{Name: "IOS", Version: "17.13.3", Id: "5", DeviceModel: "iPhone14,3"}
)

type adaptiveFormat struct {
	Url        string `json:"url"`
	Bitrate    int64  `json:"bitrate"`
	MimeType   string `json:"mimeType"`
	Itag       int    `json:"itag"`
	AudioTrack *struct {
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
				Url    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"thumbnails"`
		} `json:"thumbnail"`
	} `json:"videoDetails"`
	StreamingData *struct {
		AdaptiveFormats []adaptiveFormat `json:"adaptiveFormats"`
	} `json:"streamingData"`
}

func ExtractVideoId(id string) string {
	id = strings.TrimSpace(id)
	if len(id) == 0 {
		return ""
	}

	if len(id) == 11 && isId(id) {
		return id
	}

	if len(id) > 11 && isId(id[:11]) {
		c := id[11]
		if c == '&' || c == '?' {
			return id[:11]
		}
	}

	if idx := strings.Index(id, "v="); idx != -1 {
		sub := id[idx+2:]
		if len(sub) >= 11 && isId(sub[:11]) {
			return sub[:11]
		}
	}
	if idx := strings.Index(id, "youtu.be/"); idx != -1 {
		sub := id[idx+9:]
		if len(sub) >= 11 && isId(sub[:11]) {
			return sub[:11]
		}
	}
	if idx := strings.Index(id, "/shorts/"); idx != -1 {
		sub := id[idx+8:]
		if len(sub) >= 11 && isId(sub[:11]) {
			return sub[:11]
		}
	}
	if idx := strings.Index(id, "/embed/"); idx != -1 {
		sub := id[idx+7:]
		if len(sub) >= 11 && isId(sub[:11]) {
			return sub[:11]
		}
	}
	if idx := strings.Index(id, "/live/"); idx != -1 {
		sub := id[idx+6:]
		if len(sub) >= 11 && isId(sub[:11]) {
			return sub[:11]
		}
	}
	if idx := strings.Index(id, "/v/"); idx != -1 {
		sub := id[idx+3:]
		if len(sub) >= 11 && isId(sub[:11]) {
			return sub[:11]
		}
	}

	return ""
}

func isId(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func GetPlayerData(videoId string) (*models.PlayerData, error) {
	playerData, err := attemptExtraction(videoId, clientAndroid)
	if err != nil {
		lowerErr := strings.ToLower(err.Error())
		if strings.Contains(lowerErr, "login_required") || strings.Contains(lowerErr, "age") {
			return attemptExtraction(videoId, clientIos)
		}
		return nil, err
	}
	return playerData, nil
}

func attemptExtraction(videoId string, cfg clientConfig) (*models.PlayerData, error) {
	var builder strings.Builder
	builder.Grow(384 + len(videoId))

	builder.WriteString(`{"context":{"client":{"clientName":"`)
	builder.WriteString(cfg.Name)
	builder.WriteString(`","clientVersion":"`)
	builder.WriteString(cfg.Version)
	if cfg.DeviceModel != "" {
		builder.WriteString(`","deviceModel":"`)
		builder.WriteString(cfg.DeviceModel)
	}
	builder.WriteString(`","hl":"en","gl":"US"},"user":{"lockedSafetyMode":false}},"videoId":"`)
	builder.WriteString(videoId)
	builder.WriteString(`","contentCheckOk":true,"racyCheckOk":true}`)

	req, _ := http.NewRequest("POST", apiEndpoint, strings.NewReader(builder.String()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Youtube-Client-Name", cfg.Id)
	req.Header.Set("X-Youtube-Client-Version", cfg.Version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api request failed with status code: %d", resp.StatusCode)
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
	thumbs := apiResp.VideoDetails.Thumbnail.Thumbnails
	if len(thumbs) > 0 {
		thumbUrl = thumbs[len(thumbs)-1].Url
	} else {
		thumbUrl = fmt.Sprintf("%s%s/maxresdefault.jpg", thumbnailBaseUrl, videoId)
	}

	return &models.PlayerData{
		Title:        strings.TrimSpace(apiResp.VideoDetails.Title),
		ThumbnailUrl: thumbUrl,
		Videos:       videos,
		Audios:       audios,
	}, nil
}

func parseStreams(formats []adaptiveFormat) ([]models.VideoStream, []models.AudioStream) {
	videoMap := make(map[string]models.VideoStream, len(formats)/2)
	audioMap := make(map[string]models.AudioStream, 6)

	for _, fmt := range formats {
		if fmt.Url == "" || fmt.Bitrate == 0 {
			continue
		}

		if strings.HasPrefix(fmt.MimeType, "video/") {
			quality, ok := itagQualityMap[fmt.Itag]
			if !ok {
				continue
			}
			if existing, exists := videoMap[quality]; !exists || fmt.Bitrate > existing.Bitrate {
				videoMap[quality] = models.VideoStream{
					Stream:  models.Stream{Url: fmt.Url, Bitrate: fmt.Bitrate},
					Quality: quality,
				}
			}
		} else if strings.HasPrefix(fmt.MimeType, "audio/") {
			langCode := "und"
			displayName := "Original"
			isDefault := false

			if fmt.AudioTrack != nil {
				if fmt.AudioTrack.DisplayName != "" {
					displayName = fmt.AudioTrack.DisplayName
				} else {
					displayName = "Unknown"
				}
				if fmt.AudioTrack.Id != "" {
					parts := strings.Split(fmt.AudioTrack.Id, ".")
					if len(parts) > 0 {
						langCode = parts[0]
					}
				}
				isDefault = fmt.AudioTrack.AudioIsDefault
			}

			if existing, exists := audioMap[langCode]; !exists || fmt.Bitrate > existing.Bitrate {
				audioMap[langCode] = models.AudioStream{
					Stream:    models.Stream{Url: fmt.Url, Bitrate: fmt.Bitrate},
					Language:  langCode,
					Name:      displayName,
					IsDefault: isDefault,
				}
			}
		}
	}

	videos := make([]models.VideoStream, 0, len(videoMap))
	for _, v := range videoMap {
		videos = append(videos, v)
	}
	slices.SortFunc(videos, func(a, b models.VideoStream) int {
		return int(b.Bitrate - a.Bitrate)
	})

	audios := make([]models.AudioStream, 0, len(audioMap))
	for _, a := range audioMap {
		audios = append(audios, a)
	}
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
