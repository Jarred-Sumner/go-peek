package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"gitPeek/src/peek"
	"gitPeek/src/repo"

	"github.com/google/go-github/v33/github"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const defaultGithubApiDomain = "api.github.com"

var (
	// Used for flags.
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "git-peek",
		Args:  cobra.MinimumNArgs(0),
		Short: "`git peek` is the fastest way to open a remote git repository in your local text editor.",
		Long:  `Pass git peek a git repository or a github link, and it will quickly fetch and open it in your local editor. It stores the repository in a temporary directory and deletes it when you close the editor or git peek.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			if len(args) > 0 {
				query := repo.GetQuery(args[0])
				destination, err := ioutil.TempDir(os.TempDir(), query.PrettyDirname())
				peek.Log("Destination: %v", destination)
				if err != nil {
					peek.Error("Error creating tmpdir %v for query %v", err, query.Pretty())
				}
				repo.TarballClone(query, destination)
				os.Exit(0)
			} else {
			}
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.git-peek)")
	rootCmd.PersistentFlags().Bool("fromscript", false, "loading git-peek outside a terminal")
	rootCmd.PersistentFlags().BoolP("register", "r", false, "register git-peek://")
	rootCmd.PersistentFlags().BoolP("confirm", "c", false, "always confirm before deleting repository")
	rootCmd.PersistentFlags().StringVarP(&peek.Editor, "editor", "e", "auto", "editor to open with, possible values: auto, code, vim, subl. By default, it will search $EDITOR. If not found, it will try code, then subl, then vim.")
	rootCmd.PersistentFlags().StringVarP(&peek.Branch, "branch", "b", "auto", "branch/ref to use if it can't be detected")
	rootCmd.PersistentFlags().BoolP("defaultBranch", "d", false, "use default_branch from github")
	rootCmd.PersistentFlags().Bool("keep", false, "don't delete the repository")
	rootCmd.PersistentFlags().BoolP("wait", "w", false, "Wait for the repository to completely download before opening. Defaults to false, unless its vim. Then its always true.")
	rootCmd.PersistentFlags().StringVarP(&peek.GithubToken, "token", "t", "", "Wait for the repository to completely download before opening. Defaults to false, unless its vim. Then its always true.")

	viper.BindEnv("editor", "EDITOR")
	viper.BindEnv("token", "GITHUB_TOKEN")
	viper.BindEnv("github_domain", "GITHUB_BASE_DOMAIN")
	viper.BindEnv("github_api_domain", "GITHUB_API_DOMAIN")
	viper.BindPFlag("editor", rootCmd.PersistentFlags().Lookup("editor"))
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.SetDefault("editor", "auto")
	viper.SetDefault("github_domain", "github.com")
	viper.SetDefault("github_api_domain", defaultGithubApiDomain)

}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigName(".git-peek")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	token := viper.GetString("token")

	var httpClient *http.Client
	if token != "" {

		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(ctx, ts)

	} else {
		httpClient = nil
	}

	var githubApiDomain = viper.GetString("github_api_domain")

	if githubApiDomain == defaultGithubApiDomain {
		peek.GithubClient = github.NewClient(httpClient)
	} else {
		client, _ := github.NewEnterpriseClient(githubApiDomain, githubApiDomain, httpClient)
		peek.GithubClient = client
	}

}
