package cmd

import (
	"fmt"
	"os"

	"github.com/rguziy/ndrop/internal/config"
	"github.com/spf13/cobra"
)

// globalFlags are flags shared across all subcommands.
type globalFlags struct {
	ConfigPath string
	ServerURL  string
	Token      string
}

// loadConfig is a helper used by every subcommand to build a validated config.
func loadConfig(g *globalFlags) (config.ClientConfig, error) {
	cfg, err := config.LoadClient(g.ConfigPath, g.ServerURL, g.Token)
	if err != nil {
		return cfg, fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// NewRootCmd builds and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	globals := &globalFlags{}

	root := &cobra.Command{
		Use:   "ndrop",
		Short: "Transfer text and files between devices",
		Long: `ndrop — a cross-platform data transfer utility.

Content is end-to-end encrypted: the server only stores ciphertext
derived from your token.

Push supports:
  ndrop push "text"             inline text
  ndrop push ./file             file contents
  echo "text" | ndrop push      stdin
  ndrop push --clipboard        system clipboard text
  ndrop push -c "command"       command stdout

Pull supports:
  ndrop pull                    text to stdout; files to default save dir
  ndrop pull --clipboard        text to system clipboard
  ndrop pull --save <dir>       save file output to a directory
  ndrop pull --stdout           raw bytes to stdout

Configuration: ~/.config/ndrop/ndrop.toml
Environment:   NDROP_URL, NDROP_TOKEN`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global persistent flags available to every subcommand.
	root.PersistentFlags().StringVar(
		&globals.ConfigPath, "config",
		config.DefaultConfigPath(),
		"path to config file",
	)
	root.PersistentFlags().StringVar(
		&globals.ServerURL, "server", "",
		"server URL (overrides config and NDROP_URL)",
	)
	root.PersistentFlags().StringVar(
		&globals.Token, "token", "",
		"auth token (overrides config and NDROP_TOKEN)",
	)

	root.AddCommand(
		newInitCmd(),
		newPushCmd(globals),
		newPullCmd(globals),
	)

	return root
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
