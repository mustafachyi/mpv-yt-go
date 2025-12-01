package ui

import (
	"bufio"
	"fmt"
	"mpy-yt/internal/models"
	"mpy-yt/internal/youtube"
	"os"
	"sort"
	"strconv"
	"strings"
)

func GetIdentifierFromInput() string {
	if clip := getClipboard(); clip != "" {
		if youtube.ExtractVideoId(clip) != "" {
			fmt.Printf("Using clipboard: %s\n", clip)
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
		fmt.Println("No video streams available, attempting audio only.")
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
		printHeader(data.Title)
		fmt.Printf("Video Quality: %s\n", video.Quality)
	}

	if audio := selectAudio(data.Audios, langPref); audio != nil {
		return models.VideoSelection{Video: *video, Audio: *audio}
	}
	return nil
}

func printHeader(title string) {
	fmt.Println(title)
	width := len(title)
	if width > 80 {
		width = 80
	}
	fmt.Println(strings.Repeat("─", width))
	fmt.Println()
}

func selectVideo(videos []models.VideoStream, qualityPref string) *models.VideoStream {
	if qualityPref != "" {
		lowerPref := strings.ToLower(qualityPref)
		if lowerPref == "highest" {
			fmt.Printf("Video: Selected 'highest' -> %s\n", videos[0].Quality)
			return &videos[0]
		}
		if lowerPref == "lowest" {
			v := videos[len(videos)-1]
			fmt.Printf("Video: Selected 'lowest' -> %s\n", v.Quality)
			return &v
		}

		for _, v := range videos {
			if strings.EqualFold(v.Quality, qualityPref) {
				fmt.Printf("Video: Matched quality -> %s\n", v.Quality)
				return &v
			}
		}

		reqQuality := parseQuality(qualityPref)
		if reqQuality == -1 {
			fmt.Printf("Video: Could not parse '%s'. Using highest available: %s\n", qualityPref, videos[0].Quality)
			return &videos[0]
		}

		type videoOption struct {
			v    *models.VideoStream
			diff int
		}
		options := make([]videoOption, len(videos))
		for i := range videos {
			q := parseQuality(videos[i].Quality)
			diff := q - reqQuality
			if diff < 0 {
				diff = -diff
			}
			options[i] = videoOption{&videos[i], diff}
		}

		sort.Slice(options, func(i, j int) bool {
			return options[i].diff < options[j].diff
		})

		closest := options[0].v
		fmt.Printf("Video: Quality '%s' not found. Using closest: %s\n", qualityPref, closest.Quality)
		return closest
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
	numStr := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, q)
	if val, err := strconv.Atoi(numStr); err == nil {
		return val
	}
	return -1
}

func selectAudio(audios []models.AudioStream, langPref string) *models.AudioStream {
	if len(audios) == 1 {
		fmt.Printf("Audio: Only one track available -> %s\n", audios[0].Name)
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
			if strings.HasPrefix(strings.ToLower(a.Language), "en") {
				defaultIdx = i
				break
			}
		}
	}

	if langPref != "" {
		for _, a := range audios {
			if strings.EqualFold(a.Language, langPref) {
				fmt.Printf("Audio: Matched language '%s' -> %s\n", langPref, a.Name)
				return &a
			}
		}
		for _, a := range audios {
			if strings.HasPrefix(strings.ToLower(a.Language), strings.ToLower(langPref)) {
				fmt.Printf("Audio: Matched language '%s' -> %s\n", langPref, a.Name)
				return &a
			}
		}
		fmt.Printf("Audio: Language '%s' not found. Using default -> %s\n", langPref, audios[defaultIdx].Name)
		return &audios[defaultIdx]
	}

	fmt.Println("\nAudio Track")
	langCounts := make(map[string]int)
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
