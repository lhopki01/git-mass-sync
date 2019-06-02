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
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Wrong number of arguments")
		}
		return nil
	},
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

	reposToSync := actions.GetGitDirList(dir)
	lenSync := len(reposToSync)

	fmt.Println("=============")
	colorstring.Printf("[green]%d repos to sync\n", lenSync)
	fmt.Println("=============")

	failedSyncRepos, warningSyncRepos := actions.SyncRepos(reposToSync, dir)
	lenSyncWarnings := len(warningSyncRepos)
	lenSyncFailures := len(failedSyncRepos)

	if lenSyncWarnings > 0 {
		fmt.Println("=============")
		//nolint:errcheck
		colorstring.Println("[green]Sync repos [yellow]warnings")
		for _, s := range warningSyncRepos {
			//nolint:errcheck
			colorstring.Println(s)
		}
	}
	if lenSyncFailures > 0 {
		fmt.Println("=============")
		//nolint:errcheck
		colorstring.Println("[red]Failed [green]sync repos")
		for _, s := range failedSyncRepos {
			//nolint:errcheck
			colorstring.Println(s)
		}
	}
	if !viper.GetBool("dry-run") {
		fmt.Println("=============")
		if lenSyncFailures > 0 {
			colorstring.Printf("[red]%d[reset]/[green]%d repos synced", lenSync-lenSyncFailures, lenSync)

		} else {
			colorstring.Printf("[green]%d/%d repos synced", lenSync-lenSyncFailures, lenSync)
		}
	}
}
