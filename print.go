/*
 * Copyright © 2019 Hedzr Yeh.
 */

package cmdr

import (
	"bufio"
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

func printHelp(command *Command, justFlags bool) {
	if GetIntP(getPrefix(), "help-zsh") > 0 {
		printHelpZsh(command, justFlags)
	} else if GetBoolP(getPrefix(), "help-bash") {
		// TODO for bash
		printHelpZsh(command, justFlags)
	} else {
		paintFromCommand(currentHelpPainter, command, justFlags)
	}

	if rxxtOptions.GetBool("debug") {
		// "  [\x1b[2m\x1b[%dm%s\x1b[0m]"
		fp("\n\x1b[2m\x1b[%dmDUMP:\n\n%v\x1b[0m\n", DarkColor, rxxtOptions.DumpAsString())
	}
}

// WalkAllCommands loop on all commands, started from root.
func WalkAllCommands(walk func(cmd *Command, index int) (err error)) (err error) {
	command := &rootCommand.Command
	err = walkFromCommand(command, 0, walk)
	return
}

func walkFromCommand(cmd *Command, index int, walk func(cmd *Command, index int) (err error)) (err error) {
	if err = walk(cmd, index); err != nil {
		return
	}
	for ix, cc := range cmd.SubCommands {
		if err = walkFromCommand(cc, ix, walk); err != nil {
			return
		}
	}
	return
}

func paintFromCommand(p Painter, command *Command, justFlags bool) {
	printHeader(p, command)

	printHelpUsages(p, command)
	printHelpDescription(p, command)
	printHelpExamples(p, command)
	printHelpSection(p, command, justFlags)

	printHelpTailLine(p, command)

	p.Flush()
}

func printHeader(p Painter, command *Command) {
	p.FpPrintHeader(command)
}

func printHelpTailLine(p Painter, command *Command) {
	p.FpPrintHelpTailLine(command)
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
		x.WriteString(fmt.Sprintf("%d: :((", GetIntP(getPrefix(), "help-zsh")))
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

func printHelpUsages(p Painter, command *Command) {
	if len(rootCommand.Header) == 0 {
		p.FpUsagesTitle(command, "Usages")

		ttl := "[Commands] "
		if command.owner != nil {
			if len(command.SubCommands) == 0 {
				ttl = ""
			} else {
				ttl = "[Sub-Commands] "
			}
		}

		cmds := strings.ReplaceAll(backtraceCmdNames(command), ".", " ")
		if len(cmds) > 0 {
			cmds += " "
		}

		p.FpUsagesLine(command, "", rootCommand.Name, cmds, ttl, command.TailPlaceHolder)
	}
}

func printHelpDescription(p Painter, command *Command) {
	if len(command.Description) > 0 {
		p.FpDescTitle(command, "Description")
		p.FpDescLine(command)
		// fp("\nDescription: \n    %v", command.Description)
	}
}

func printHelpExamples(p Painter, command *Command) {
	if len(command.Examples) > 0 {
		p.FpExamplesTitle(command, "Examples")
		p.FpExamplesLine(command)
		// fp("%v", command.Examples)
	}
}

func printHelpSection(p Painter, command *Command, justFlags bool) {
	if !justFlags {
		printHelpCommandSection(p, command, justFlags)
	}
	printHelpFlagSections(p, command, justFlags)
}

func getSortedKeysFromCmdGroupedMap(m map[string]map[string]*Command) (k0 []string) {
	k0 = make([]string, 0)
	for k := range m {
		if k != UnsortedGroup {
			k0 = append(k0, k)
		}
	}
	sort.Strings(k0)
	// k0 = append(k0, UnsortedGroup)
	k0 = append([]string{UnsortedGroup}, k0...)
	return
}

func getSortedKeysFromCmdMap(groups map[string]*Command) (k1 []string) {
	k1 = make([]string, 0)
	for k := range groups {
		k1 = append(k1, k)
	}
	sort.Strings(k1)
	return
}

func printHelpCommandSection(p Painter, command *Command, justFlags bool) {
	count := 0
	for _, items := range command.allCmds {
		count += len(items)
	}

	if count > 0 {
		p.FpCommandsTitle(command)

		k0 := getSortedKeysFromCmdGroupedMap(command.allCmds)
		for _, group := range k0 {
			groups := command.allCmds[group]
			if len(groups) > 0 {
				p.FpCommandsGroupTitle(group)
				for _, nm := range getSortedKeysFromCmdMap(groups) {
					p.FpCommandsLine(groups[nm])
				}
			}
		}
	}
}

func getSortedKeysFromFlgGroupedMap(m map[string]map[string]*Flag) (k2 []string) {
	k2 = make([]string, 0)
	for k := range m {
		if k != UnsortedGroup {
			k2 = append(k2, k)
		}
	}
	sort.Strings(k2)
	k2 = append([]string{UnsortedGroup}, k2...)
	return
}

func getSortedKeysFromFlgMap(groups map[string]*Flag) (k3 []string) {
	k3 = make([]string, 0)
	for k := range groups {
		k3 = append(k3, k)
	}
	sort.Strings(k3)
	return
}

func printHelpFlagSections(p Painter, command *Command, justFlags bool) {
	sectionName := "Options"

GO_PRINT_FLAGS:
	count := 0
	for _, items := range command.allFlags {
		count += len(items)
	}

	if count > 0 {
		p.FpFlagsTitle(command, nil, sectionName)
		k2 := getSortedKeysFromFlgGroupedMap(command.allFlags)
		for _, group := range k2 {
			groups := command.allFlags[group]
			if len(groups) > 0 {
				p.FpFlagsGroupTitle(group)
				for _, nm := range getSortedKeysFromFlgMap(groups) {
					flg := groups[nm]
					if !flg.Hidden {
						defValStr := ""
						if flg.DefaultValue != nil {
							if ss, ok := flg.DefaultValue.(string); ok && len(ss) > 0 {
								if len(flg.DefaultValuePlaceholder) > 0 {
									defValStr = fmt.Sprintf(" (default %v='%s')", flg.DefaultValuePlaceholder, ss)
								} else {
									defValStr = fmt.Sprintf(" (default='%s')", ss)
								}
							} else {
								if len(flg.DefaultValuePlaceholder) > 0 {
									defValStr = fmt.Sprintf(" (default %v=%v)", flg.DefaultValuePlaceholder, flg.DefaultValue)
								} else {
									defValStr = fmt.Sprintf(" (default=%v)", flg.DefaultValue)
								}
							}
						}
						p.FpFlagsLine(command, flg, defValStr)
						// fp("  %-48s%v%s", flg.GetTitleFlagNames(), flg.Description, defValStr)
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

// SetInternalOutputStreams sets the internal output streams for debugging
func SetInternalOutputStreams(out, err *bufio.Writer) {
	defaultStdout = out
	defaultStderr = err
}

// SetCustomShowVersion supports your `ShowVersion()` instead of internal `showVersion()`
func SetCustomShowVersion(fn func()) {
	globalShowVersion = fn
}

// SetCustomShowBuildInfo supports your `ShowBuildInfo()` instead of internal `showBuildInfo()`
func SetCustomShowBuildInfo(fn func()) {
	globalShowBuildInfo = fn
}

func showVersion() {
	if globalShowVersion != nil {
		globalShowVersion()
		return
	}

	fp(`v%v
%v
%v
%v
%v`, conf.Version, conf.AppName, conf.Buildstamp, conf.Githash, conf.GoVersion)
}

func showBuildInfo() {
	if globalShowBuildInfo != nil {
		globalShowBuildInfo()
		return
	}

	printHeader(currentHelpPainter, &rootCommand.Command)
	// buildTime
	fp(`
       Built by: %v
Build Timestamp: %v
        Githash: %v`, conf.GoVersion, conf.Buildstamp, conf.Githash)
}

func StripOrderPrefix(s string) string {
	if xre.MatchString(s) {
		s = s[strings.Index(s, ".")+1:]
	}
	return s
}

const (
	defaultTimestampFormat = time.RFC3339

	FgBlack        = 30
	FgRed          = 31
	FgGreen        = 32
	FgYellow       = 33
	FgBlue         = 34
	FgMagenta      = 35
	FgCyan         = 36
	FgLightGray    = 37
	FgDarkGray     = 90
	FgLightRed     = 91
	FgLightGreen   = 92
	FgLightYellow  = 93
	FgLightBlue    = 94
	FgLightMagenta = 95
	FgLightCyan    = 96
	FgWhite        = 97

	BgNormal       = 0
	BgBoldOrBright = 1
	BgDim          = 2
	BgItalic       = 3
	BgUnderline    = 4
	BgUlink        = 5
	BgHidden       = 8

	DarkColor = FgLightGray
)

var (
	xre = regexp.MustCompile(`^[0-9A-Za-z]+\.(.+)$`)
)
