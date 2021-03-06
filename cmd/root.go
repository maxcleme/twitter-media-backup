/*
Copyright © 2020 maxcleme <maximeclement93+git@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/maxcleme/twitter-media-backup/exporter"
	"github.com/maxcleme/twitter-media-backup/twitter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "twitter-media-backup",
	Short: "Backup twitter media somewhere else",
	Long: `Backup twitter media somewhere else.
Supported media :
 - Photos
 - Videos

Supported destination : 
 - Local
 - Google Photos
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		level, _ := cmd.Flags().GetString("log_level")
		lv, err := logrus.ParseLevel(level)
		if err != nil {
			logrus.WithError(err).Fatal("cannot parse log level")
		}
		logrus.SetLevel(lv)

		applicationKey, _ := cmd.Flags().GetString("twitter_application_key")
		applicationSecret, _ := cmd.Flags().GetString("twitter_application_secret")
		accessToken, _ := cmd.Flags().GetString("twitter_access_token")
		accessTokenSecret, _ := cmd.Flags().GetString("twitter_access_token_secret")
		since, _ := cmd.Flags().GetInt64("twitter_since_tweet_id")
		pollInterval, _ := cmd.Flags().GetDuration("twitter_poll_interval")

		fetcher, err := twitter.NewFetcher(
			twitter.WithApplicationKey(applicationKey),
			twitter.WithApplicationSecret(applicationSecret),
			twitter.WithAccessToken(accessToken),
			twitter.WithAccessTokenSecret(accessTokenSecret),
			twitter.WithSince(since),
			twitter.WithPollInterval(pollInterval),
		)
		if err != nil {
			logrus.WithError(err).Fatal("cannot create twitter fetcher")
		}

		var exporters []exporter.Exporter
		if local, _ := cmd.Flags().GetBool("local"); local {
			rootPath, _ := cmd.Flags().GetString("local_root_path")
			exp, err := exporter.NewLocalExporter(
				exporter.WithRootPath(rootPath),
			)
			if err != nil {
				logrus.WithError(err).Fatal("cannot create local exporter")
			}
			exporters = append(exporters, exp)
		}

		if gphotos, _ := cmd.Flags().GetBool("gphotos"); gphotos {
			tokenPath, _ := cmd.Flags().GetString("gphotos_oauth2_token_path")
			redirectURL, _ := cmd.Flags().GetString("gphotos_oauth2_redirect_url")
			port, _ := cmd.Flags().GetInt("gphotos_oauth2_port")
			applicationKey, _ := cmd.Flags().GetString("gphotos_oauth2_application_key")
			applicationSecret, _ := cmd.Flags().GetString("gphotos_oauth2_application_secret")
			albumName, _ := cmd.Flags().GetString("gphotos_album")

			exp, err := exporter.NewGPhotosExporter(
				exporter.WithApplicationKey(applicationKey),
				exporter.WithApplicationSecret(applicationSecret),
				exporter.WithCallbackPort(port),
				exporter.WithRedirectURL(redirectURL),
				exporter.WithTokenPath(tokenPath),
				exporter.WithAlbumName(albumName),

			)
			if err != nil {
				logrus.WithError(err).Fatal("cannot create gphotos exporter")
			}
			exporters = append(exporters, exp)
		}

		if len(exporters) == 0 {
			logrus.Fatal("at least one exporter need to be enable")
		}
		mediaCh, errCh := fetcher.Content()
		for {
			select {
			case media := <-mediaCh:
				for _, exp := range exporters {
					start := time.Now()
					if err := exp.Export(media); err != nil {
						logrus.
							WithField("type", exp.Type()).
							WithField("media", media.Name).
							WithError(err).
							Fatal("cannot export media")
					}
					logrus.
						WithField("type", exp.Type()).
						WithField("media", media.Name).
						WithField("duration", time.Since(start)).
						Info("success")
				}
			case err := <-errCh:
				logrus.WithError(err).Fatal("error during media polling")
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.twitter-media-backup.yaml)")

	// misc flags
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().String("log_level", "info", "Log level")

	// twitter flags
	rootCmd.Flags().Duration("twitter_poll_interval", time.Second*10, "Twitter polling interval")
	rootCmd.Flags().Int64("twitter_since_tweet_id", -1, "Twitter polling since tweet ID")

	rootCmd.Flags().String("twitter_application_key", "", "Twitter application key")
	rootCmd.MarkFlagRequired("twitter_application_key")

	rootCmd.Flags().String("twitter_application_secret", "", "Twitter application secret")
	rootCmd.MarkFlagRequired("twitter_application_secret")

	rootCmd.Flags().String("twitter_access_token", "", "Twitter user access token")
	rootCmd.MarkFlagRequired("twitter_access_token")

	rootCmd.Flags().String("twitter_access_token_secret", "", "Twitter user access token secret")
	rootCmd.MarkFlagRequired("twitter_access_token_secret")

	// local exporter flags
	rootCmd.Flags().Bool("local", false, "enable local exporter")
	rootCmd.Flags().String("local_root_path", os.TempDir(), "local exporter directory destination")

	// gphotos exporter flags
	rootCmd.Flags().Bool("gphotos", false, "enable Google Photos exporter")
	rootCmd.Flags().String("gphotos_oauth2_token_path", filepath.Join(os.TempDir(), "twitter-media-backup", "gphotos", "token.json"), "Google Photos oauth2 token file location")
	rootCmd.Flags().String("gphotos_oauth2_redirect_url", "http://localhost:8080/callback", "Google Photos oauth2 redirect url used when token file does not exist yet")
	rootCmd.Flags().Int("gphotos_oauth2_port", 8080, "Google Photos oauth2 port used when token file does not exist yet")

	rootCmd.Flags().String("gphotos_oauth2_application_key", "", "Google Photos oauth2 application key")
	rootCmd.MarkFlagRequired("gphotos_oauth2_application_key")

	rootCmd.Flags().String("gphotos_oauth2_application_secret", "", "Google Photos oauth2 application secret")
	rootCmd.MarkFlagRequired("gphotos_oauth2_application_secret")

	rootCmd.Flags().String("gphotos_album", "", "Google Photos album name destination")
	rootCmd.MarkFlagRequired("gphotos_album")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".twitter-media-backup" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".twitter-media-backup")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// workaround to set cobra required flags with viper
	postInitCommands([]*cobra.Command{rootCmd})
}

func postInitCommands(commands []*cobra.Command) {
	for _, cmd := range commands {
		presetRequiredFlags(cmd)
		if cmd.HasSubCommands() {
			postInitCommands(cmd.Commands())
		}
	}
}

func presetRequiredFlags(cmd *cobra.Command) {
	viper.BindPFlags(cmd.Flags())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			cmd.Flags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}
