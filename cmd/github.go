// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lhopki01/git-mass-sync/pkg/actions"
	"github.com/lhopki01/git-mass-sync/pkg/debug"
	"github.com/mitchellh/colorstring"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type action int

const (
	actionClone action = iota
	actionSync
	actionArchive
	actionCloneArchive
	actionNone
)

type repo struct {
	SSHURL   string `json:"ssh_url"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

// githubCmd represents the base command when called without any subcommands
var githubCmd = &cobra.Command{
	Use:   "github [org] [download dir]",
	Short: "Download all repos in a github org",
	Run: func(cmd *cobra.Command, args []string) {
		runGithub(args)
	},
}

func init() {
	rootCmd.AddCommand(githubCmd)

	githubCmd.Flags().String("include", ".*", "Regex to match repo names against")
	githubCmd.Flags().String("exclude", "^$", "Regex to exclude repo names against")
	githubCmd.Flags().String("archive-dir", "", "Repo to put archived repos in\n(default is .archive in the download dir)")

	err := viper.BindPFlags(githubCmd.Flags())
	if err != nil {
		log.Fatalf("Binding flags failed: %s", err)
	}
	viper.AutomaticEnv()
}

func processFlags(args []string) (string, string, string, *regexp.Regexp, *regexp.Regexp) {
	if len(args) != 2 {
		log.Fatal("Wrong number of arguments")
	}
	org := args[0]
	dir := filepath.Clean(args[1])

	fmt.Println("=============")
	fmt.Printf("Syncing org %s into %s\n", org, dir)

	archiveDir := viper.GetString("archive-dir")
	if archiveDir == "" {
		archiveDir = fmt.Sprintf("%s/.archive", dir)
	} else {
		archiveDir = filepath.Clean(archiveDir)
	}
	fmt.Printf("Archiving repos into %s\n", archiveDir)

	inR := regexp.MustCompile(viper.GetString("include"))
	exR := regexp.MustCompile(viper.GetString("exclude"))

	fmt.Println("=============")

	return dir, archiveDir, org, inR, exR
}

func runGithub(args []string) {
	dir, archiveDir, org, inR, exR := processFlags(args)

	client := &http.Client{}
	repoList := getRepoList(org, client)
	dirList := actions.GetGitDirList(dir)

	reposToSync, reposToClone, reposToArchive := repoActions(repoList, dirList, archiveDir, inR, exR)

	lenSync := len(reposToSync)
	lenClone := len(reposToClone)
	lenArchive := len(reposToArchive)

	fmt.Println("=============")
	colorstring.Printf("[green]%d repos to sync\n", lenSync)
	colorstring.Printf("[cyan]%d repos to clone\n", lenClone)
	colorstring.Printf("[light_magenta]%d repos to archive\n", lenArchive)
	fmt.Println("=============")

	// Order is very important here.  Clone must always come before archive
	failedSyncRepos, warningSyncRepos := actions.SyncRepos(reposToSync, dir)
	failedCloneRepos := actions.CloneRepos(reposToClone, dir)
	failedArchiveRepos := actions.ArchiveRepos(reposToArchive, dir, archiveDir)

	lenSyncWarnings := len(warningSyncRepos)
	lenSyncFailures := len(failedSyncRepos)
	lenCloneFailures := len(failedCloneRepos)
	lenArchiveFailures := len(failedArchiveRepos)

	if lenSyncWarnings > 0 {
		fmt.Println("=============")
		//nolint:errcheck
		colorstring.Println("[yellow]Warnings:")
		for _, s := range warningSyncRepos {
			colorstring.Printf(s)
		}
	}
	if lenSyncFailures > 0 || lenCloneFailures > 0 || lenArchiveFailures > 0 {
		fmt.Println("=============")
		//nolint:errcheck
		colorstring.Println("[red]Errors:")
		for _, s := range failedSyncRepos {
			//nolint:errcheck
			colorstring.Println(s)
		}
		for _, s := range failedCloneRepos {
			//nolint:errcheck
			colorstring.Println(s)
		}
		for _, s := range failedArchiveRepos {
			//nolint:errcheck
			colorstring.Println(s)
		}
	}

	if !viper.GetBool("dry-run") {
		fmt.Println("=============")
		if lenSyncFailures > 0 {
			colorstring.Printf(
				"[red]%d[reset]/[green]%d repos synced\n",
				lenSync-lenSyncFailures,
				lenSync,
			)
		} else if lenSync != 0 {
			colorstring.Printf(
				"[green]%d/%d repos synced\n",
				lenSync-lenSyncFailures,
				lenSync,
			)
		}
		if lenCloneFailures > 0 {
			colorstring.Printf("[red]%d[reset]/[cyan]%d repos cloned\n", lenClone-lenCloneFailures, lenClone)
		} else if lenClone != 0 {
			colorstring.Printf("[cyan]%d/%d repos cloned\n", lenClone-lenCloneFailures, lenClone)

		}
		if lenArchiveFailures > 0 {
			colorstring.Printf("[red]%d[reset]/[light_magenta]%d repos archived\n", lenArchive-lenArchiveFailures, lenArchive)
		} else if lenArchive != 0 {
			colorstring.Printf("[light_magenta]%d/%d repos archived\n", lenArchive-lenArchiveFailures, lenArchive)
		}
	}
}

func repoAction(repo repo, dirList []string) (action, []string) {
	for i, dir := range dirList {
		if dir == repo.Name {
			if repo.Archived {
				dirList = actions.RemoveElementFromSlice(dirList, i)
				return actionArchive, dirList
			}
			dirList = actions.RemoveElementFromSlice(dirList, i)
			return actionSync, dirList
		}
	}
	if !repo.Archived {
		return actionClone, dirList
	} else if repo.Archived {
		return actionCloneArchive, dirList
	}
	return actionNone, dirList
}

func repoActions(repoList []repo, dirList []string, archiveDir string, inR *regexp.Regexp, exR *regexp.Regexp) ([]string, []string, []string) {
	var reposToSync []string
	var reposToClone []string
	var reposToArchive []string

	for _, repo := range repoList {
		if inR.MatchString(repo.Name) && !exR.MatchString(repo.Name) {
			var a action
			a, dirList = repoAction(repo, dirList)
			switch a {
			case actionArchive:
				reposToArchive = append(reposToArchive, repo.Name)
				continue
			case actionSync:
				reposToSync = append(reposToSync, repo.Name)
				continue
			case actionClone:
				reposToClone = append(reposToClone, repo.SSHURL)
				continue
			case actionCloneArchive:
				if _, err := os.Stat(fmt.Sprintf("%s/%s", archiveDir, repo.Name)); os.IsNotExist(err) {
					reposToArchive = append(reposToArchive, repo.Name)
					reposToClone = append(reposToClone, repo.SSHURL)
				}
				continue
			}
		}
	}
	reposToArchive = append(reposToArchive, dirList...)

	return reposToSync, reposToClone, reposToArchive
}

func getRepoList(org string, client HttpClient) []repo {
	fmt.Printf("Getting repo list")

	var repoList []repo
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100", org)
	//url := fmt.Sprintf("https://api.github.com/user/repos?per_page=100")
	token := os.Getenv("GITHUB_TOKEN")
	for url != "" {
		fmt.Printf(".")
		debug.Debug(url)

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
		resp, err := client.Do(req)
		if resp.StatusCode != 200 {
			log.Fatalf("Unknown response %d for request: %s", resp.StatusCode, url)
		}
		if err != nil {
			log.Fatalf("Github api request failed with err: %v", err)
		}

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(resp.Body)
		if err != nil {
			log.Fatalf("Failed to read repose body: %v", resp.Body)
		}

		var repos []repo
		err = json.Unmarshal(buf.Bytes(), &repos)
		if err != nil {
			fmt.Println(buf.String())
			fmt.Println(err)
		}
		repoList = append(repoList, repos...)

		url = getNextPageLink(resp.Header)
	}
	fmt.Println("")
	return repoList
}

func getNextPageLink(headers http.Header) (nextPage string) {
	links, ok := headers["Link"]
	if ok {
		for _, link := range strings.Split(links[0], ",") {
			segments := strings.Split(strings.TrimSpace(link), ";")
			if len(segments) < 2 {
				continue
			}
			if strings.TrimSpace(segments[1]) == `rel="next"` {
				// check we have a real url between <>
				url, err := url.Parse(segments[0][1 : len(segments[0])-1])
				if err != nil {
					continue
				}
				return url.String()
			}
		}
	} else {
		return ""
	}
	return ""
}
