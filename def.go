/*
 * Copyright © 2019 Hedzr Yeh.
 */

package cmdr

import (
	"bufio"
	"errors"
	"os"
)

const (
	appNameDefault = "cmdr"

	// UnsortedGroup for commands and flags
	UnsortedGroup = "zzzz.unsorted"
	// SysMgmtGroup for commands and flags
	SysMgmtGroup = "zzz9.Misc"
)

type (
	getopt struct {
		Name string
	}

	// BaseOpt is base of `Command`, `Flag`
	BaseOpt struct {
		Name string
		// single char. example for flag: "a" -> "-a"
		// Short rune.
		Short string
		// word string. example for flag: "addr" -> "--addr"
		Full string
		// more synonyms
		Aliases []string
		// group name
		Group string
		// to-do: Toggle Group
		ToggleGroup string

		owner  *Command
		strHit string

		Flags []*Flag

		Description             string
		LongDescription         string
		Examples                string
		Hidden                  bool
		DefaultValuePlaceholder string

		// cmd 是 flag 被识别时已经得到的子命令
		// return: ErrShouldBeStopException will break the following flow and exit right now
		Action func(cmd *Command, args []string) (err error)
	}

	// Command holds the structure of commands and subcommands
	Command struct {
		BaseOpt
		SubCommands []*Command
		// return: ErrShouldBeStopException will break the following flow and exit right now
		PreAction func(cmd *Command, args []string) (err error)
		// PostAction will be run after Action() invoked.
		PostAction func(cmd *Command, args []string)
		// be shown at tail of command usages line. Such as for TailPlaceHolder="<host-fqdn> <ipv4/6>":
		// austr dns add <host-fqdn> <ipv4/6> [Options] [Parent/Global Options]
		TailPlaceHolder string

		root            *RootCommand
		allCmds         map[string]map[string]*Command // key1: Commnad.Group, key2: Command.Full
		allFlags        map[string]map[string]*Flag    // key1: Command.Flags[#].Group, key2: Command.Flags[#].Full
		plainCmds       map[string]*Command
		plainShortFlags map[string]*Flag
		plainLongFlags  map[string]*Flag
	}

	// RootCommand holds some application information
	RootCommand struct {
		Command

		AppName    string
		Version    string
		VersionInt uint32

		Copyright string
		Author    string
		Header    string // using `Header` for header and ignore built with `Copyright` and `Author`, and no usage lines too.

		ow   *bufio.Writer
		oerr *bufio.Writer
	}

	// Flag means a flag, a option, or a opt.
	Flag struct {
		BaseOpt

		// default value for flag
		DefaultValue interface{}
		ValidArgs    []string
		Required     bool

		// by default, a flag is always `optional`.
	}

	// Options is a holder of all options
	Options struct {
		entries   map[string]interface{}
		hierarchy map[string]interface{}
	}

	// OptOne struct {
	// 	Children map[string]*OptOne `yaml:"c,omitempty"`
	// 	Value    interface{}        `yaml:"v,omitempty"`
	// }

	// ConfigReloaded for config reloaded
	ConfigReloaded interface {
		OnConfigReloaded()
	}

	HookXrefFunc func(root *RootCommand, args []string)
)

var (
	// EnableVersionCommands supports injecting the default `--version` flags and commands
	EnableVersionCommands = true
	// EnableHelpCommands supports injecting the default `--help` flags and commands
	EnableHelpCommands = true
	// EnableVerboseCommands supports injecting the default `--verbose` flags and commands
	EnableVerboseCommands = true
	// EnableGenerateCommands supports injecting the default `generate` commands and subcommands
	EnableGenerateCommands = true

	// rootCommand the root of all commands
	rootCommand *RootCommand
	// rootOptions *Opt
	rxxtOptions = NewOptions()

	// RxxtPrefix create a top-level namespace, which contains all normalized `Flag`s.
	RxxtPrefix = []string{"app"}

	// EnvPrefix attaches a prefix to key to retrieve the option value.
	EnvPrefix = []string{"CMDR"}

	// usedConfigFile
	usedConfigFile            string
	usedConfigSubDir          string
	configFiles               []string
	onConfigReloadedFunctions map[ConfigReloaded]bool
	//
	predefinedLocations = []string{
		"./ci/etc/%s/%s.yml",
		"/etc/%s/%s.yml",
		"/usr/local/etc/%s/%s.yml",
		os.Getenv("HOME") + "/.%s/%s.yml",
	}

	//
	defaultStdout = bufio.NewWriterSize(os.Stdout, 16384)
	defaultStderr = bufio.NewWriterSize(os.Stderr, 16384)

	//
	currentHelpPainter       = new(helpPainter)
	CurrentDescColor         = FgDarkGray
	CurrentDefaultValueColor = FgDarkGray
	CurrentGroupTitleColor   = DarkColor

	globalShowVersion   func()
	globalShowBuildInfo func()

	beforeXrefBuilding []HookXrefFunc
	afterXrefBuilt     []HookXrefFunc

	// ErrShouldBeStopException tips `Exec()` cancelled the following actions after `PreAction()`
	ErrShouldBeStopException = errors.New("should be stop right now")
)

// GetStrictMode enables error when opt value missed. such as:
// xxx a b --prefix''   => error: prefix opt has no value specified.
// xxx a b --prefix'/'  => ok.
//
// ENV: use `CMDR_APP_STRICT_MODE=true` to enable strict-mode.
// NOTE: `CMDR_APP_` prefix could be set by user (via: `EnvPrefix` && `RxxtPrefix`).
//
// the flag value of `--strict-mode`.
func GetStrictMode() bool {
	return GetBool("app.strict-mode")
}

// GetDebugMode returns the flag value of `--debug`/`-D`
func GetDebugMode() bool {
	return GetBool("app.debug")
}

// GetDebugMode returns the flag value of `--verbose`/`-v`
func GetVerboseMode() bool {
	return GetBool("app.verbose")
}

// GetDebugMode returns the flag value of `--quiet`/`-q`
func GetQuietMode() bool {
	return GetBool("app.quiet")
}
