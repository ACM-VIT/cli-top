package cmd

import (
	"cli-top/debug"
	"cli-top/helpers"
	"fmt"
	"os"
	"strings"

	"github.com/lpernett/godotenv"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or change cli-top settings",
	Run: func(cmd *cobra.Command, args []string) {
		loadConfigEnv()
		helpers.Printf("auto-select  %s\n", onOff(helpers.AutoSelectSemesterEnabled()))
	},
}

var configAutoSelectCmd = &cobra.Command{
	Use:          "auto-select [on|off]",
	Short:        "Use the latest semester by default instead of listing semesters",
	Args:         cobra.MaximumNArgs(1),
	ValidArgs:    []string{"on", "off"},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		loadConfigEnv()
		if len(args) == 0 {
			helpers.Printf("auto-select is %s\n", onOff(helpers.AutoSelectSemesterEnabled()))
			return nil
		}

		enabled, err := parseOnOff(args[0])
		if err != nil {
			return err
		}
		if err := helpers.SetAutoSelectSemester(enabled); err != nil {
			if debug.Debug {
				return fmt.Errorf("write config: %w", err)
			}
			return fmt.Errorf("unable to write the cli-top configuration")
		}
		helpers.Printf("auto-select is now %s\n", onOff(enabled))
		return nil
	},
}

func loadConfigEnv() {
	if err := godotenv.Load(configFilePath()); err != nil && !os.IsNotExist(err) && debug.Debug {
		helpers.Println("Error loading .env file:", err)
	}
}

func parseOnOff(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value %q, use on or off", value)
	}
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func init() {
	configCmd.AddCommand(configAutoSelectCmd)
	rootCmd.AddCommand(configCmd)
}
