/*
Copyright © 2026 Kot1gruN
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/gophercloud/gophercloud/v2/openstack"
	_ "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/catalog"
	_ "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"

	"jakapil-cli/cmd/craas"
	"jakapil-cli/cmd/mks"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jakapil-cli",
	Short: "Absolutely unofficial CLI client for Selectel services",
	Long: `Absolutely unofficial CLI client for Selectel cloud and baremetal services.
Just for now it supports only those subcommands, which can use already
existing go libraries from Selectel.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Register the mks command tree.
	rootCmd.AddCommand(mks.NewMksCommand())

	// Register the craas command tree.
	rootCmd.AddCommand(craas.NewCraasCommand())

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jakapil-cli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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

		// Search config in home directory with name ".jakapil-cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".jakapil-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
