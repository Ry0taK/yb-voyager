/*
Copyright (c) YugaByte, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nightlyone/lockfile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yugabyte/yb-voyager/yb-voyager/src/callhome"
	"github.com/yugabyte/yb-voyager/yb-voyager/src/utils"
)

var (
	cfgFile    string
	exportDir  string
	startClean bool
	lockFile   lockfile.Lockfile
)

var rootCmd = &cobra.Command{
	Use:   "yb-voyager",
	Short: "A CLI based migration engine to migrate complete database(schema + data) from some source database to YugabyteDB",
	Long: `A CLI based migration engine for complete database migration from a source database to YugabyteDB. Currently supported source databases are Oracle, MySQL, PostgreSQL.
Refer to docs (https://docs.yugabyte.com/preview/migrate/) for more details like setting up source/target, migration workflow etc.`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		if exportDir != "" && utils.FileOrFolderExists(exportDir) {
			if cmd.Use != "version" && cmd.Use != "status" {
				lockExportDir()
			}
			InitLogging(exportDir, cmd.Use == "status")
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if exportDir != "" && utils.FileOrFolderExists(exportDir) && cmd.Use != "version" && cmd.Use != "status" {
			unlockExportDir()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	callhome.ReadEnvSendDiagnostics()
}

func registerCommonGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&source.VerboseMode, "verbose", false,
		"enable verbose mode for the console output")

	cmd.PersistentFlags().StringVarP(&exportDir, "export-dir", "e", "",
		"export directory is the workspace used to keep the exported schema, data, state, and logs")

	cmd.PersistentFlags().BoolVarP(&utils.DoNotPrompt, "yes", "y", false,
		"assume answer as yes for all questions during migration (default false)")

	cmd.PersistentFlags().BoolVar(&callhome.SendDiagnostics, "send-diagnostics", true,
		"enable or disable the 'send-diagnostics' feature that sends analytics data to Yugabyte.")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".yb-voyager" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".yb-voyager")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func validateExportDirFlag() {
	if exportDir == "" {
		utils.ErrExit(`ERROR: required flag "export-dir" not set`)
	}
	if !utils.FileOrFolderExists(exportDir) {
		utils.ErrExit("export-dir %q doesn't exists.\n", exportDir)
	} else if exportDir == "." {
		fmt.Println("Note: Using current working directory as export directory")
	} else {
		exportDir = strings.TrimRight(exportDir, "/")
	}
}

func lockExportDir() {
	lockFileName, err := filepath.Abs(filepath.Join(exportDir, ".lockfile.lck"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get the lockfile path: %v\n", err)
		os.Exit(1)
	}

	lockFile, err = lockfile.New(lockFileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create lockfile %q: %v\n", lockFileName, err)
		os.Exit(1)
	}

	err = lockFile.TryLock()
	if err == nil {
		return
	} else if err == lockfile.ErrBusy {
		fmt.Fprintf(os.Stderr, "Another instance of yb-voyager is running in the export-dir = %s\n", exportDir)
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "Unable to lock the export-dir: %v\n", err)
		os.Exit(1)
	}

}

func unlockExportDir() {
	err := lockFile.Unlock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to unlock %q: %v\n", lockFile, err)
		os.Exit(1)
	}
}
