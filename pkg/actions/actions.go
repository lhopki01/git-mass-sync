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

func (repos Repos) SyncRepos(dir string) {
	num := len(repos)
	if num == 0 {
		return
	}

	verbose := viper.GetBool("verbose")
	dryRun := viper.GetBool("dry-run")

	swg := sizedwaitgroup.New(viper.GetInt("parallelism"))

	bar := progressbar.NewOptions(
		num,
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

	for _, repo := range repos {
		if dryRun {
			colorstring.Printf("[green]Would sync %s\n", repo.Name)
		} else {
			swg.Add()
			if verbose {
				//nolint:errcheck
				colorstring.Printf("[green]Syncing %s\n", repo.Name)
			}
			go repo.syncRepo(dir, &swg, bar)
		}
	}

	swg.Wait()

	if !verbose || dryRun {
		err := bar.Finish()
		if err != nil {
			fmt.Printf("Can't render progress bar finish")
		}

		println("")
	}
}

func (repo *Repo) syncRepo(dir string, swg *sizedwaitgroup.SizedWaitGroup, bar *progressbar.ProgressBar) {
	cmd := exec.Command("hub", "sync")
	cmd.Dir = fmt.Sprintf("%s/%s", dir, repo.Name)
	output, err := cmd.CombinedOutput()
	repo.Message = string(output)
	debug.Debugf("Output of hub sync %s: %s", repo.Name, string(output))

	if strings.Contains(string(output), "warning: ") {
		repo.Severity = Warning
	}

	if err != nil {
		repo.Severity = Error
	}

	if !viper.GetBool("verbose") {
		//nolint:gomnd
		err := bar.Add(1)
		if err != nil {
			fmt.Printf("Can't add to progress bar")
		}
	}

	swg.Done()
}

func (repos Repos) CloneRepos(dir string) {
	swg := sizedwaitgroup.New(viper.GetInt("parallelism"))

	for _, repo := range repos {
		if viper.GetBool("dry-run") {
			colorstring.Printf("[cyan]Would clone %s\n", repo.Name)
		} else {
			swg.Add()
			colorstring.Printf("[cyan]Cloning %s\n", repo.Name)
			go repo.cloneRepo(dir, &swg)
		}
	}

	swg.Wait()
}

func (repo *Repo) cloneRepo(dir string, swg *sizedwaitgroup.SizedWaitGroup) {
	defer swg.Done()

	//nolint:gosec
	cmd := exec.Command("git", "clone", repo.SSHURL)

	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	repo.Message = string(output)

	debug.Debugf("Output of git clone %s: %s", repo.Name, output)

	if err != nil {
		repo.Severity = Error
	}
}

func (repos Repos) ArchiveRepos(dir, archiveDir string) {
	swg := sizedwaitgroup.New(viper.GetInt("parallelism"))

	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		if viper.GetBool("dry-run") {
			fmt.Printf("Would create archive dir %s if not exists\n", archiveDir)
		} else {
			fmt.Printf("Creating archiveDir %s\n", archiveDir)
			err := os.MkdirAll(archiveDir, 0755)
			if err != nil {
				//nolint:errcheck
				colorstring.Println("[red]Failed to create archive dir")
				//nolint:gomnd
				os.Exit(1)
			}
		}
	}

	for _, repo := range repos {
		if viper.GetBool("dry-run") {
			colorstring.Printf("[light_magenta]Would archive %s in %s\n", repo.Name, archiveDir)
		} else {
			swg.Add()
			colorstring.Printf("[light_magenta]Archiving %s in %s\n", repo.Name, archiveDir)
			go repo.archiveRepo(dir, archiveDir, &swg)
		}
	}

	swg.Wait()
}

func (repo *Repo) archiveRepo(dir, archiveDir string, swg *sizedwaitgroup.SizedWaitGroup) {
	defer swg.Done()

	err := os.Rename(
		fmt.Sprintf("%s/%s", dir, repo.Name),
		fmt.Sprintf("%s/%s", archiveDir, repo.Name),
	)

	if err != nil {
		repo.Severity = Error
		repo.Message = err.Error()
	}
}

func GetGitDirList(dir string) []string {
	fmt.Printf("Getting existing git directory list")

	var dirList []string

	files, err := ioutil.ReadDir(dir)

	if err != nil {
		log.Fatal(err)
	}

	for i, f := range files {
		if i%100 == 0 && !viper.GetBool("verbose") {
			fmt.Printf(".")
		}

		if f.IsDir() {
			cmd := exec.Command("git", "rev-parse")
			cmd.Dir = fmt.Sprintf("%s/%s", dir, f.Name())
			err = cmd.Run()

			if err == nil {
				dirList = append(dirList, f.Name())
			} else {
				debug.Debugf("\n[%s] is not a git directory", f.Name())
			}
		} else {
			debug.Debugf("\n[%s] is not a directory", f.Name())
		}
	}

	if !viper.GetBool("verbose") {
		fmt.Println("")
	}

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
