package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"skill-eval-harness/internal/dataset"
	"skill-eval-harness/internal/policy"
	"skill-eval-harness/internal/runner"
)

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "seh",
		Short: "SEH is the skill evaluation harness CLI",
		Long:  "SEH is the skill evaluation harness CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newScoreCmd())
	rootCmd.AddCommand(newGateCmd())
	rootCmd.AddCommand(newReportCmd())
	rootCmd.AddCommand(newCompareCmd())
	rootCmd.AddCommand(newDriftCmd())
	rootCmd.AddCommand(newMatrixCmd())
	rootCmd.AddCommand(newFrontierCmd())
	rootCmd.AddCommand(newSimulateCmd())
	rootCmd.AddCommand(newLeaderboardCmd())
	rootCmd.AddCommand(newIngestCmd())
	rootCmd.AddCommand(newBaselineCmd())
	rootCmd.AddCommand(newHistoryCmd())

	return rootCmd
}

func newRunCmd() *cobra.Command {
	var skill string
	var casesDir string
	var outPath string
	var workers int
	var caseTimeout time.Duration
	var maxRetries int
	var tag string
	var sample float64
	var seed int64
	var strict bool

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run evaluation cases with the demo runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			cases, err := dataset.LoadCases(casesDir)
			if err != nil {
				return fmt.Errorf("load cases: %w", err)
			}
			cases = dataset.FilterByTag(cases, tag)
			if sample < 0 || sample > 1 {
				return fmt.Errorf("sample must be between 0 and 1")
			}
			cases = dataset.FilterBySample(cases, sample, seed)

			manifest, err := dataset.LoadManifest(casesDir)
			if err != nil {
				return fmt.Errorf("load manifest: %w", err)
			}
			datasetHash, err := dataset.ComputeDatasetHash(casesDir)
			if err != nil {
				return fmt.Errorf("compute dataset hash: %w", err)
			}

			runtime, err := runner.ResolveRuntime(skill)
			if err != nil {
				return fmt.Errorf("resolve runtime: %w", err)
			}

			result := runner.RunCasesWithOptions(skill, cases, runtime, runner.RunOptions{
				Workers:        workers,
				CaseTimeout:    caseTimeout,
				MaxRetries:     maxRetries,
				DatasetVersion: manifest.Version,
				DatasetHash:    datasetHash,
				Seed:           seed,
			})
			if result.Metrics != nil {
				stabilityVariance, err := runner.ComputeStabilityVariance(result)
				if err != nil {
					return fmt.Errorf("compute stability variance: %w", err)
				}
				result.Metrics.StabilityVariance = stabilityVariance
			}
			if err := runner.WriteJSON(outPath, result); err != nil {
				return fmt.Errorf("write result: %w", err)
			}
			if err := runner.PersistRun(result); err != nil {
				return fmt.Errorf("persist run: %w", err)
			}

			if strict && !result.Success {
				return fmt.Errorf("strict mode: run failed")
			}

			return nil
		},
	}

	runCmd.Flags().StringVar(&skill, "skill", "", "Skill name to execute")
	runCmd.Flags().StringVar(&casesDir, "cases", "", "Directory containing evaluation case YAML files")
	runCmd.Flags().StringVar(&outPath, "out", "", "Path to write the JSON result")
	runCmd.Flags().IntVar(&workers, "workers", 1, "Number of concurrent workers to use")
	runCmd.Flags().DurationVar(&caseTimeout, "case-timeout", 0, "Per-case timeout, for example 250ms or 2s")
	runCmd.Flags().IntVar(&maxRetries, "max-retries", 0, "Maximum retry attempts per case after the first attempt")
	runCmd.Flags().StringVar(&tag, "tag", "", "Only run cases with the specified tag")
	runCmd.Flags().Float64Var(&sample, "sample", 0, "Random sample ratio of cases to run, for example 0.2")
	runCmd.Flags().Int64Var(&seed, "seed", 0, "Deterministic replay seed")
	runCmd.Flags().BoolVar(&strict, "strict", false, "Exit with status 1 if any case fails")

	mustMarkFlagRequired(runCmd, "skill")
	mustMarkFlagRequired(runCmd, "cases")
	mustMarkFlagRequired(runCmd, "out")

	return runCmd
}

func newCompareCmd() *cobra.Command {
	var runRef string
	var baselineRef string
	var failOnRegression bool

	compareCmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare a run against a baseline",
		RunE: func(cmd *cobra.Command, args []string) error {
			current, err := loadRunReference(runRef)
			if err != nil {
				return fmt.Errorf("load run: %w", err)
			}

			baseline, err := loadRunReference(baselineRef)
			if err != nil {
				return fmt.Errorf("load baseline: %w", err)
			}

			comparison := runner.CompareRuns(current, baseline)
			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"Regression summary\nsuccess_rate_delta: %.6f\nlatency_delta: %d\ntoken_delta: %d\n",
				comparison.SuccessRateDelta,
				comparison.LatencyDelta,
				comparison.TokenDelta,
			); err != nil {
				return fmt.Errorf("write comparison summary: %w", err)
			}

			if failOnRegression && comparison.HasRegression() {
				return fmt.Errorf("regression detected")
			}

			return nil
		},
	}

	compareCmd.Flags().StringVar(&runRef, "run", "", "Run JSON file path or persisted run ID")
	compareCmd.Flags().StringVar(&baselineRef, "baseline", "", "Baseline JSON file path or persisted run ID")
	compareCmd.Flags().BoolVar(&failOnRegression, "fail-on-regression", false, "Exit with status 1 if regression is detected")

	mustMarkFlagRequired(compareCmd, "run")
	mustMarkFlagRequired(compareCmd, "baseline")

	return compareCmd
}

func newDriftCmd() *cobra.Command {
	var runRef string
	var baselineRef string
	var outPath string
	var threshold float64

	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Compute output similarity drift across runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			current, err := loadRunReference(runRef)
			if err != nil {
				return fmt.Errorf("load run: %w", err)
			}

			baseline, err := loadRunReference(baselineRef)
			if err != nil {
				return fmt.Errorf("load baseline: %w", err)
			}

			report := runner.ComputeDriftReport(current, baseline, threshold)
			if err := runner.WriteJSON(outPath, report); err != nil {
				return fmt.Errorf("write drift report: %w", err)
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), runner.FormatDriftSummary(report)); err != nil {
				return fmt.Errorf("write drift summary: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&runRef, "run", "", "Run JSON file path or persisted run ID")
	cmd.Flags().StringVar(&baselineRef, "baseline", "", "Baseline JSON file path or persisted run ID")
	cmd.Flags().StringVar(&outPath, "out", "", "Path to write the drift report JSON")
	cmd.Flags().Float64Var(&threshold, "threshold", 0.8, "Similarity threshold below which drift is flagged")

	mustMarkFlagRequired(cmd, "run")
	mustMarkFlagRequired(cmd, "baseline")
	mustMarkFlagRequired(cmd, "out")
	return cmd
}

func newMatrixCmd() *cobra.Command {
	var runtimeNames []string
	var casesDir string
	var outPath string
	var workers int
	var caseTimeout time.Duration
	var maxRetries int
	var tag string
	var sample float64
	var seed int64

	cmd := &cobra.Command{
		Use:   "matrix",
		Short: "Run the same dataset against multiple runtimes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cases, err := dataset.LoadCases(casesDir)
			if err != nil {
				return fmt.Errorf("load cases: %w", err)
			}
			cases = dataset.FilterByTag(cases, tag)
			if sample < 0 || sample > 1 {
				return fmt.Errorf("sample must be between 0 and 1")
			}
			cases = dataset.FilterBySample(cases, sample, seed)

			manifest, err := dataset.LoadManifest(casesDir)
			if err != nil {
				return fmt.Errorf("load manifest: %w", err)
			}
			datasetHash, err := dataset.ComputeDatasetHash(casesDir)
			if err != nil {
				return fmt.Errorf("compute dataset hash: %w", err)
			}

			matrix, err := runner.BuildComparisonMatrix(runtimeNames, cases, runner.RunOptions{
				Workers:        workers,
				CaseTimeout:    caseTimeout,
				MaxRetries:     maxRetries,
				DatasetVersion: manifest.Version,
				DatasetHash:    datasetHash,
				Seed:           seed,
			})
			if err != nil {
				return err
			}

			if err := runner.WriteJSON(outPath, matrix); err != nil {
				return fmt.Errorf("write matrix: %w", err)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "comparison matrix written: %s\n", outPath); err != nil {
				return fmt.Errorf("write matrix output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&runtimeNames, "runtimes", nil, "Comma-separated runtime names to compare")
	cmd.Flags().StringVar(&casesDir, "cases", "", "Directory containing evaluation case YAML files")
	cmd.Flags().StringVar(&outPath, "out", "", "Path to write the JSON comparison matrix")
	cmd.Flags().IntVar(&workers, "workers", 1, "Number of concurrent workers to use")
	cmd.Flags().DurationVar(&caseTimeout, "case-timeout", 0, "Per-case timeout, for example 250ms or 2s")
	cmd.Flags().IntVar(&maxRetries, "max-retries", 0, "Maximum retry attempts per case after the first attempt")
	cmd.Flags().StringVar(&tag, "tag", "", "Only run cases with the specified tag")
	cmd.Flags().Float64Var(&sample, "sample", 0, "Random sample ratio of cases to run, for example 0.2")
	cmd.Flags().Int64Var(&seed, "seed", 0, "Deterministic replay seed")

	mustMarkFlagRequired(cmd, "runtimes")
	mustMarkFlagRequired(cmd, "cases")
	mustMarkFlagRequired(cmd, "out")
	return cmd
}

func newFrontierCmd() *cobra.Command {
	var outPath string

	cmd := &cobra.Command{
		Use:   "frontier",
		Short: "Plot Pareto frontier for cost vs score",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := runner.BuildParetoFrontier()
			if err != nil {
				return err
			}

			if err := runner.WriteJSON(outPath, report); err != nil {
				return fmt.Errorf("write frontier: %w", err)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "frontier written: %s\n", outPath); err != nil {
				return fmt.Errorf("write frontier output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&outPath, "out", "", "Path to write the JSON frontier report")
	mustMarkFlagRequired(cmd, "out")
	return cmd
}

func newIngestCmd() *cobra.Command {
	var runPath string

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest a remote run JSON file into local history",
		RunE: func(cmd *cobra.Command, args []string) error {
			run, err := runner.IngestRun(runPath)
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ingested run: %s\n", run.RunID); err != nil {
				return fmt.Errorf("write ingest output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&runPath, "run", "", "Path to a remote run JSON file")
	mustMarkFlagRequired(cmd, "run")
	return cmd
}

func newBaselineCmd() *cobra.Command {
	baselineCmd := &cobra.Command{
		Use:   "baseline",
		Short: "Manage the promoted baseline run",
	}

	baselineCmd.AddCommand(newBaselinePromoteCmd())
	return baselineCmd
}

func newSimulateCmd() *cobra.Command {
	var outPath string

	cmd := &cobra.Command{
		Use:   "simulate",
		Short: "Simulate routing policies from historical runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := runner.SimulateRoutingPolicies()
			if err != nil {
				return err
			}

			if err := runner.WriteJSON(outPath, report); err != nil {
				return fmt.Errorf("write simulation report: %w", err)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "simulation report generated: %s\n", outPath); err != nil {
				return fmt.Errorf("write simulation output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&outPath, "out", "", "Path to write the simulation report JSON")
	mustMarkFlagRequired(cmd, "out")
	return cmd
}

func newLeaderboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "leaderboard",
		Short: "Rank skills by historical composite score",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := runner.BuildLeaderboard()
			if err != nil {
				return err
			}

			for index, entry := range entries {
				if _, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"%d. %s composite_score=%.6f runs=%d\n",
					index+1,
					entry.Skill,
					entry.CompositeScore,
					entry.Runs,
				); err != nil {
					return fmt.Errorf("write leaderboard output: %w", err)
				}
			}

			return nil
		},
	}
}

func newBaselinePromoteCmd() *cobra.Command {
	var runID string

	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote a persisted run to the baseline pointer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runner.PromoteBaseline(runID); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "promoted baseline: %s\n", runID); err != nil {
				return fmt.Errorf("write baseline promote output: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&runID, "run", "", "Persisted run ID to promote")
	mustMarkFlagRequired(cmd, "run")
	return cmd
}

func newScoreCmd() *cobra.Command {
	var runPath string
	var outPath string
	var configPath string

	scoreCmd := &cobra.Command{
		Use:   "score",
		Short: "Score a prior run result",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(runPath)
			if err != nil {
				return fmt.Errorf("read run result: %w", err)
			}

			var result runner.RunResult
			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("decode run result: %w", err)
			}

			result.Metrics = runner.CalculateMetrics(result.Results)
			stabilityVariance, err := runner.ComputeStabilityVariance(result)
			if err != nil {
				return fmt.Errorf("compute stability variance: %w", err)
			}
			result.Metrics.StabilityVariance = stabilityVariance
			scoreConfig := runner.DefaultScoreConfig
			if configPath != "" {
				cfg, err := runner.LoadScoreConfig(configPath)
				if err != nil {
					return fmt.Errorf("load score config: %w", err)
				}
				scoreConfig = cfg
			}
			result.Metrics.Score = runner.CalculateScore(result, scoreConfig)
			if err := runner.WriteJSON(outPath, result); err != nil {
				return fmt.Errorf("write scored result: %w", err)
			}

			return nil
		},
	}

	scoreCmd.Flags().StringVar(&runPath, "run", "", "Path to a run result JSON file")
	scoreCmd.Flags().StringVar(&outPath, "out", "", "Path to write the scored JSON result")
	scoreCmd.Flags().StringVar(&configPath, "config", "", "Path to a YAML score config file")

	mustMarkFlagRequired(scoreCmd, "run")
	mustMarkFlagRequired(scoreCmd, "out")

	return scoreCmd
}

func newGateCmd() *cobra.Command {
	var reportPath string
	var policyPath string

	gateCmd := &cobra.Command{
		Use:   "gate",
		Short: "Evaluate a scored report against a policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(reportPath)
			if err != nil {
				return fmt.Errorf("read report: %w", err)
			}

			var report runner.RunResult
			if err := json.Unmarshal(data, &report); err != nil {
				return fmt.Errorf("decode report: %w", err)
			}

			policyConfig, err := policy.Load(policyPath)
			if err != nil {
				return fmt.Errorf("load policy: %w", err)
			}

			if err := policy.Evaluate(report, policyConfig); err != nil {
				return err
			}

			return nil
		},
	}

	gateCmd.Flags().StringVar(&reportPath, "report", "", "Path to a scored report JSON file")
	gateCmd.Flags().StringVar(&policyPath, "policy", "", "Path to a YAML policy file")

	mustMarkFlagRequired(gateCmd, "report")
	mustMarkFlagRequired(gateCmd, "policy")

	return gateCmd
}

func newReportCmd() *cobra.Command {
	var inPath string
	var outPath string

	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a simple HTML dashboard from a run report",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(inPath)
			if err != nil {
				return fmt.Errorf("read report: %w", err)
			}

			var run runner.RunResult
			if err := json.Unmarshal(data, &run); err != nil {
				return fmt.Errorf("decode report: %w", err)
			}

			if err := runner.WriteHTMLReport(outPath, run); err != nil {
				return fmt.Errorf("write html report: %w", err)
			}

			return nil
		},
	}

	reportCmd.Flags().StringVar(&inPath, "in", "", "Path to a run or scored report JSON file")
	reportCmd.Flags().StringVar(&outPath, "out", "", "Path to write the HTML report")

	mustMarkFlagRequired(reportCmd, "in")
	mustMarkFlagRequired(reportCmd, "out")

	return reportCmd
}

func newHistoryCmd() *cobra.Command {
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Inspect persisted run history",
	}

	historyCmd.AddCommand(newHistoryListCmd())
	return historyCmd
}

func newHistoryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List persisted run IDs",
		RunE: func(cmd *cobra.Command, args []string) error {
			runIDs, err := runner.ListRuns()
			if err != nil {
				return err
			}

			for _, runID := range runIDs {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), runID); err != nil {
					return fmt.Errorf("write history list: %w", err)
				}
			}

			return nil
		},
	}
}

func loadRunReference(ref string) (runner.RunResult, error) {
	if _, err := os.Stat(ref); err == nil {
		data, err := os.ReadFile(ref)
		if err != nil {
			return runner.RunResult{}, fmt.Errorf("read run file: %w", err)
		}

		var run runner.RunResult
		if err := json.Unmarshal(data, &run); err != nil {
			return runner.RunResult{}, fmt.Errorf("decode run file: %w", err)
		}

		return run, nil
	}

	run, err := runner.LoadRun(ref)
	if err != nil {
		return runner.RunResult{}, err
	}

	return run, nil
}

func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(err)
	}
}
