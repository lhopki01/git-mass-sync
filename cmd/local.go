package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/lhopki01/git-mass-sync/pkg/actions"
	"github.com/mitchellh/colorstring"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var localCmd = &cobra.Command{
	Use:   "local [target dir]",
	Short: "Sync all repos within the target directory",
	//nolint:gomnd
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runLocal(args)
	},
}

func init() {
	rootCmd.AddCommand(localCmd)
}

func runLocal(args []string) {
	dir := filepath.Clean(args[0])

	fmt.Println("=============")
	fmt.Printf("Syncing all git repos in %s", dir)
	fmt.Println("=============")

	var reposToSync actions.Repos
	for _, dir := range actions.GetGitDirList(dir) {
		reposToSync = append(reposToSync, &actions.Repo{
			Name: dir,
		})
	}

	lenSync := len(reposToSync)

	fmt.Println("=============")
	colorstring.Printf("[green]%d repos to sync\n", lenSync)
	fmt.Println("=============")

	reposToSync.SyncRepos(dir)

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

	lenSyncFailures := 0
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

	if errors {
		fmt.Println("=============")
	}

	if !viper.GetBool("dry-run") {
		fmt.Println("=============")

		if lenSyncFailures > 0 {
			colorstring.Printf("[red]%d[reset]/[green]%d repos synced\n", lenSync-lenSyncFailures, lenSync)
		} else {
			colorstring.Printf("[green]%d/%d repos synced\n", lenSync-lenSyncFailures, lenSync)
		}
	}
}
