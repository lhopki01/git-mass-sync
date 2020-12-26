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
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "git-mass-sync [org] [download dir]",
	Short: "Utility to mass download all git repos",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		//nolint:gomnd
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("dry-run", "n", false, "Show what would happen")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Make the operation more talkative")
	rootCmd.PersistentFlags().Int("parallelism", 50, "Max parallel processes to run")

	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		log.Fatalf("Binding flags failed: %s", err)
	}

	viper.AutomaticEnv()
}
