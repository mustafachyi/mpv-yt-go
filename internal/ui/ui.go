package ui

import (
	"bufio"
	"fmt"
	"mpy-yt/internal/models"
	"mpy-yt/internal/youtube"
	"os"
	"strconv"
	"strings"
)

var stdin = bufio.NewScanner(os.Stdin)

func GetIdentifierFromInput() string {
	if clip := getClipboard(); clip != "" && len(clip) < 2048 {
		if youtube.ExtractVideoId(clip) != "" {
			return clip
		}
	}
	fmt.Print("Enter YouTube URL or Video ID: ")
	if stdin.Scan() {
		return strings.TrimSpace(stdin.Text())
	}
	return ""
}

func GetStreamSelection(data *models.PlayerData, qualityPref, langPref string, audioOnly bool) (*models.VideoStream, *models.AudioStream) {
	if audioOnly {
		return nil, selectAudio(data.Audios, langPref)
	}
	if len(data.Videos) == 0 {
		return nil, selectAudio(data.Audios, langPref)
	}
	video := selectVideo(data.Videos, qualityPref)
	if video == nil {
		return nil, nil
	}
	if qualityPref == "" && langPref == "" {
		fmt.Print("\033[H\033[2J")
		fmt.Println(data.Title)
		fmt.Println()
		fmt.Printf("Video Quality: %s\n", video.Quality)
	}
	return video, selectAudio(data.Audios, langPref)
}

func selectVideo(videos []models.VideoStream, qualityPref string) *models.VideoStream {
	if qualityPref != "" {
		if strings.EqualFold(qualityPref, "highest") {
			return &videos[0]
		}
		if strings.EqualFold(qualityPref, "lowest") {
			return &videos[len(videos)-1]
		}
		for i := range videos {
			if strings.EqualFold(videos[i].Quality, qualityPref) {
				return &videos[i]
			}
		}
		reqQuality := parseQuality(qualityPref)
		if reqQuality == -1 {
			return &videos[0]
		}
		bestIdx, minDiff := 0, 1<<30
		for i := range videos {
			q := parseQuality(videos[i].Quality)
			diff := q - reqQuality
			if diff < 0 {
				diff = -diff
			}
			if diff < minDiff {
				minDiff = diff
				bestIdx = i
			}
		}
		return &videos[bestIdx]
	}
	fmt.Println("Video Quality")
	for i, v := range videos {
		fmt.Printf("  %d) %s\n", i+1, v.Quality)
	}
	fmt.Print("> Select video [1]: ")
	if stdin.Scan() {
		line := strings.TrimSpace(stdin.Text())
		if line == "" {
			return &videos[0]
		}
		if choice, err := strconv.Atoi(line); err == nil && choice >= 1 && choice <= len(videos) {
			return &videos[choice-1]
		}
	}
	os.Stderr.WriteString("Invalid selection.\n")
	return nil
}

func parseQuality(q string) int {
	v := 0
	hasDigit := false
	for i := 0; i < len(q); i++ {
		b := q[i]
		if b >= '0' && b <= '9' {
			v = v*10 + int(b-'0')
			hasDigit = true
		}
	}
	if !hasDigit {
		return -1
	}
	return v
}

func selectAudio(audios []models.AudioStream, langPref string) *models.AudioStream {
	if len(audios) == 1 {
		return &audios[0]
	}
	defaultIdx := 0
	for i := range audios {
		if audios[i].IsDefault {
			defaultIdx = i
			break
		}
		lang := audios[i].Language
		if len(lang) >= 2 && (lang[0]|32) == 'e' && (lang[1]|32) == 'n' {
			defaultIdx = i
		}
	}
	if langPref != "" {
		for i := range audios {
			if strings.EqualFold(audios[i].Language, langPref) {
				return &audios[i]
			}
		}
		prefLen := len(langPref)
		for i := range audios {
			if len(audios[i].Language) >= prefLen && strings.EqualFold(audios[i].Language[:prefLen], langPref) {
				return &audios[i]
			}
		}
		return &audios[defaultIdx]
	}
	fmt.Println("\nAudio Track")
	langCounts := make(map[string]int, len(audios))
	for i := range audios {
		langCounts[audios[i].Name]++
	}
	for i := range audios {
		indicator := ""
		if i == defaultIdx {
			indicator = " (default)"
		}
		displayName := audios[i].Name
		if langCounts[displayName] > 1 {
			displayName = audios[i].Name + " (" + audios[i].Language + ")"
		}
		fmt.Printf("  %d) %s%s\n", i+1, displayName, indicator)
	}
	fmt.Printf("> Select audio [%d]: ", defaultIdx+1)
	if stdin.Scan() {
		line := strings.TrimSpace(stdin.Text())
		if line == "" {
			return &audios[defaultIdx]
		}
		if choice, err := strconv.Atoi(line); err == nil && choice >= 1 && choice <= len(audios) {
			return &audios[choice-1]
		}
	}
	os.Stderr.WriteString("Invalid selection.\n")
	return nil
}
