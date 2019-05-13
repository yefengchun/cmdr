/*
 * Copyright © 2019 Hedzr Yeh.
 */

package cmdr

import (
	"fmt"
	"github.com/hedzr/cmdr/conf"
	"regexp"
	"sort"
	"strings"
	"time"
)

func fp(fmtStr string, args ...interface{}) {
	_, _ = fmt.Fprintf(rootCommand.ow, fmtStr+"\n", args...)
}

func ferr(fmtStr string, args ...interface{}) {
	_, _ = fmt.Fprintf(rootCommand.oerr, fmtStr+"\n", args...)
}

func printHeader() {
	if len(rootCommand.Header) == 0 {
		fp("%v by %v - v%v", rootCommand.Copyright, rootCommand.Author, rootCommand.Version)
	} else {
		fp("%v", rootCommand.Header)
	}
}

func printHelp(command *Command, justFlags bool) {
	if GetInt("app.help-zsh") > 0 {
		// if !GetBool("app.quiet") {
		// 	logrus.Debugf("zsh-dump")
		// }
		printHelpZsh(command, justFlags)

	} else if GetBool("app.help-bash") {
		printHelpZsh(command, justFlags)

	} else {

		printHeader()

		printHelpUsages(command)
		printHelpExamples(command)

		printHelpSection(command, justFlags)

		printHelpTailLine(command)

	}

	if RxxtOptions.GetBool("debug") {
		// "  [\x1b[2m\x1b[%dm%s\x1b[0m]"
		fp("\n\x1b[2m\x1b[%dmDUMP:\n\n%v\x1b[0m\n", darkColor, RxxtOptions.DumpAsString())
	}
}

func printHelpZsh(command *Command, justFlags bool) {
	if command == nil {
		command = &rootCommand.Command
	}

	printHelpZshCommands(command, justFlags)
}

func printHelpZshCommands(command *Command, justFlags bool) {
	if !justFlags {
		var x strings.Builder
		x.WriteString(fmt.Sprintf("%d: :((", GetInt("app.help-zsh")))
		for _, cx := range command.SubCommands {
			for _, n := range cx.GetExpandableNamesArray() {
				x.WriteString(fmt.Sprintf(`%v:'%v' `, n, cx.Description))
			}

			// fp(`  %-25s  %v%v`, cx.GetName(), cx.GetQuotedGroupName(), cx.Description)

			// fp(`%v:%v`, cx.GetExpandableNames(), cx.Description)
			// printHelpZshCommands(cx)
		}
		x.WriteString("))")
		fp("%v", x.String())
	} else {
		for _, flg := range command.Flags {
			// fp(`  %-25s  %v`,
			// 	// "--help", //
			// 	// flg.GetTitleZshFlagNames(" "),
			// 	flg.GetTitleZshFlagName(), flg.GetDescZsh())
			for _, ff := range flg.GetTitleZshFlagNamesArray() {
				// fp(`  %-25s  %v`, ff, flg.GetDescZsh())
				fp(`%s[%v]`, ff, flg.GetDescZsh())
				// fp(`%s[%v]:%v:`, ff, flg.GetDescZsh(), flg.DefaultValuePlaceholder)
			}
		}
		fp(`(: -)--help[Print usage]`)
		// fp(`  %-25s  %v`, "--help", "Print Usage")
	}
}

func printHelpUsages(command *Command) {
	if len(rootCommand.Header) == 0 {
		fp("\nUsages: ")
		ttl := "Commands"
		if command.owner != nil {
			ttl = "Sub-Commands"
		}
		cmds := strings.ReplaceAll(backtraceCmdNames(command), ".", " ")
		if len(cmds) > 0 {
			cmds += " "
		}
		fp("    %s %v[%s] [Options] [Parent/Global Options]", rootCommand.Name, cmds, ttl)
	}
}

func printHelpExamples(command *Command) {
	if len(command.Examples) > 0 {
		fp("%v", command.Examples)
	}
}

func printHelpTailLine(command *Command) {
	fp("\nType '-h' or '--help' to get command help screen.")
}

func printHelpSection(command *Command, justFlags bool) {
	if !justFlags {
		printHelpCommandSection(command, justFlags)
	}
	printHelpFlagSections(command, justFlags)
}

func printHelpCommandSection(command *Command, justFlags bool) {
	count := 0
	for _, items := range command.allCmds {
		count += len(items)
	}

	if count > 0 {
		if command.owner == nil {
			fp("\nCommands:")
		} else {
			fp("\nSub-Commands:")
		}
		k0 := make([]string, 0)
		for k, _ := range command.allCmds {
			if k != UNSORTED_GROUP {
				k0 = append(k0, k)
			}
		}
		sort.Strings(k0)
		// k0 = append(k0, UNSORTED_GROUP)
		k0 = append([]string{UNSORTED_GROUP}, k0...)

		for _, group := range k0 {
			groups := command.allCmds[group]
			if len(groups) > 0 {
				if group != UNSORTED_GROUP {
					// fp("  [%s]:", normalize(group))
					fp("  [\x1b[2m\x1b[%dm%s\x1b[0m]", darkColor, normalize(group))
				}

				k1 := make([]string, 0)
				for k, _ := range groups {
					k1 = append(k1, k)
				}
				sort.Strings(k1)

				for _, nm := range k1 {
					cmd := groups[nm]
					if !cmd.Hidden {
						fp("  %-48s%v", cmd.GetTitleNames(), cmd.Description)
					}
				}
			}
		}
	}
}

func printHelpFlagSections(command *Command, justFlags bool) {
	sectionName := "Options"

GO_PRINT_FLAGS:
	count := 0
	for _, items := range command.allFlags {
		count += len(items)
	}

	if count > 0 {
		fp("\n%v:", sectionName)
		k2 := make([]string, 0)
		for k, _ := range command.allFlags {
			if k != UNSORTED_GROUP {
				k2 = append(k2, k)
			}
		}
		sort.Strings(k2)
		k2 = append([]string{UNSORTED_GROUP}, k2...)

		for _, group := range k2 {
			groups := command.allFlags[group]
			if len(groups) > 0 {
				if group != UNSORTED_GROUP {
					// // echo -e "Normal \e[2mDim"
					// _, _ = fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m\x1b[2m\x1b[%dm[%04d]\x1b[0m%-44s \x1b[2m\x1b[%dm%s\x1b[0m ",
					// 	levelColor, levelText, darkColor, int(entry.Time.Sub(baseTimestamp)/time.Second), entry.Message, darkColor, caller)
					fp("  [\x1b[2m\x1b[%dm%s\x1b[0m]", darkColor, normalize(group))
				}

				k3 := make([]string, 0)
				for k, _ := range groups {
					k3 = append(k3, k)
				}
				sort.Strings(k3)

				for _, nm := range k3 {
					flg := groups[nm]
					if !flg.Hidden {
						fp("  %-48s%v", flg.GetTitleFlagNames(), flg.Description)
					}
				}
			}
		}
	}

	if command.owner != nil {
		command = command.owner
		// sectionName = "Parent/Global Options"
		if command.owner == nil {
			sectionName = "Global Options"
		} else {
			sectionName = fmt.Sprintf("Parent (`%v`) Options", command.GetTitleName())
		}
		goto GO_PRINT_FLAGS
	}

}

func normalize(s string) string {
	if xre.MatchString(s) {
		s = s[strings.Index(s, ".")+1:]
	}
	return s
}

func showVersion() {
	if globalShowVersion != nil {
		globalShowVersion()
		return
	}

	fp("v%v", conf.Version)
	fp("%v", conf.AppName)
	fp("%v", conf.Buildstamp)
	fp("%v", conf.Githash)
}

func showBuildInfo() {
	if globalShowBuildInfo != nil {
		globalShowBuildInfo()
		return
	}

	printHeader()
	// buildTime
	fp("Build Timestamp: %v. Githash: %v", conf.Buildstamp, conf.Githash)
}

const (
	defaultTimestampFormat = time.RFC3339

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97

	boldOrBright = 1
	dim          = 2
	underline    = 4
	blink        = 5
	hidden       = 8

	darkColor = lightGray
)

var (
	xre = regexp.MustCompile(`^[0-9A-Za-z]+\.(.+)$`)
)