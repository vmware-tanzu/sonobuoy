/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/crash-diagnostics/exec"
	"github.com/vmware-tanzu/crash-diagnostics/logging"
	"github.com/vmware-tanzu/crash-diagnostics/util"
	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	"github.com/vmware-tanzu/sonobuoy/pkg/discovery"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/tarball"
)

const (
	ArgWorkDir = "workdir"
)

var (
	allowedGenFlagsWithRunFile = []string{kubeconfig, kubecontext}
)

func givenAnyGenConfigFlags(fs *pflag.FlagSet, allowedFlagNames []string) bool {
	changed := false
	fs.Visit(func(f *pflag.Flag) {
		if changed {
			return
		}
		if f.Changed && !stringInList(allowedFlagNames, f.Name) {
			changed = true
		}
	})
	return changed
}

func NewCmdRun() *cobra.Command {
	var f genFlags
	fs := GenFlagSet(&f, DetectRBACMode)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Starts a Sonobuoy run by launching the Sonobuoy aggregator and plugin pods.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkFlagValidity(fs, f)
		},
		Run:  submitSonobuoyRun(&f),
		Args: cobra.ExactArgs(0),
	}

	cmd.Flags().AddFlagSet(fs)
	return cmd
}

func checkFlagValidity(fs *pflag.FlagSet, rf genFlags) error {
	if rf.genFile != "" && givenAnyGenConfigFlags(fs, allowedGenFlagsWithRunFile) {
		return fmt.Errorf("setting the --file flag is incompatible with any other options besides %v", allowedGenFlagsWithRunFile)
	}
	return nil
}

func submitSonobuoyRun(f *genFlags) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		sbc, err := getSonobuoyClientFromKubecfg(f.kubecfg)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		runCfg, err := f.RunConfig()
		if err != nil {

		}

		if runCfg.IsLocal() {
			if err := runLocal(runCfg, f.kubecfg); err != nil {
				errlog.LogError(errors.Wrap(err, "could not retrieve E2E config"))
				os.Exit(1)
			}
			return
		}

		if !contains(f.skipPreflight, "true") && !contains(f.skipPreflight, "*") {
			pcfg := &client.PreflightConfig{
				Namespace:           f.sonobuoyConfig.Namespace,
				DNSNamespace:        f.dnsNamespace,
				DNSPodLabels:        f.dnsPodLabels,
				PreflightChecksSkip: f.skipPreflight,
			}
			if errs := sbc.PreflightChecks(pcfg); len(errs) > 0 {
				errlog.LogError(errors.New("Preflight checks failed"))
				for _, err := range errs {
					errlog.LogError(err)
				}
				os.Exit(1)
			}
		}

		if err := sbc.Run(runCfg); err != nil {
			errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
			os.Exit(1)
		}
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func tryCopyFile(src, dst string) {
	if err := copyFile(src, dst); err != nil {
		logrus.Errorf("Failed to copy the script file %v to %v: %v", src, dst, err)
	}
}

func runLocal(cfg *client.RunConfig, kubecfg Kubeconfig) error {

	t := time.Now()
	runUUID := uuid.New()
	outpath := filepath.Join(".", runUUID.String())
	metapath := filepath.Join(outpath, discovery.MetaLocation)
	err := os.MkdirAll(metapath, 0755)
	if err != nil {
		panic(err)
	}
	runInfo := discovery.RunInfo{LoadedPlugins: []string{}}
	cfg.Config.UUID = runUUID.String()
	cfg.Config.QueryDir = runUUID.String()

	for _, p := range cfg.StaticPlugins {
		runInfo.LoadedPlugins = append(runInfo.LoadedPlugins, p.SonobuoyConfig.PluginName)
		file, err := os.Open(p.ScriptFile)
		if err != nil {
			return err
		}

		pluginWorkdir := filepath.Join(".", cfg.Config.UUID, "plugins", p.SonobuoyConfig.PluginName)
		err = os.MkdirAll(pluginWorkdir, 0755)
		if err != nil {
			panic(err)
		}
		tryCopyFile(p.ScriptFile, filepath.Join(pluginWorkdir, filepath.Base(p.ScriptFile)))
		tryCopyFile(p.ArgsFile, filepath.Join(pluginWorkdir, filepath.Base(p.ArgsFile)))

		setupLocalPluginLogging(filepath.Join(pluginWorkdir, "plugin.log"))
		argsMap, err := processScriptArguments(p.ArgsFile)
		if err != nil {
			panic(err)
		}

		// Force this value to work within the sonobuoy tar output format.
		argsMap[ArgWorkDir] = pluginWorkdir

		if err := exec.ExecuteFile(file, argsMap); err != nil {
			return err
		}
	}

	// 3. Dump the config.json we used to run our test
	if blob, err := json.Marshal(cfg); err == nil {
		logrus.Trace("Recording the marshalled Sonobuoy config")
		if err = ioutil.WriteFile(filepath.Join(metapath, "config.json"), blob, 0644); err != nil {
			errlog.LogError(errors.Wrap(err, "could not write config.json file"))
		}
	} else {
		errlog.LogError(errors.Wrap(err, "error marshalling Sonobuoy config"))
	}

	// TODO(jschnake) Local plugins can still benefit from having queries run BUT you should make sure that if the API server is broken this doesn't make things grind to a halt. It should just error and put something in the logs about it.
	restConf, err := kubecfg.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get rest config")
	}
	discovery.QueryCluster(restConf, cfg.Config)

	/*
		// Postprocessing before we create the tarball.
		for _, p := range cfg.LoadedPlugins {
			logrus.WithField("plugin", p.GetName()).Trace("Post-processing")
			item, errs := results.PostProcessPlugin(p, outpath)
			for _, e := range errs {
				logrus.Errorf("Error processing plugin %v: %v", p.GetName(), e)
			}

			// Save results object regardless of errors; it is our best effort to understand the results.
			if err := results.SaveProcessedResults(p.GetName(), outpath, item); err != nil {
				logrus.Errorf("Unable to save results for plugin %v: %v", p.GetName(), err)
			}
		}

	*/

	// Dump extra metadata that may be useful to postprocessors or analysis.
	blob, err := json.Marshal(runInfo)
	if err != nil {
		logrus.Errorf("marshalling run info: %v", err)
	}
	if err == nil {
		if err = ioutil.WriteFile(filepath.Join(metapath, results.InfoFile), blob, 0644); err != nil {
			logrus.Errorf("Error saving runinfo file %v: %v", results.InfoFile, err)
		}
	}

	filename := fmt.Sprintf("%v_sonobuoy_%v.tar.gz", t.Format("200601021504"), runUUID)
	tb := filepath.Join(".", filename)
	err = tarball.DirToTarball(outpath, tb, true)
	if err == nil {
		defer os.RemoveAll(outpath)
	} else {
		logrus.Error("Failed to tar up the directory %v: %v", tb, err)
	}

	fmt.Printf("Result tarball available at %v\n", tb)

	return nil
}

// prepares a map of key-value strings to be passed to the execution script
// It builds the map from the args-file as well as the args flag passed to
// the run command.
func processScriptArguments(argsFile string) (map[string]string, error) {
	scriptArgs := map[string]string{}

	// get args from script args file
	if err := util.ReadArgsFile(argsFile, scriptArgs); err != nil {
		return nil, errors.Wrapf(err, "failed to parse scriptArgs file: %s", argsFile)
	}

	return scriptArgs, nil
}

// TODO(jschnake) uncouple this from crashd specifics; no need for ~/crashd directory etc
// TODO(jschnake) print output doesnt get put in the log. Either need to reroute the `print` logic to use log instead or need to redirect os.stdout/stderr
// I think the executor should just take this sort of logging info/output info as part of its creation.
func setupLocalPluginLogging(logfile string) {
	// Log everything to file, regardless of settings for CLI.
	filehook, err := logging.NewFileHook(logfile)
	if err != nil {
		logrus.Warning("Failed to log to file, logging to stdout (default)")
	} else {
		logrus.AddHook(filehook)
	}

	level := logrus.TraceLevel
	logrus.AddHook(logging.NewCLIHook(os.Stdout, level))

	// Set to trace so all hooks fire. We will handle levels differently for CLI/file.
	logrus.SetOutput(io.Discard)
	logrus.SetLevel(logrus.TraceLevel)
}

func stringInList(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
