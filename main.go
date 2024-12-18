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
	Find        string
	Verbose     bool
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

func print(log string, args []interface{}, printFunc func(format string, a ...interface{})) {
	if config.Verbose {
		printFunc(log, args...)
	}
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

func extractLog(commitId string, specificLog ...string) string {
	re := regexp.MustCompile(`(#\d+|^U-\d+|\(.*\) #\d+)`)
	logBytes, _ := exec.Command("git", "show", "--quiet", commitId).Output()
	log := strings.TrimSpace(string(logBytes))

	filteredLog := make([]string, 0)
	for _, logs := range specificLog {
		if len(strings.TrimSpace(logs)) > 0 {
			filteredLog = append(filteredLog, logs)
		}
	}

	if len(log) > 0 {
		logArr := strings.Split(log, "\n")
		for _, logLine := range logArr {
			ll := strings.TrimSpace(logLine)
			if re.Match([]byte(ll)) {
				if len(filteredLog) > 0 {
					for _, sl := range filteredLog {
						findL := regexp.MustCompile(sl)
						if findL.Match([]byte(ll)) {
							return fmt.Sprintf("%s\n", ll)
						}
					}
				} else {
					return fmt.Sprintf("%s\n", ll)
				}
			}
		}
	}

	return ""
}

func captureLogMessages(commitIds []string, specificLog string) {
	collection := make([]string, 0)
	for _, commitId := range commitIds {
		extracted := extractLog(commitId, specificLog)
		if !contains(extracted, collection) {
			collection = append(collection, extracted)
		}
	}

	if !config.Verbose {
		re := regexp.MustCompile(`(#\d+|U-\d+)`)
		for _, line := range collection {
			bb := []byte(line)
			if re.Match(bb) {
				s := string(re.Find(bb))
				fmt.Println(s)
			}
		}
	} else {
		for _, line := range collection {
			color.HiGreen(line)
		}
	}
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

func findLogsByRecentGitTags() {
	print("find by recent git tags", []interface{}{}, color.HiMagenta)
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

	print("commitIds to pick: %v\n", []interface{}{color.HiCyanString("%v", commitIds)}, color.HiMagenta)
	print("tag ids from new to old: %v\n", []interface{}{color.HiCyanString("%v", tagIds)}, color.HiMagenta)

	captureLogMessages(commitIds, config.Find)
}

func findLogsByStartingAndEndingTags() {
	print("find by starting and ending tags", []interface{}{}, color.HiMagenta)
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

	print("commit ids to pick: %v\n", []interface{}{color.HiCyanString("%v", commitIds)}, color.HiMagenta)
	print("tag ids from new to old: %v\n", []interface{}{color.HiCyanString("%v", tagIds)}, color.HiMagenta)

	captureLogMessages(commitIds, config.Find)
}

func findLogsByTagCount() {
	print("find logs by tag count", []interface{}{}, color.HiMagenta)
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

	print("commit ids to pick: %v\n", []interface{}{color.HiCyanString("%v", commitIds)}, color.HiMagenta)
	print("tag ids from new to old: %v\n", []interface{}{color.HiCyanString("%v", tagIds)}, color.HiMagenta)

	captureLogMessages(commitIds, config.Find)
}

func findLogsWithinDeploymentTag() {
	print("find logs within deployment tag", []interface{}{}, color.HiMagenta)
	reader := getGitLogs(config.Path)
	line, err := reader.ReadString('\n')

	count := config.Count
	if count == -1 {
		count = 10 // default find within 10 tags
	}

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
			if contains(config.StartingTag, tagIdsArr) {
				startPicking = true
			}

			if count <= 0 {
				break
			}

			if startPicking {
				tag := tagIdsArr[0]
				tagIds = append(tagIds, tag)
				count--
			}
		}

		if startPicking {
			commitIds = append(commitIds, commitId)
		}

		line, err = reader.ReadString('\n')
	}

	print("commit ids to pick: %v\n", []interface{}{color.HiCyanString("%v", commitIds)}, color.HiMagenta)
	print("tag ids from new to old: %v\n", []interface{}{color.HiCyanString("%v", tagIds)}, color.HiMagenta)

	captureLogMessages(commitIds, config.Find)
}

func main() {
	projectPath := flag.String("path", "", "project path")
	branchInput := flag.String("branch", "", "branch to use")
	startTag := flag.String("start", "", "starting repo git tag")
	endTag := flag.String("end", "", "ending repo git tag")
	count := flag.Int("tag-count", -1, "tag count; e.g. -tag-count 3 ; will include 3 tags counting from the first tag")
	find := flag.String("find", "", "log message to find")
	verbose := flag.Bool("v", false, "verbose log messages, default: false")
	flag.Parse()

	if *projectPath == "" {
		fmt.Println("please provide project path!")
		os.Exit(1)
	}

	config.Path = *projectPath
	config.StartingTag = *startTag
	config.EndingTag = *endTag
	config.Count = *count
	config.Find = *find
	config.Verbose = *verbose
	// recursive function to look for project
	/* if *projectPath != "" {
		repoLocation = *projectPath
	} else {
		currentDir, _ := os.Getwd()
		visited := make(map[string]bool)
		findRepoLocation(currentDir, &repoLocation, visited)
	} */

	_ = os.Chdir(*projectPath)
	gitBranchCmd := exec.Command("git", "branch", "--show-current")
	branch, err := gitBranchCmd.Output()
	if err != nil {
		panic(err)
	}

	print("current branch: %v", []interface{}{color.HiCyanString("%s", branch)}, color.HiMagenta)

	if *branchInput != "" {
		print("switching to branch: %v", []interface{}{color.HiCyanString("%v", *branchInput)}, color.HiMagenta)
		exec.Command("git", "checkout", ".").Run()
		switchBranchCmd := exec.Command("git", "switch", *branchInput)
		err := switchBranchCmd.Run()
		if err != nil {
			print("error: %v", []interface{}{err}, color.HiRed)
			print("branch switch failed, skip...", []interface{}{}, color.HiRed)
		}
		branch, _ = exec.Command("git", "branch", "--show-current").Output()
		print("switched to branch: %s", []interface{}{color.HiCyanString("%s", branch)}, color.HiMagenta)
	}

	exec.Command("git", "fetch", "origin", string(branch)).Run()

	if config.StartingTag == "" && config.EndingTag == "" && config.Count == -1 {
		findLogsByRecentGitTags()
	} else if config.StartingTag != "" && config.EndingTag != "" {
		findLogsByStartingAndEndingTags()
	} else if config.Find != "" && config.StartingTag != "" {
		findLogsWithinDeploymentTag()
	} else {
		findLogsByTagCount()
	}
}
