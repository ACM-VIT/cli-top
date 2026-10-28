package cmd

import (
	"github.com/spf13/cobra"
	"strings"
)

func isStartupSideEffectFreeArg(arg string) bool {
	switch arg {
	case "schema", "completion", "__complete", "__completeNoDesc", "help", "--help", "-h", "--version", "-v":
		return true
	default:
		return false
	}
}

type startupArgsInfo struct {
	remainingArgs        []string
	rootHelpRequested    bool
	rootVersionRequested bool
	rootUpdateRequested  bool
}

func inspectStartupArgs(args []string) startupArgsInfo {
	info := startupArgsInfo{}

	for i, arg := range args {
		switch {
		case arg == "--":
			if i+1 < len(args) {
				info.remainingArgs = args[i+1:]
			}
			return info
		case arg == "" || arg == "-" || !strings.HasPrefix(arg, "-"):
			info.remainingArgs = args[i:]
			return info
		case strings.HasPrefix(arg, "--"):
			switch arg {
			case "--debug":
				continue
			case "--update":
				info.rootUpdateRequested = true
				continue
			case "--version":
				info.rootVersionRequested = true
				return info
			case "--help":
				info.rootHelpRequested = true
				return info
			default:
				info.remainingArgs = args[i:]
				return info
			}
		default:
			recognizedCluster := true
			for _, shortFlag := range arg[1:] {
				switch shortFlag {
				case 'd':
				case 'u':
					info.rootUpdateRequested = true
				case 'v':
					info.rootVersionRequested = true
					return info
				case 'h':
					info.rootHelpRequested = true
					return info
				default:
					recognizedCluster = false
				}
			}

			if recognizedCluster {
				continue
			}

			info.remainingArgs = args[i:]
			return info
		}
	}

	return info
}

func shouldSkipStartupSideEffects(info startupArgsInfo) bool {
	if info.rootHelpRequested || info.rootVersionRequested {
		return true
	}

	if len(info.remainingArgs) == 0 {
		return false
	}

	if isStartupSideEffectFreeArg(info.remainingArgs[0]) {
		return true
	}

	for _, arg := range info.remainingArgs[1:] {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}

	return false
}

func ShouldSkipStartupSideEffects(args []string) bool {
	return shouldSkipStartupSideEffects(inspectStartupArgs(args))
}

func ShouldSkipCommandSideEffects(cmd *cobra.Command, rootVersionRequested bool) bool {
	if cmd == nil {
		return false
	}

	if flag := cmd.Flags().Lookup("help"); flag != nil && flag.Changed {
		return true
	}
	if flag := cmd.InheritedFlags().Lookup("help"); flag != nil && flag.Changed {
		return true
	}

	switch cmd.Name() {
	case "completion", "__complete", "__completeNoDesc", "help":
		return true
	case "cli-top":
		return rootVersionRequested
	default:
		return false
	}
}
