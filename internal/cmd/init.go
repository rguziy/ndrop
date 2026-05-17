package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rguziy/ndrop/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a default client config in ~/.config/ndrop/ndrop.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := config.DefaultConfigPath()
			cfgDir := filepath.Dir(cfgPath)
			if err := os.MkdirAll(cfgDir, 0o755); err != nil {
				return fmt.Errorf("create config dir: %w", err)
			}

			if _, err := os.Stat(cfgPath); err == nil && !force {
				fmt.Fprintf(os.Stderr, "skipping %s (exists)\n", cfgPath)
				return nil
			} else if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("inspect config file: %w", err)
			}

			content := fmt.Sprintf("[server]\nurl = \"http://localhost:8080\"\ntoken = \"your-token\"\n\n[pull]\ndefault_save_dir = \"%s\"\n", config.DefaultClientConfig().Pull.DefaultSaveDir)
			if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			fmt.Fprintf(os.Stderr, "wrote %s\n", cfgPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing client config")
	return cmd
}
