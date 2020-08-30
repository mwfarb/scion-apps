// Copyright 2019 ETH Zurich
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"flag"
	"os"
	"path"
	"path/filepath"
)

// default params for localhost testing
var listenAddrDef = "127.0.0.1"
var listenPortDef = 8000
var defaultSciond = "127.0.0.1:30255"

// command argument constants
var CMD_ADR = "a"
var CMD_PRT = "p"
var CMD_SCD = "sciond"
var CMD_ART = "sabin"
var CMD_WEB = "srvroot"

var GOPATH = os.Getenv("GOPATH")

type CmdOptions struct {
	Addr       string
	Port       int
	Sciond     string
	StaticRoot string
	AppsRoot   string
}

func (o *CmdOptions) AbsPathCmdOptions() {
	o.StaticRoot, _ = filepath.Abs(o.StaticRoot)
	o.AppsRoot, _ = filepath.Abs(o.AppsRoot)
}

func isFlagUsed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// defaultAppsRoot returns the directory containing the webapp executable as
// the default base directory for the apps resources
func defaultAppsRoot() string {
	exec, err := os.Executable()
	if err != nil {
		return ""
	}
	return path.Dir(exec)
}

func defaultStaticRoot(appsRoot string) string {
	return path.Join(appsRoot, "../chat/web")
}

func ParseFlags() CmdOptions {
	addr := flag.String(CMD_ADR, listenAddrDef, "Address of server host.")
	port := flag.Int(CMD_PRT, listenPortDef, "Port of server host.")
	sciond := flag.String(CMD_SCD, defaultSciond, "SCIOND address")
	appsRoot := flag.String(CMD_ART, defaultAppsRoot(),
		"Path to execute the installed scionlab apps binaries")
	staticRoot := flag.String(CMD_WEB, defaultStaticRoot(*appsRoot),
		"Path to read/write web server files.")
	flag.Parse()
	// recompute root args to use the proper relative defaults if undefined
	if !isFlagUsed(CMD_WEB) {
		*staticRoot = defaultStaticRoot(*appsRoot)
	}
	options := CmdOptions{*addr, *port, *sciond, *staticRoot, *appsRoot}
	options.AbsPathCmdOptions()
	return options
}
