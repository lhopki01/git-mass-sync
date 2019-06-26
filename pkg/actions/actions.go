package actions

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/lhopki01/git-mass-sync/pkg/debug"
	"github.com/mitchellh/colorstring"
	"github.com/remeh/sizedwaitgroup"
	"github.com/schollz/progressbar/v2"
	"github.com/spf13/viper"
)

func SyncRepos(reposToSync []string, dir string) ([]string, []string) {
	if len(reposToSync) == 0 {
		return nil, nil
	}
	verbose := viper.GetBool("verbose")
	dryRun := viper.GetBool("dry-run")

	swg := sizedwaitgroup.New(viper.GetInt("parallelism"))

	failureChannel := make(chan string)
	doneFailure := make(chan bool)
	warningChannel := make(chan string)
	doneWarning := make(chan bool)
	var failures []string
	var warnings []string
	go collectMessages(&failures, failureChannel, doneFailure)
	go collectMessages(&warnings, warningChannel, doneWarning)

	bar := progressbar.NewOptions(
		len(reposToSync),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[green]Syncing repos"),
	)

	if verbose || dryRun {
		//nolint:errcheck
		colorstring.Println("[green]Syncing repos")
	} else {
		err := bar.RenderBlank()
		if err != nil {
			fmt.Printf("Can't render progress bar")
		}
	}

	for _, repo := range reposToSync {
		if dryRun {
			colorstring.Printf("[green]Would sync %s\n", repo)
		} else {
			swg.Add()
			if verbose {
				colorstring.Printf("[green]Syncing %s\n", repo)
			}
			go syncRepo(dir, repo, &swg, failureChannel, warningChannel, bar)
		}
	}

	swg.Wait()
	close(failureChannel)
	close(warningChannel)
	<-doneFailure
	<-doneWarning

	if !verbose || dryRun {
		err := bar.Finish()
		if err != nil {
			fmt.Printf("Can't render progress bar finish")
		}
		println("")
	}

	return failures, warnings
}

func syncRepo(dir string, repo string, swg *sizedwaitgroup.SizedWaitGroup, failureChannel chan string, warningChannel chan string, bar *progressbar.ProgressBar) {

	cmd := exec.Command("hub", "sync")
	cmd.Dir = fmt.Sprintf("%s/%s", dir, repo)
	output, err := cmd.CombinedOutput()
	debug.Debugf("Output of hub sync %s: %s", repo, string(output))
	if strings.Contains(string(output), "warning: ") {
		warningChannel <- fmt.Sprintf("[green]Syncing %s: [yellow]%s", repo, string(output))
	}
	if err != nil {
		failureChannel <- fmt.Sprintf("[green]Syncing %s: [red]%s\n%s", repo, err, string(output))
	}
	if !viper.GetBool("verbose") {
		err := bar.Add(1)
		if err != nil {
			fmt.Printf("Can't add to progress bar")
		}
	}
	swg.Done()
}

func collectMessages(p *[]string, channel chan string, done chan bool) {
	for msg := range channel {
		*p = append(*p, msg)
	}
	done <- true
}

func CloneRepos(reposToClone []string, dir string) []string {
	swg := sizedwaitgroup.New(viper.GetInt("parallelism"))

	failureChannel := make(chan string)
	done := make(chan bool)
	var failures []string
	go collectMessages(&failures, failureChannel, done)

	for _, repo := range reposToClone {
		if viper.GetBool("dry-run") {
			colorstring.Printf("[cyan]Would clone %s\n", repo)
		} else {
			swg.Add()
			colorstring.Printf("[cyan]Cloning %s\n", repo)
			go cloneRepo(dir, repo, &swg, failureChannel)
		}
	}
	swg.Wait()
	close(failureChannel)
	<-done
	return failures
}

func cloneRepo(dir string, repo string, swg *sizedwaitgroup.SizedWaitGroup, failureChannel chan string) {
	defer swg.Done()
	cmd := exec.Command("git", "clone", repo)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	debug.Debugf("Output of git clone %s: %s", repo, output)
	if err != nil {
		failureChannel <- fmt.Sprintf("[cyan]Cloning %s: [red]%s\n%s", repo, err, string(output))
	}
}

func ArchiveRepos(reposToArchive []string, dir string, archiveDir string) []string {
	swg := sizedwaitgroup.New(viper.GetInt("parallelism"))

	failureChannel := make(chan string)
	done := make(chan bool)
	var failures []string
	go collectMessages(&failures, failureChannel, done)

	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		if viper.GetBool("dry-run") {
			fmt.Printf("Would create archive dir %s if not exists\n", archiveDir)
		} else {
			fmt.Printf("Creating archiveDir %s", archiveDir)
			err := os.MkdirAll(archiveDir, 0755)
			if err != nil {
				//nolint:errcheck
				colorstring.Println("[red]Failed to create archive dir")
				os.Exit(1)
			}
		}
	}
	for _, repo := range reposToArchive {
		if viper.GetBool("dry-run") {
			colorstring.Printf("[light_magenta]Would archive %s in %s\n", repo, archiveDir)
		} else {
			swg.Add()
			colorstring.Printf("[light_magenta]Archiving %s in %s\n", repo, archiveDir)
			go func(dir string, repo string, swg *sizedwaitgroup.SizedWaitGroup) {
				defer swg.Done()

				err := os.Rename(
					fmt.Sprintf("%s/%s", dir, repo),
					fmt.Sprintf("%s/%s", archiveDir, repo),
				)
				if err != nil {
					failureChannel <- fmt.Sprintf("[light_magenta]Archiving %s: [red]%s\n", repo, err)
				}
			}(dir, repo, &swg)
		}
	}
	swg.Wait()
	close(failureChannel)
	<-done
	return failures
}

func GetGitDirList(dir string) []string {
	fmt.Printf("Getting existing git directory list")
	var dirList []string
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for i, f := range files {
		if i%100 == 0 {
			fmt.Printf(".")
		}
		if f.IsDir() {
			cmd := exec.Command("git", "rev-parse")
			cmd.Dir = fmt.Sprintf("%s/%s", dir, f.Name())
			err = cmd.Run()
			if err == nil {
				dirList = append(dirList, f.Name())
			} else {
				debug.Debugf("%s is not a git directory", f.Name())
			}
		} else {
			debug.Debugf("%s is not a directory", f.Name())
		}
	}
	fmt.Println("")
	return dirList
}

func RemoveElementFromSlice(s []string, i int) []string {
	// Does not preserve order
	if len(s) <= i {
		return s
	}
	s[i] = s[0]
	return s[1:]
}
