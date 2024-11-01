package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

type Config struct {
	Path        string
	StartingTag string
	EndingTag   string
	Count       int
}

var config *Config = &Config{}

func contains(s string, sarr []string) bool {
	for _, v := range sarr {
		if s == v {
			return true
		}
	}

	return false
}

// function to recursively look for project path
// currently not used, might take a long time, use with caution
func findRepoLocation(dirRootPath string, repo *string, visited map[string]bool) {
	if _, ok := visited[dirRootPath]; ok {
		return
	}

	if *repo != "" {
		return
	}

	_ = filepath.WalkDir(dirRootPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && d.Name() == "<your-repository-name-here>" {
			fmt.Printf("path: %v\n", path)
			fmt.Printf("name: %v\n", d.Name())

			dirs, _ := os.ReadDir(path)
			for _, dir := range dirs {
				if dir.Name() == ".git" {
					*repo = path
					break
				}
			}
		}
		return nil
	})

	visited[dirRootPath] = true

	absDirRootPath, _ := filepath.Abs(dirRootPath)
	_ = os.Chdir(absDirRootPath)
	_ = os.Chdir("..")
	previousDir, _ := os.Getwd()
	fmt.Printf("previousDir: %v\n", previousDir)
	findRepoLocation(previousDir, repo, visited)
}

func getGitLogs(repoLocation string) *bufio.Reader {
	_ = os.Chdir(repoLocation)

	fetchLogCmd := exec.Command("git", "log", "--oneline")
	pipe, err := fetchLogCmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err = fetchLogCmd.Start(); err != nil {
		panic(err)
	}

	return bufio.NewReader(pipe)
}

func captureLogMessages(commitIds []string) {
	builder := make([]string, 0)
	re, _ := regexp.Compile(`^#\d+|^U-\d+|\(.*\) #\d+`)
	for _, commitId := range commitIds {
		logBytes, _ := exec.Command("git", "show", "--quiet", commitId).Output()
		log := strings.TrimSpace(string(logBytes))

		if len(log) > 0 {
			logArr := strings.Split(log, "\n")
			for _, logLine := range logArr {
				ll := strings.TrimSpace(logLine)
				if re.Match([]byte(ll)) && !contains(ll, builder) {
					builder = append(builder, ll)
				}
			}
		}
	}

	color.HiGreen(strings.Join(builder, "\n"))
}

func sortTags(tags []string) {
	sort.Slice(tags, func(i, j int) bool {
		idArrA := strings.Split(tags[i], "-")
		idArrB := strings.Split(tags[j], "-")
		timestampA, _ := strconv.Atoi(idArrA[len(idArrA)-1])
		timestampB, _ := strconv.Atoi(idArrB[len(idArrB)-1])
		return timestampA > timestampB
	})
}

func findLogsByRecentGitTags(config *Config) {
	reader := getGitLogs(config.Path)
	line, err := reader.ReadString('\n')

	tagIds := make([]string, 0)
	commitIds := make([]string, 0)
	startPicking := false
	for err == nil {
		commitId := strings.Split(line, " ")[0]

		tagCmd := exec.Command("git", "tag", "--points-at", commitId)
		tagBytes, _ := tagCmd.Output()
		tagContent := strings.TrimSpace(string(tagBytes))

		if len(tagContent) > 0 {
			tagIdsArr := strings.Split(tagContent, "\n")
			if len(tagIdsArr) >= 2 {
				sortTags(tagIdsArr)
			}

			tagIdFirst := tagIdsArr[0]
			if !contains(tagIdFirst, tagIds) {
				tagIds = append(tagIds, tagIdFirst)
				startPicking = true
			}
		}

		if len(tagIds) > 1 {
			break
		}

		if startPicking {
			commitIds = append(commitIds, commitId)
		}

		line, err = reader.ReadString('\n')
	}

	color.HiMagenta("commitIds to pick: %v\n", color.HiCyanString("%v", commitIds))
	color.HiMagenta("tag ids from new to old: %v\n", color.HiCyanString("%v", tagIds))

	captureLogMessages(commitIds)
}

func findLogsByStartingAndEndingTags(config *Config) {
	reader := getGitLogs(config.Path)
	line, err := reader.ReadString('\n')

	count := config.Count // default will be -1
	tagIds := make([]string, 0)
	commitIds := make([]string, 0)
	startPicking := false
	for err == nil {
		commitId := strings.Split(line, " ")[0]

		tagCmd := exec.Command("git", "tag", "--points-at", commitId)
		tagBytes, _ := tagCmd.Output()
		tagContent := strings.TrimSpace(string(tagBytes))

		if len(tagContent) > 0 {
			tagIdsArr := strings.Split(tagContent, "\n")
			if len(tagIdsArr) > 1 {
				sortTags(tagIdsArr)
			}
			tag := tagIdsArr[0]
			if contains(config.StartingTag, tagIdsArr) {
				startPicking = true
				tag = config.StartingTag
			} else if contains(config.EndingTag, tagIdsArr) || count == 0 {
				break
			}

			tagIds = append(tagIds, tag)
			count--
		}

		if startPicking && count != 0 {
			commitIds = append(commitIds, commitId)
		}

		line, err = reader.ReadString('\n')
	}

	color.HiMagenta("commit ids to pick: %v\n", color.HiCyanString("%v", commitIds))
	color.HiMagenta("tag ids from new to old: %v\n", color.HiCyanString("%v", tagIds))

	captureLogMessages(commitIds)
}

func findLogsByTagCount(config *Config) {
	reader := getGitLogs(config.Path)
	line, err := reader.ReadString('\n')

	count := config.Count
	tagIds := make([]string, 0)
	commitIds := make([]string, 0)
	startPicking := false
	for err == nil {
		commitId := strings.Split(line, " ")[0]

		tagCmd := exec.Command("git", "tag", "--points-at", commitId)
		tagBytes, _ := tagCmd.Output()
		tagContent := strings.TrimSpace(string(tagBytes))

		if len(tagContent) > 0 {
			startPicking = true
			tagIdsArr := strings.Split(tagContent, "\n")
			if len(tagIdsArr) > 1 {
				sortTags(tagIdsArr)
			}
			tag := tagIdsArr[0]
			tagIds = append(tagIds, tag)
			count--

			if count == 0 {
				break
			}
		}

		if startPicking {
			commitIds = append(commitIds, commitId)
		}

		line, err = reader.ReadString('\n')
	}

	color.HiMagenta("commit ids to pick: %v\n", color.HiCyanString("%v", commitIds))
	color.HiMagenta("tag ids from new to old: %v\n", color.HiCyanString("%v", tagIds))

	captureLogMessages(commitIds)
}

func main() {
	projectPath := flag.String("path", "", "project path")
	startTag := flag.String("start", "", "starting repo git tag")
	endTag := flag.String("end", "", "ending repo git tag")
	count := flag.Int("tag-count", -1, "tag count; e.g. -tag-count 3 ; will include 3 tags counting from the first tag")
	flag.Parse()

	if *projectPath == "" {
		fmt.Println("please provide project path!")
		os.Exit(1)
	}

	config.Path = *projectPath
	config.StartingTag = *startTag
	config.EndingTag = *endTag
	config.Count = *count
	// recursive function to look for project
	/* if *projectPath != "" {
		repoLocation = *projectPath
	} else {
		currentDir, _ := os.Getwd()
		visited := make(map[string]bool)
		findRepoLocation(currentDir, &repoLocation, visited)
	} */

	if config.StartingTag == "" && config.EndingTag == "" && config.Count == -1 {
		findLogsByRecentGitTags(config)
	} else if config.StartingTag != "" && config.EndingTag != "" {
		findLogsByStartingAndEndingTags(config)
	} else {
		findLogsByTagCount(config)
	}
}
