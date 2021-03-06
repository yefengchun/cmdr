/*
 * Copyright © 2019 Hedzr Yeh.
 */

package cmdr

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

//

//

// Exec is main entry of `cmdr`.
func Exec(rootCmd *RootCommand) (err error) {
	err = InternalExecFor(rootCmd, os.Args)
	return
}

// ExecWith is main entry of `cmdr`.
func ExecWith(rootCmd *RootCommand, beforeXrefBuilding_, afterXrefBuilt_ HookXrefFunc) (err error) {
	beforeXrefBuilding = append(beforeXrefBuilding, beforeXrefBuilding_)
	afterXrefBuilt = append(afterXrefBuilt, afterXrefBuilt_)
	err = InternalExecFor(rootCmd, os.Args)
	return
}

func setRootCommand(rootCmd *RootCommand) {
	rootCommand = rootCmd

	rootCommand.ow = defaultStdout
	rootCommand.oerr = defaultStderr
}

func buildRefs(rootCmd *RootCommand) (err error) {
	buildRootCrossRefs(rootCmd)
	for _, s := range getExpandedPredefinedLocations() {
		if FileExists(s) {
			fn := fmt.Sprintf(s, rootCmd.AppName, rootCmd.AppName)
			err = rxxtOptions.LoadConfigFile(fn)
			if err != nil {
				return
			}
			break
		}
	}
	return
}

// AddOnBeforeXrefBuilding add hook func
func AddOnBeforeXrefBuilding(cb HookXrefFunc) {
	beforeXrefBuilding = append(beforeXrefBuilding, cb)
}

// AddOnAfterXrefBuilt add hook func
func AddOnAfterXrefBuilt(cb HookXrefFunc) {
	afterXrefBuilt = append(afterXrefBuilt, cb)
}

// InternalExecFor is an internal helper, esp for debugging
func InternalExecFor(rootCmd *RootCommand, args []string) (err error) {
	var (
		pkg       = new(ptpkg)
		ok        = false
		goCommand = &rootCmd.Command
		// helpFlag = rootCmd.allFlags[UnsortedGroup]["help"]
	)

	if rootCommand == nil {
		setRootCommand(rootCmd)
	}

	defer func() {
		_ = rootCmd.ow.Flush()
		_ = rootCmd.oerr.Flush()
	}()

	for _, x := range beforeXrefBuilding {
		x(rootCmd, args)
	}

	err = buildRefs(rootCmd)
	if err != nil {
		return
	}

	for _, x := range afterXrefBuilt {
		x(rootCmd, args)
	}

	for pkg.i = 1; pkg.i < len(args); pkg.i++ {
		pkg.a = args[pkg.i]
		pkg.assigned = false
		pkg.short = false
		pkg.savedFn = ""
		pkg.savedVal = ""

		// --debug: long opt
		// -D:      short opt
		// -nv:     double chars short opt
		// ~~debug: long opt without opt-entry prefix.
		// ~D:      short opt without opt-entry prefix.
		// -abc:    the combined short opts
		// -nvabc, -abnvc: a,b,c,nv the four short opts, if no -n & -v defined.
		// --name=consul, --name consul, --name=consul: opt with a string, int, string slice argument
		// -nconsul, -n consul, -n=consul: opt with an argument.
		//  - -nconsul is not good format, but it could get somewhat works.
		//  - -n'consul', -n"consul" could works too.
		// -t3: opt with an argument.
		if pkg.a[0] == '-' || pkg.a[0] == '/' || pkg.a[0] == '~' {
			if len(pkg.a) == 1 {
				pkg.needHelp = true
				pkg.needFlagsHelp = true
				continue
			}

			// flag
			if len(pkg.a) > 1 && (pkg.a[1] == '-' || pkg.a[1] == '~') {
				if len(pkg.a) == 2 {
					// disableParser = true // '--': ignore the following args
					break
				}

				// long flag
				pkg.fn = pkg.a[2:]
				findValueAttached(pkg, &pkg.fn)
			} else {
				pkg.fn = pkg.a[1:2]
				pkg.savedFn = pkg.a[2:]
				pkg.short = true
				findValueAttached(pkg, &pkg.savedFn)
			}

			pkg.suffix = pkg.fn[len(pkg.fn)-1]
			if pkg.suffix == '+' || pkg.suffix == '-' {
				pkg.fn = pkg.fn[0 : len(pkg.fn)-1]
			} else {
				pkg.suffix = 0
			}

			// fn + val
			// fn: short,
			// fn: long
			// fn: short||val: such as '-t3'
			// fn: long=val, long='val', long="val", long val, long 'val', long "val"
			// fn: longval, long'val', long"val"

			pkg.savedGoCommand = goCommand
			cc := goCommand
		GO_UP:
			pkg.found = false
			if pkg.short {
				pkg.flg, ok = cc.plainShortFlags[pkg.fn]
			} else {
				var fn = pkg.fn
				var ln = len(fn)
				for ; ln > 1; ln-- {
					fn = pkg.fn[0:ln]
					pkg.flg, ok = cc.plainLongFlags[fn]
					if ok {
						if ln < len(pkg.fn) {
							pkg.val = pkg.fn[ln:]
							pkg.fn = fn
							pkg.assigned = true
						}
						break
					}
				}
			}

			if ok {
				if err = recogValue(pkg, args); err != nil {
					return
				}

				if pkg.found {
					// if !GetBoolP(getPrefix(), "quiet") {
					// 	logrus.Debugf("-- flag '%v' hit, go ahead...", pkg.flg.GetTitleName())
					// }
					if pkg.flg.Action != nil {
						if err = pkg.flg.Action(goCommand, getArgs(pkg, args)); err == ErrShouldBeStopException {
							return nil
						}
					}

					if !pkg.assigned {
						if len(pkg.savedFn) > 0 && len(pkg.savedVal) == 0 {
							pkg.fn = pkg.savedFn[0:1]
							pkg.savedFn = pkg.savedFn[1:]
							goto GO_UP
						}
					}
				}
			} else {
				if cc.owner != nil {
					// match the flag within parent's flags set.
					cc = cc.owner
					goto GO_UP
				}
				if !pkg.assigned && pkg.short {
					// try matching 2-chars short opt
					if len(pkg.savedFn) > 0 {
						fnf := pkg.fn + pkg.savedFn
						pkg.fn = fnf[0:2]
						pkg.savedFn = fnf[2:]
						goCommand = pkg.savedGoCommand
						goto GO_UP
					}
				}
				ferr("Unknown flag: %v", pkg.a)
				pkg.unknownFlags = append(pkg.unknownFlags, pkg.a)
			}

		} else {
			// command, files
			if cmd, ok := goCommand.plainCmds[pkg.a]; ok {
				cmd.strHit = pkg.a
				goCommand = cmd
				// logrus.Debugf("-- command '%v' hit, go ahead...", cmd.GetTitleName())
				if cmd.PreAction != nil {
					if err = cmd.PreAction(goCommand, getArgs(pkg, args)); err == ErrShouldBeStopException {
						return nil
					}
				}
			} else {
				if goCommand.Action != nil && len(goCommand.SubCommands) == 0 {
					// the args remained are files, not sub-commands.
					pkg.i--
					break
				}

				ferr("Unknown command: %v", pkg.a)
				pkg.unknownCmds = append(pkg.unknownCmds, pkg.a)
			}
		}
	}

	if !pkg.needHelp {
		pkg.needHelp = GetBoolP(getPrefix(), "help")
	}

	if !pkg.needHelp && len(pkg.unknownCmds) == 0 && len(pkg.unknownFlags) == 0 {
		if goCommand.Action != nil {
			args := getArgs(pkg, args)

			if goCommand != &rootCmd.Command {
				if rootCmd.PostAction != nil {
					defer rootCmd.PostAction(goCommand, args)
				}
				if rootCmd.PreAction != nil {
					if err = rootCmd.PreAction(goCommand, getArgs(pkg, args)); err == ErrShouldBeStopException {
						return nil
					}
				}
			}

			if goCommand.PostAction != nil {
				defer goCommand.PostAction(goCommand, args)
			}

			if err = goCommand.Action(goCommand, args); err == ErrShouldBeStopException {
				return nil
			}

			return
		}
	}

	if GetIntP(getPrefix(), "help-zsh") > 0 || GetBoolP(getPrefix(), "help-bash") {
		if len(goCommand.SubCommands) == 0 && !pkg.needFlagsHelp {
			// pkg.needFlagsHelp = true
		}
	}

	printHelp(goCommand, pkg.needFlagsHelp)

	return
}

func getPrefix() string {
	return strings.Join(RxxtPrefix, ".")
}

func getArgs(pkg *ptpkg, args []string) []string {
	var a []string
	if pkg.i+1 < len(args) {
		a = args[pkg.i+1:]
	}
	return a
}

func isTypeInt(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int:
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
	default:
		return false
	}
	return true
}

func isTypeUInt(kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
	default:
		return false
	}
	return true
}

func isTypeSInt(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int:
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
	default:
		return false
	}
	return true
}

type ptpkg struct {
	assigned          bool
	found             bool
	short             bool
	fn, val           string
	savedFn, savedVal string
	i                 int
	a                 string
	flg               *Flag
	savedGoCommand    *Command
	needHelp          bool
	needFlagsHelp     bool
	suffix            uint8
	unknownCmds       []string
	unknownFlags      []string
}

func findValueAttached(pkg *ptpkg, fn *string) {
	if strings.Contains(*fn, "=") {
		aa := strings.Split(*fn, "=")
		*fn = aa[0]
		pkg.val = trimQuotes(aa[1])
		pkg.assigned = true
	} else {
		splitQuotedValueIfNecessary(pkg, fn)
	}
}

func splitQuotedValueIfNecessary(pkg *ptpkg, fn *string) {
	if pos := strings.Index(*fn, "'"); pos >= 0 {
		pkg.val = trimQuotes((*fn)[pos:])
		*fn = (*fn)[0:pos]
		pkg.assigned = true
	} else if pos := strings.Index(*fn, "\""); pos >= 0 {
		pkg.val = trimQuotes((*fn)[pos:])
		*fn = (*fn)[0:pos]
		pkg.assigned = true
	} else {
		// --xVALUE need to be parsed.
	}
}

func recogValue(pkg *ptpkg, args []string) (err error) {
	if _, ok := pkg.flg.DefaultValue.(bool); ok {
		if pkg.suffix == '+' {
			pkg.flg.DefaultValue = true
		} else if pkg.suffix == '-' {
			pkg.flg.DefaultValue = false
		} else {
			pkg.flg.DefaultValue = true
		}

		if pkg.a[0] == '~' {
			rxxtOptions.SetNx(backtraceFlagNames(pkg.flg), pkg.flg.DefaultValue)
		} else {
			rxxtOptions.Set(backtraceFlagNames(pkg.flg), pkg.flg.DefaultValue)
		}
		pkg.found = true

	} else {
		vv := reflect.ValueOf(pkg.flg.DefaultValue)
		kind := vv.Kind()
		switch kind {
		case reflect.String:
			if err = processTypeString(pkg, args); err != nil {
				return
			}

		case reflect.Slice:
			typ := reflect.TypeOf(pkg.flg.DefaultValue).Elem()
			if typ.Kind() == reflect.String {
				if err = processTypeStringSlice(pkg, args); err != nil {
					return
				}
			} else if isTypeSInt(typ.Kind()) {
				if err = processTypeIntSlice(pkg, args); err != nil {
					return
				}
			} else if isTypeSInt(typ.Kind()) {
				if err = processTypeUintSlice(pkg, args); err != nil {
					return
				}
			}

		default:
			if isTypeSInt(kind) {
				if err = processTypeInt(pkg, args); err != nil {
					return
				}
			} else if isTypeUInt(kind) {
				if err = processTypeUint(pkg, args); err != nil {
					return
				}
			} else {
				ferr("Unacceptable default value kind=%v", kind)
			}
		}
	}
	return
}

func preprocessPkg(pkg *ptpkg, args []string) (err error) {
	if !pkg.assigned {
		if len(pkg.savedVal) > 0 {
			pkg.val = pkg.savedVal
			pkg.savedVal = ""
		} else if len(pkg.savedFn) > 0 {
			pkg.val = pkg.savedFn
			pkg.savedFn = ""
		} else {
			if pkg.i < len(args)-1 {
				pkg.i++
				pkg.val = args[pkg.i]
			} else if GetStrictMode() {
				err = fmt.Errorf("unexpect end of command line [i=%v,args=(%v)], need more args for %v", pkg.i, args, pkg)
				return
			}
		}
		pkg.assigned = true
	}
	return
}

func processTypeInt(pkg *ptpkg, args []string) (err error) {
	if err = preprocessPkg(pkg, args); err != nil {
		return
	}

	v, err := strconv.ParseInt(pkg.val, 10, 64)
	if err != nil {
		ferr("wrong number: flag=%v, number=%v", pkg.fn, pkg.val)
	}

	if pkg.a[0] == '~' {
		rxxtOptions.SetNx(backtraceFlagNames(pkg.flg), v)
	} else {
		rxxtOptions.Set(backtraceFlagNames(pkg.flg), v)
	}
	pkg.found = true
	return
}

func processTypeUint(pkg *ptpkg, args []string) (err error) {
	if err = preprocessPkg(pkg, args); err != nil {
		return
	}

	v, err := strconv.ParseUint(pkg.val, 10, 64)
	if err != nil {
		ferr("wrong number: flag=%v, number=%v", pkg.fn, pkg.val)
	}

	if pkg.a[0] == '~' {
		rxxtOptions.SetNx(backtraceFlagNames(pkg.flg), v)
	} else {
		rxxtOptions.Set(backtraceFlagNames(pkg.flg), v)
	}
	pkg.found = true
	return
}

func processTypeString(pkg *ptpkg, args []string) (err error) {
	if err = preprocessPkg(pkg, args); err != nil {
		return
	}

	if pkg.a[0] == '~' {
		rxxtOptions.SetNx(backtraceFlagNames(pkg.flg), pkg.val)
	} else {
		rxxtOptions.Set(backtraceFlagNames(pkg.flg), pkg.val)
	}
	pkg.found = true
	return
}

func processTypeStringSlice(pkg *ptpkg, args []string) (err error) {
	if err = preprocessPkg(pkg, args); err != nil {
		return
	}

	if pkg.a[0] == '~' {
		rxxtOptions.SetNx(backtraceFlagNames(pkg.flg), strings.Split(pkg.val, ","))
	} else {
		rxxtOptions.Set(backtraceFlagNames(pkg.flg), strings.Split(pkg.val, ","))
	}
	pkg.found = true
	return
}

func processTypeIntSlice(pkg *ptpkg, args []string) (err error) {
	if err = preprocessPkg(pkg, args); err != nil {
		return
	}

	valary := make([]int64, 0)
	for _, x := range strings.Split(pkg.val, ",") {
		if xi, err := strconv.ParseInt(x, 10, 64); err == nil {
			valary = append(valary, xi)
		}
	}

	if pkg.a[0] == '~' {
		rxxtOptions.SetNx(backtraceFlagNames(pkg.flg), valary)
	} else {
		rxxtOptions.Set(backtraceFlagNames(pkg.flg), valary)
	}
	pkg.found = true
	return
}

func processTypeUintSlice(pkg *ptpkg, args []string) (err error) {
	if err = preprocessPkg(pkg, args); err != nil {
		return
	}

	valary := make([]uint64, 0)
	for _, x := range strings.Split(pkg.val, ",") {
		if xi, err := strconv.ParseUint(x, 10, 64); err == nil {
			valary = append(valary, xi)
		}
	}

	if pkg.a[0] == '~' {
		rxxtOptions.SetNx(backtraceFlagNames(pkg.flg), valary)
	} else {
		rxxtOptions.Set(backtraceFlagNames(pkg.flg), valary)
	}
	pkg.found = true
	return
}
