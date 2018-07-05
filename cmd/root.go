// Copyright © 2017 Mesosphere Inc. <http://mesosphere.com>
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
	"fmt"
	"os"
	"runtime"

	"github.com/dcos/dcos-check-runner/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var defaultCheckConfig = "/opt/mesosphere/etc/dcos-check-config.json"
var defaultCheckConfigWindows = "\\DCOS\\check-runner\\config\\dcos-check-config.json"

var (
	version       bool
	checkCfgFile  string
	cfgFile       string
	defaultConfig = &config.Config{}
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "dcos-check-runner",
	Short: "DC/OS check-runner service",
	Long:  "dcos-check-runner check provides CLI functionality to run checks on DC/OS cluster.",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if version {
			fmt.Printf("Version: %s\n", config.Version)
			os.Exit(0)
		}

		cmd.Help()
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	if runtime.GOOS == "windows" {
		defaultCheckConfig = os.Getenv("SYSTEMDRIVE") + defaultCheckConfigWindows
	}

	RootCmd.PersistentFlags().StringVar(&checkCfgFile, "check-config", defaultCheckConfig,
		"Path to check configuration file")
	RootCmd.PersistentFlags().BoolVar(&version, "version", false, "Print dcos-check-runner version")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /opt/mesosphere/etc/dcos-check-runner.yaml)")
	RootCmd.PersistentFlags().BoolVar(&defaultConfig.FlagVerbose, "verbose", defaultConfig.FlagVerbose,
		"Use verbose debug output.")
	RootCmd.PersistentFlags().StringVar(&defaultConfig.FlagRole, "role", defaultConfig.FlagRole,
		"Set node role")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName("dcos-check-runner") // name of config file (without extension)
	viper.AddConfigPath("/opt/mesosphere/etc/")
	viper.AutomaticEnv()

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if err := defaultConfig.LoadFromViper(viper.AllSettings()); err != nil {
			logrus.Fatalf("Error loading config file: %s", err)
		}
	}
}
