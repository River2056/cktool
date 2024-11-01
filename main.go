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

func findLogsByRecentGitTags(repoLocation string) {
	reader := getGitLogs(repoLocation)
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

	// discard endTag commit
	// commitIds = commitIds[:len(commitIds)-1]
	color.HiMagenta("commitIds to pick: %v\n", color.HiCyanString("%v", commitIds))
	color.HiMagenta("tag ids from new to old: %v\n", color.HiCyanString("%v", tagIds))

	captureLogMessages(commitIds)
}

func findLogsByStartingAndEndingTags(repoLocation, startingTag, endingTag string) {
	reader := getGitLogs(repoLocation)
	line, err := reader.ReadString('\n')

	commitIds := make([]string, 0)
	startPicking := false
	for err == nil {
		commitId := strings.Split(line, " ")[0]

		tagCmd := exec.Command("git", "tag", "--points-at", commitId)
		tagBytes, _ := tagCmd.Output()
		tagContent := strings.TrimSpace(string(tagBytes))

		if len(tagContent) > 0 {
			tagIdsArr := strings.Split(tagContent, "\n")
			if contains(startingTag, tagIdsArr) {
				startPicking = true
			} else if contains(endingTag, tagIdsArr) {
				break
			}
		}

		if startPicking {
			commitIds = append(commitIds, commitId)
		}

		line, err = reader.ReadString('\n')
	}

	color.HiMagenta("commit ids to pick: %v\n", color.HiGreenString("%v", commitIds))
	color.HiMagenta("tag ids from new to old: %v, %v\n", color.HiGreenString("%v", startingTag), color.HiGreenString("%v", endingTag))

	captureLogMessages(commitIds)
}

func main() {
	// logCmd := "git log --oneline"
	// mainCmd := "git log --simplify-by-decoration --decorate=short --oneline %v..%v"
	// git log --tags --simplify-by-decoration --decorate=full
	// git log --oneline | awk '{print $1}'
	// fetchLogCmd := exec.Command("git", "log", "--oneline", "--tags", "--simplify-by-decoration", "--decorate=full", fmt.Sprintf("%s..%s", startTag, endTag))

	projectPath := flag.String("path", "", "project path")
	startTag := flag.String("start", "", "starting repo git tag")
	endTag := flag.String("end", "", "ending repo git tag")
	flag.Parse()

	repoLocation := ""

	if *projectPath == "" {
		fmt.Println("please provide project path!")
		os.Exit(1)
	}

	repoLocation = *projectPath
	startingTag := *startTag
	endingTag := *endTag
	// recursive function to look for project
	/* if *projectPath != "" {
		repoLocation = *projectPath
	} else {
		currentDir, _ := os.Getwd()
		visited := make(map[string]bool)
		findRepoLocation(currentDir, &repoLocation, visited)
	} */

	if startingTag == "" && endingTag == "" {
		findLogsByRecentGitTags(repoLocation)
	} else {
		findLogsByStartingAndEndingTags(repoLocation, startingTag, endingTag)
	}
}
