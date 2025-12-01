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

func GetIdentifierFromInput() string {
	if clip := getClipboard(); clip != "" {
		if youtube.ExtractVideoId(clip) != "" {
			return clip
		}
	}

	fmt.Print("Enter YouTube URL or Video ID: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func GetStreamSelection(data *models.PlayerData, qualityPref, langPref string, audioOnly bool) models.StreamSelection {
	if audioOnly {
		if audio := selectAudio(data.Audios, langPref); audio != nil {
			return models.AudioSelection{Audio: *audio}
		}
		return nil
	}

	if len(data.Videos) == 0 {
		if audio := selectAudio(data.Audios, langPref); audio != nil {
			return models.AudioSelection{Audio: *audio}
		}
		return nil
	}

	video := selectVideo(data.Videos, qualityPref)
	if video == nil {
		return nil
	}

	if qualityPref == "" && langPref == "" {
		fmt.Print("\033[H\033[2J")
		fmt.Println(data.Title)
		fmt.Println()
		fmt.Printf("Video Quality: %s\n", video.Quality)
	}

	if audio := selectAudio(data.Audios, langPref); audio != nil {
		return models.VideoSelection{Video: *video, Audio: *audio}
	}
	return nil
}

func selectVideo(videos []models.VideoStream, qualityPref string) *models.VideoStream {
	if qualityPref != "" {
		if strings.EqualFold(qualityPref, "highest") {
			return &videos[0]
		}
		if strings.EqualFold(qualityPref, "lowest") {
			v := videos[len(videos)-1]
			return &v
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

		bestIdx := 0
		minDiff := int(^uint(0) >> 1)

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

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			return &videos[0]
		}
		if choice, err := strconv.Atoi(line); err == nil && choice >= 1 && choice <= len(videos) {
			return &videos[choice-1]
		}
	}
	fmt.Fprintln(os.Stderr, "Invalid selection.")
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
	foundDefault := false
	for i, a := range audios {
		if a.IsDefault {
			defaultIdx = i
			foundDefault = true
			break
		}
	}
	if !foundDefault {
		for i, a := range audios {
			if len(a.Language) >= 2 && (a.Language[0] == 'e' || a.Language[0] == 'E') && (a.Language[1] == 'n' || a.Language[1] == 'N') {
				defaultIdx = i
				break
			}
		}
	}

	if langPref != "" {
		for i := range audios {
			if strings.EqualFold(audios[i].Language, langPref) {
				return &audios[i]
			}
		}
		for i := range audios {
			if len(audios[i].Language) >= len(langPref) && strings.EqualFold(audios[i].Language[:len(langPref)], langPref) {
				return &audios[i]
			}
		}
		return &audios[defaultIdx]
	}

	fmt.Println("\nAudio Track")
	langCounts := make(map[string]int, len(audios))
	for _, a := range audios {
		langCounts[a.Name]++
	}

	for i, a := range audios {
		indicator := ""
		if i == defaultIdx {
			indicator = " (default)"
		}
		displayName := a.Name
		if langCounts[displayName] > 1 {
			displayName = fmt.Sprintf("%s (%s)", a.Name, a.Language)
		}
		fmt.Printf("  %d) %s%s\n", i+1, displayName, indicator)
	}
	fmt.Printf("> Select audio [%d]: ", defaultIdx+1)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			return &audios[defaultIdx]
		}
		if choice, err := strconv.Atoi(line); err == nil && choice >= 1 && choice <= len(audios) {
			return &audios[choice-1]
		}
	}
	fmt.Fprintln(os.Stderr, "Invalid selection.")
	return nil
}
