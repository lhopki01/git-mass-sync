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

package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/google/go-github/github"
	"github.com/lhopki01/git-mass-sync/actions"
	"github.com/mitchellh/colorstring"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type HTTPClient interface {
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

type idType int

const (
	org idType = iota
	user
	unknown
)

const reposPerPage = 100

// githubCmd represents the base command when called without any subcommands
var githubCmd = &cobra.Command{
	Use:   "github [org|user] [download dir]",
	Short: "Download all repos from a github organization or user",
	//nolint:gomnd
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runGithub(args)
	},
}

func init() {
	rootCmd.AddCommand(githubCmd)

	githubCmd.Flags().String("include", ".*", "Regex to match repo names against")
	githubCmd.Flags().String("exclude", "^$", "Regex to exclude repo names against")
	githubCmd.Flags().String("archive-dir", "", "Repo to put archived repos in\n(default is .archive in the download dir)")
	githubCmd.Flags().StringP("search", "s", "", "Github search string to use")
	githubCmd.Flags().String("private", "", `DEPRECATED use [--search "is:public"] instead`)
	githubCmd.Flags().String("forks", "", `DEPRECATED use [--search "fork:false"] instead`)

	err := viper.BindPFlags(githubCmd.Flags())
	if err != nil {
		log.Fatalf("Binding flags failed: %s", err)
	}

	viper.AutomaticEnv()
}

func processFlags(args []string) (string, string, string, *regexp.Regexp, *regexp.Regexp) {
	id := args[0]

	dir := filepath.Clean(args[1])

	fmt.Println("=============")
	fmt.Printf("Syncing %s into %s\n", id, dir)

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

	return dir, archiveDir, id, inR, exR
}

func runGithub(args []string) {
	dir, archiveDir, id, inR, exR := processFlags(args)

	repoList := getRepoList(id)

	if !viper.GetBool("verbose") {
		fmt.Println("")
	}

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
	reposToSync.SyncRepos(dir)
	reposToClone.CloneRepos(dir)
	reposToArchive.ArchiveRepos(dir, archiveDir)

	lenSyncWarnings := 0
	warnings := false

	for _, repo := range reposToSync {
		if repo.Severity == actions.Warning {
			if !warnings {
				fmt.Println("=============")
				//nolint:errcheck
				colorstring.Println("[yellow]Warnings:")

				warnings = true
			}

			colorstring.Printf("[green]Sync %s: [yellow]%s", repo.Name, repo.Message)
			lenSyncWarnings++
		}
	}
	// No warnings from clone or archive
	if warnings {
		fmt.Println("=============")
	}

	lenSyncFailures, lenCloneFailures, lenArchiveFailures := 0, 0, 0
	errors := false

	for _, repo := range reposToSync {
		if repo.Severity == actions.Error {
			if !errors {
				fmt.Println("=============")
				//nolint:errcheck
				colorstring.Println("[red]Errors:")

				errors = true
			}

			colorstring.Printf("[green]Sync %s: [red]%s", repo.Name, repo.Message)
			lenSyncFailures++
		}
	}

	for _, repo := range reposToClone {
		if repo.Severity == actions.Error {
			if !errors {
				fmt.Println("=============")
				//nolint:errcheck
				colorstring.Println("[red]Errors:")

				errors = true
			}

			colorstring.Printf("[cyan]Clone %s: [red]%s", repo.Name, repo.Message)
			lenCloneFailures++
		}
	}

	for _, repo := range reposToArchive {
		if repo.Severity == actions.Error {
			if !errors {
				fmt.Println("=============")
				//nolint:errcheck
				colorstring.Println("[red]Errors:")

				errors = true
			}

			colorstring.Printf("[light_magenta]Archive %s: [red]%s", repo.Name, repo.Message)
			lenArchiveFailures++
		}
	}

	if errors {
		fmt.Println("=============")
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

func repoAction(repo *actions.Repo, dirList []string) (action, []string) {
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

func repoActions(
	repoList actions.Repos,
	dirList []string,
	archiveDir string,
	inR *regexp.Regexp,
	exR *regexp.Regexp,
) (actions.Repos, actions.Repos, actions.Repos) {
	var reposToSync actions.Repos

	var reposToClone actions.Repos

	var reposToArchive actions.Repos

	for _, repo := range repoList {
		if inR.MatchString(repo.Name) && !exR.MatchString(repo.Name) {
			var a action

			a, dirList = repoAction(repo, dirList)
			switch a {
			case actionArchive:
				reposToArchive = append(reposToArchive, repo)
			case actionSync:
				reposToSync = append(reposToSync, repo)
			case actionClone:
				reposToClone = append(reposToClone, repo)
			case actionCloneArchive:
				if _, err := os.Stat(fmt.Sprintf("%s/%s", archiveDir, repo.Name)); os.IsNotExist(err) {
					reposToArchive = append(reposToArchive, repo)
					reposToClone = append(reposToClone, repo)
				}
			}
		}
	}

	for _, dir := range dirList {
		reposToArchive = append(reposToArchive, &actions.Repo{
			Name: dir,
		})
	}

	return reposToSync, reposToClone, reposToArchive
}

func getRepoList(id string) actions.Repos {
	fmt.Printf("Getting remote repo list")

	ctx := context.Background()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("Cannot find Github Personal Access Token at env var GITHUB_TOKEN with 'repo' permissions")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	searchQuery := fmt.Sprintf("user:%s fork:true %s", id, viper.GetString("search"))

	rs, err := repoSearch(client, searchQuery)
	if err != nil {
		fmt.Println("")
		log.Fatal(err)
	}

	repos := convertToRepos(rs)

	return repos
}

func getIdType(client *github.Client, id string) (idType, error) {
	result, _, err := client.Users.Get(context.Background(), id)
	if err != nil {
		return unknown, fmt.Errorf("Cannot find org or user [%s] with err: %w", id, err)
	}
	switch *result.Type {
	case "Organization":
		return org, nil
	case "User":
		return user, nil
	}
	return unknown, nil
}

// RepoSearch performs a query against github, consumes all the pages and returns the aggregated results.
func repoSearch(client *github.Client, query string) ([]github.Repository, error) {
	var results []github.Repository
	page := 1
	for {
		fmt.Print(".")
		res, resp, err := client.Search.Repositories(
			context.Background(),
			query,
			&github.SearchOptions{
				ListOptions: github.ListOptions{
					PerPage: 100,
					Page:    page,
				},
			},
		)
		if err != nil {
			if _, ok := err.(*github.AbuseRateLimitError); ok {
				fmt.Print("throttled, backing off")
				time.Sleep(5 * time.Second)
				continue
			}

			return nil, errors.Wrap(err, "unable to perform github repository search request")
		}

		page++
		results = append(results, res.Repositories...)

		if resp.NextPage == 0 {
			break
		}
	}

	return results, nil
}

func convertToRepos(rs []github.Repository) actions.Repos {
	var repos actions.Repos
	for _, r := range rs {
		repos = append(repos, &actions.Repo{
			Name:     *r.Name,
			SSHURL:   *r.SSHURL,
			Archived: *r.Archived,
		})
	}

	return repos
}
