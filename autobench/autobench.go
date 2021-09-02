package main

import (
	"os"
	"fmt"
	"time"
	"github.com/jessevdk/go-flags"
	"github.com/zededa-yuri/nextgen-storage/autobench/pkg/mkconfig"
	"github.com/zededa-yuri/nextgen-storage/autobench/pkg/fiotests"
)


const fioComandDesc = `
	This command allows you to run FIO testing via an ssh client.
	This command can take configuration values for FIO.

	Example use:
	autobench fio -a=127.0.0.1:22 -u=ubuntu -o=read,randread -b=4k,1m -t=30

`
type Options struct {
	//Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
}

type FioParametrs struct {
	SSHhost string 				`short:"a" long:"adress" description:"ip:port for ssh connections." default:"127.0.0.1:22"`
	SSHUser string 				`short:"u" long:"user" description:"A user name for ssh connections" default:"root"`
	TimeOneTest int 			`short:"t" long:"time" description:"The time that each test will run in sec" default:"60"`
	OpType string				`short:"o" long:"optype" description:"Operation types I/O for fio config" long-description:"Use comma separated string with combinations of read, write, randread, randwrite ..." default:"read,write"`
	BlockSize string 			`short:"b" long:"bs" description:"Block size for fio config"  default:"4k,64k,1m"`
	Iodepth string 				`short:"d" long:"iodepth" description:"Iodepth for fio config" default:"8,16,32"`
	Jobs string					`short:"j" long:"jobs" description:"Jobs for fio config" default:"1,8"`
	TargetFIODevice string 		`short:"T" long:"target" description:"[Optional] To specify block device as a target for FIO. Needs superuser rights (-u=root)."`
	LocalFolderResults string 	`short:"f" long:"folder" description:"[Optional] A name of folder with tests results" default:"FIOTestsResults"`
	LocalDirResults string 		`short:"p" long:"path" description:"[Optional] Path to directory with test results. By default, a folder with the results is created at the path where the utility is launched."`
}

var fioCmd FioParametrs
var opts Options
var parser = flags.NewParser(&opts, flags.Default)

func argparse() {
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
}

// Execute FIO tests via ssh
func (x *FioParametrs) Execute(args []string) error {
	var fioOptions = mkconfig.FioOptions{}

	err := fioOptions.Operations.Set(fioCmd.OpType)
	if err != nil {
		return fmt.Errorf("FIO tests failed: %v", err)
	}

	err = fioOptions.BlockSize.Set(fioCmd.BlockSize)
	if err != nil {
		return fmt.Errorf("FIO tests failed: %v", err)
	}

	err = fioOptions.Jobs.Set(fioCmd.Jobs)
	if err != nil {
		return fmt.Errorf("FIO tests failed: %v", err)
	}

	err = fioOptions.Iodepth.Set(fioCmd.Iodepth)
	if err != nil {
		return fmt.Errorf("FIO tests failed: %v", err)
	}

	if err = fiotests.RunFIOTest(fioCmd.SSHhost, fioCmd.SSHUser, fioCmd.LocalFolderResults, fioCmd.LocalDirResults, fioCmd.TargetFIODevice, fioOptions, 60 * time.Second); err != nil {
		return fmt.Errorf("FIO tests failed: %v", err)
	}

	return nil
}

func init() {
	parser.AddCommand("fio", "Run FIO testing via an ssh client. Use the fio --help command for more information.", fioComandDesc, &fioCmd)
}

func main() {
	argparse()
}
