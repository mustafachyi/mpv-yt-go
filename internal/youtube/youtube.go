package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mpy-yt/internal/models"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	videoIdRegex  = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)
	videoUrlRegex = regexp.MustCompile(`(?:v=|youtu\.be\/|\/shorts\/|\/embed\/|\/live\/|\/v\/)([a-zA-Z0-9_-]{11})`)
	httpClient    = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
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
	} `json:"videoDetails"`
	StreamingData *struct {
		AdaptiveFormats []adaptiveFormat `json:"adaptiveFormats"`
	} `json:"streamingData"`
}

func ExtractVideoId(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}
	if videoIdRegex.MatchString(identifier) {
		return identifier
	}
	match := videoUrlRegex.FindStringSubmatch(identifier)
	if len(match) > 1 {
		return match[1]
	}
	if len(identifier) > 11 {
		prefix := identifier[:11]
		next := identifier[11]
		if (next == '&' || next == '?') && videoIdRegex.MatchString(prefix) {
			return prefix
		}
	}
	return ""
}

func GetPlayerData(videoId string) (*models.PlayerData, error) {
	var wg sync.WaitGroup
	wg.Add(2)

	var thumbnailUrl string
	var playerData *models.PlayerData
	var err error

	go func() {
		defer wg.Done()
		thumbnailUrl = getHighestQualityThumbnail(videoId)
	}()

	go func() {
		defer wg.Done()
		playerData, err = attemptExtraction(videoId, clientAndroid)
		if err != nil && (strings.Contains(strings.ToLower(err.Error()), "login_required") || strings.Contains(strings.ToLower(err.Error()), "age")) {
			playerData, err = attemptExtraction(videoId, clientIos)
		}
	}()

	wg.Wait()

	if playerData != nil {
		playerData.ThumbnailUrl = thumbnailUrl
		return playerData, nil
	}
	return nil, err
}

func getHighestQualityThumbnail(videoId string) string {
	maxResUrl := fmt.Sprintf("%s%s/maxresdefault.jpg", thumbnailBaseUrl, videoId)
	resp, err := httpClient.Head(maxResUrl)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return maxResUrl
	}
	return fmt.Sprintf("%s%s/hqdefault.jpg", thumbnailBaseUrl, videoId)
}

func attemptExtraction(videoId string, cfg clientConfig) (*models.PlayerData, error) {
	reqBody := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    cfg.Name,
				"clientVersion": cfg.Version,
				"deviceModel":   cfg.DeviceModel,
				"hl":            "en",
				"gl":            "US",
			},
			"user": map[string]interface{}{
				"lockedSafetyMode": false,
			},
		},
		"videoId":        videoId,
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(jsonBytes))
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

	return &models.PlayerData{
		Title:  strings.TrimSpace(apiResp.VideoDetails.Title),
		Videos: videos,
		Audios: audios,
	}, nil
}

func parseStreams(formats []adaptiveFormat) ([]models.VideoStream, []models.AudioStream) {
	videoMap := make(map[string]models.VideoStream)
	audioMap := make(map[string]models.AudioStream)

	for _, fmt := range formats {
		if fmt.Url == "" || fmt.Bitrate == 0 {
			continue
		}

		if strings.Contains(fmt.MimeType, "video/") {
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
		} else if strings.Contains(fmt.MimeType, "audio/") {
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
	sort.Slice(videos, func(i, j int) bool {
		return videos[i].Bitrate > videos[j].Bitrate
	})

	audios := make([]models.AudioStream, 0, len(audioMap))
	for _, a := range audioMap {
		audios = append(audios, a)
	}
	sort.Slice(audios, func(i, j int) bool {
		if audios[i].IsDefault != audios[j].IsDefault {
			return audios[i].IsDefault
		}
		return audios[i].Bitrate > audios[j].Bitrate
	})

	return videos, audios
}
