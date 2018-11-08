package app

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
)

type runnerFlags struct {
	genFlags
	showAll         bool
	skipPreflight   bool
	pollingInterval time.Duration
	statusLoopDelay time.Duration
	retrievalDelay  time.Duration
	timeout         time.Duration
}

var (
	runnerflags    runnerFlags
	sonobuoyClient *client.SonobuoyClient
)

func RunnerFlagSet(cfg *runnerFlags) *pflag.FlagSet {
	runnerset := pflag.NewFlagSet("runner", pflag.ExitOnError)
	// Default to detect since we need kubeconfig regardless
	runnerset.AddFlagSet(GenFlagSet(&cfg.genFlags, DetectRBACMode, ConformanceImageVersionAuto))
	AddSkipPreflightFlag(&cfg.skipPreflight, runnerset)

	runnerset.DurationVar(
		&cfg.timeout, "runner-timeout", 6*time.Hour,
		"Length of time to give the runner before giving up. (Default: 6h)",
	)
	runnerset.DurationVar(
		&cfg.pollingInterval, "polling-interval", 5*time.Minute,
		"Duration of time between polling sonobuoy for status. (Default: 5m)",
	)
	runnerset.BoolVar(
		&cfg.showAll, "show-all", false,
		"Don't summarize plugin statuses, show all individually",
	)

	return runnerset
}

func (r *runnerFlags) Config() (*client.RunConfig, error) {
	gencfg, err := r.genFlags.Config()
	if err != nil {
		return nil, err
	}
	return &client.RunConfig{
		GenConfig: *gencfg,
	}, nil
}

func init() {
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Submits a sonobuoy run and waits for it's completion",
		Run:   startSonobuoyRunner,
		Args:  cobra.ExactArgs(0),
	}

	cmd.Flags().AddFlagSet(RunnerFlagSet(&runnerflags))
	RootCmd.AddCommand(cmd)
}

func startSonobuoyRunner(cmd *cobra.Command, args []string) {
	var signals chan os.Signal
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := runnerflags.kubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get REST client"))
		os.Exit(1)
	}

	runCfg, err := runnerflags.Config()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not retrieve E2E config"))
		os.Exit(1)
	}
	sonobuoyClient, err = getSonobuoyClient(cfg)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}

	err = run(runCfg)
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	ticker := time.NewTicker(runnerflags.pollingInterval)
	go func() {
		for range ticker.C {
			status, err := status()
			if err != nil {
				errlog.LogError(err)
				continue
			}

			switch status.Status {
			case aggregation.RunningStatus:
				continue
			case aggregation.FailedStatus:
				errlog.LogError(errors.New("Aggregator has status of failed"))
				os.Exit(1)
			case aggregation.CompleteStatus:
				fmt.Printf("Run completed")
				os.Exit(0)
			default:
				errlog.LogError(errors.New("Unknown status for aggregator"))
				os.Exit(1)
			}
		}
	}()

	time.Sleep(runnerflags.timeout)
	ticker.Stop()
	errlog.LogError(errors.New("Runner timed out waiting for tests to complete"))
	os.Exit(1)
}

func run(runCfg *client.RunConfig) error {
	plugins := make([]string, len(runCfg.Config.PluginSelections))
	for i, plugin := range runCfg.Config.PluginSelections {
		plugins[i] = plugin.Name
	}

	if len(plugins) > 0 {
		fmt.Printf("Running plugins: %v\n", strings.Join(plugins, ", "))
	}

	if !runnerflags.skipPreflight {
		if errs := sonobuoyClient.PreflightChecks(&client.PreflightConfig{Namespace: runnerflags.namespace}); len(errs) > 0 {
			for _, err := range errs {
				errlog.LogError(err)
			}
			return errors.New("Preflight checks failed")
		}
	}

	if err := sonobuoyClient.Run(runCfg); err != nil {
		return errors.Wrap(err, "error attempting to run sonobuoy")
	}

	return nil
}

func status() (*aggregation.Status, error) {
	status, err := sonobuoyClient.GetStatus(statusFlags.namespace)
	if err != nil {
		return status, errors.Wrap(err, "error attempting to run sonobuoy")
	}

	if runnerflags.showAll {
		err = printAll(os.Stdout, status)
	} else {
		err = printSummary(os.Stdout, status)
	}

	return status, err
}
