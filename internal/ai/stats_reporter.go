package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"powergrid/pkg/logger"
)

// StatsReporter generates detailed statistics reports
type StatsReporter struct {
	logger *logger.ColoredLogger
}

// NewStatsReporter creates a new statistics reporter
func NewStatsReporter(logger *logger.ColoredLogger) *StatsReporter {
	return &StatsReporter{
		logger: logger,
	}
}

// GenerateReport creates a comprehensive report from simulation results
func (r *StatsReporter) GenerateReport(results []GameStatistics) *SimulationReport {
	report := &SimulationReport{
		Timestamp:      time.Now(),
		TotalGames:     len(results),
		GameResults:    results,
		StrategyStats:  make(map[string]*StrategyStatistics),
		PhaseAnalysis:  make(map[string]*PhaseAnalysis),
		ResourceTrends: make(map[string]*ResourceAnalysis),
	}

	// Calculate basic stats
	completedGames := 0
	totalDuration := time.Duration(0)
	strategyWins := make(map[string]int)
	strategyGames := make(map[string]int)

	for _, result := range results {
		if result.Completed {
			completedGames++
			totalDuration += result.Duration
			
			// Track wins by strategy (would need to enhance GameStatistics)
			if result.Winner != "" {
				strategyWins[r.inferStrategy(result.Winner)]++
			}
		}

		// Track games by strategy
		for playerID := range result.PlayerActions {
			strategy := r.inferStrategy(playerID)
			strategyGames[strategy]++
		}
	}

	report.CompletedGames = completedGames
	report.FailedGames = report.TotalGames - completedGames
	
	if completedGames > 0 {
		report.AverageDuration = totalDuration / time.Duration(completedGames)
		report.CompletionRate = float64(completedGames) / float64(report.TotalGames) * 100
	}

	// Analyze by strategy
	r.analyzeStrategies(report, results, strategyWins, strategyGames)

	// Analyze phases
	r.analyzePhases(report, results)

	// Analyze resources
	r.analyzeResources(report, results)

	return report
}

// PrintReport outputs a formatted report
func (r *StatsReporter) PrintReport(report *SimulationReport) {
	r.logger.Info("=== POWER GRID AI SIMULATION REPORT ===")
	r.logger.Info("Generated: %s", report.Timestamp.Format("2006-01-02 15:04:05"))
	r.logger.Info("")
	
	// Overall Statistics
	r.logger.Info("📊 OVERALL STATISTICS")
	r.logger.Info("Total Games: %d", report.TotalGames)
	r.logger.Info("Completed: %d (%.1f%%)", report.CompletedGames, report.CompletionRate)
	r.logger.Info("Failed: %d", report.FailedGames)
	r.logger.Info("Average Duration: %v", report.AverageDuration)
	r.logger.Info("")

	// Strategy Performance
	r.logger.Info("🎯 STRATEGY PERFORMANCE")
	strategies := r.getSortedStrategies(report.StrategyStats)
	for _, strategy := range strategies {
		stats := report.StrategyStats[strategy]
		r.logger.Info("  %s:", strategy)
		r.logger.Info("    Games: %d", stats.GamesPlayed)
		r.logger.Info("    Wins: %d (%.1f%%)", stats.Wins, stats.WinRate)
		r.logger.Info("    Avg Cities: %.1f", stats.AverageCities)
		r.logger.Info("    Avg Actions: %.1f", stats.AverageActions)
		r.logger.Info("    Avg Final Money: $%.0f", stats.AverageFinalMoney)
	}
	r.logger.Info("")

	// Phase Analysis
	r.logger.Info("⏱️  PHASE ANALYSIS")
	phases := r.getSortedPhases(report.PhaseAnalysis)
	for _, phase := range phases {
		analysis := report.PhaseAnalysis[phase]
		r.logger.Info("  %s:", phase)
		r.logger.Info("    Total Occurrences: %d", analysis.TotalOccurrences)
		r.logger.Info("    Avg per Game: %.1f", analysis.AveragePerGame)
		r.logger.Info("    Avg Duration: %v", analysis.AverageDuration)
	}
	r.logger.Info("")

	// Resource Trends
	r.logger.Info("🔋 RESOURCE ANALYSIS")
	resources := r.getSortedResources(report.ResourceTrends)
	for _, resource := range resources {
		analysis := report.ResourceTrends[resource]
		r.logger.Info("  %s:", resource)
		r.logger.Info("    Total Used: %d", analysis.TotalUsed)
		r.logger.Info("    Avg per Game: %.1f", analysis.AveragePerGame)
		r.logger.Info("    Peak Demand: %d", analysis.PeakDemand)
	}
	r.logger.Info("")

	// Top Performers
	if len(report.TopPerformers) > 0 {
		r.logger.Info("🏆 TOP PERFORMERS")
		for i, performer := range report.TopPerformers {
			if i >= 5 {
				break
			}
			r.logger.Info("  %d. %s - %d wins", i+1, performer.Name, performer.Wins)
		}
		r.logger.Info("")
	}

	// Insights
	if len(report.Insights) > 0 {
		r.logger.Info("💡 KEY INSIGHTS")
		for _, insight := range report.Insights {
			r.logger.Info("  • %s", insight)
		}
		r.logger.Info("")
	}

	r.logger.Info("=======================================")
}

// SaveReport saves the report to a file
func (r *StatsReporter) SaveReport(report *SimulationReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	r.logger.Info("Report saved to: %s", filename)
	return nil
}

// Private helper methods

func (r *StatsReporter) analyzeStrategies(report *SimulationReport, results []GameStatistics, wins, games map[string]int) {
	for strategy := range games {
		stats := &StrategyStatistics{
			Strategy:    strategy,
			GamesPlayed: games[strategy],
			Wins:        wins[strategy],
		}

		if stats.GamesPlayed > 0 {
			stats.WinRate = float64(stats.Wins) / float64(stats.GamesPlayed) * 100
		}

		// Calculate averages
		totalCities := 0
		totalActions := 0
		totalMoney := 0
		count := 0

		for _, result := range results {
			for playerID, cities := range result.CitiesBuilt {
				if r.inferStrategy(playerID) == strategy {
					totalCities += cities
					totalActions += result.PlayerActions[playerID]
					count++
				}
			}
		}

		if count > 0 {
			stats.AverageCities = float64(totalCities) / float64(count)
			stats.AverageActions = float64(totalActions) / float64(count)
			stats.AverageFinalMoney = float64(totalMoney) / float64(count)
		}

		report.StrategyStats[strategy] = stats
	}
}

func (r *StatsReporter) analyzePhases(report *SimulationReport, results []GameStatistics) {
	phaseTotals := make(map[string]int)
	
	for _, result := range results {
		for phase, count := range result.PhaseCount {
			phaseTotals[phase] += count
		}
	}

	for phase, total := range phaseTotals {
		analysis := &PhaseAnalysis{
			Phase:            phase,
			TotalOccurrences: total,
		}

		if len(results) > 0 {
			analysis.AveragePerGame = float64(total) / float64(len(results))
		}

		report.PhaseAnalysis[phase] = analysis
	}
}

func (r *StatsReporter) analyzeResources(report *SimulationReport, results []GameStatistics) {
	resourceTotals := make(map[string]int)
	resourcePeaks := make(map[string]int)

	for _, result := range results {
		for resource, used := range result.ResourcesUsed {
			resourceTotals[resource] += used
			if used > resourcePeaks[resource] {
				resourcePeaks[resource] = used
			}
		}
	}

	for resource, total := range resourceTotals {
		analysis := &ResourceAnalysis{
			Resource:   resource,
			TotalUsed:  total,
			PeakDemand: resourcePeaks[resource],
		}

		if len(results) > 0 {
			analysis.AveragePerGame = float64(total) / float64(len(results))
		}

		report.ResourceTrends[resource] = analysis
	}
}

func (r *StatsReporter) inferStrategy(playerName string) string {
	// Simple inference based on player name
	// In a real implementation, this would be tracked properly
	if contains(playerName, "aggressive") || contains(playerName, "Aggressive") {
		return "aggressive"
	} else if contains(playerName, "conservative") || contains(playerName, "Conservative") {
		return "conservative"
	} else if contains(playerName, "balanced") || contains(playerName, "Balanced") {
		return "balanced"
	} else if contains(playerName, "random") || contains(playerName, "Random") {
		return "random"
	}
	return "unknown"
}

func (r *StatsReporter) getSortedStrategies(stats map[string]*StrategyStatistics) []string {
	strategies := make([]string, 0, len(stats))
	for strategy := range stats {
		strategies = append(strategies, strategy)
	}
	sort.Slice(strategies, func(i, j int) bool {
		return stats[strategies[i]].WinRate > stats[strategies[j]].WinRate
	})
	return strategies
}

func (r *StatsReporter) getSortedPhases(phases map[string]*PhaseAnalysis) []string {
	phaseList := make([]string, 0, len(phases))
	for phase := range phases {
		phaseList = append(phaseList, phase)
	}
	sort.Strings(phaseList)
	return phaseList
}

func (r *StatsReporter) getSortedResources(resources map[string]*ResourceAnalysis) []string {
	resourceList := make([]string, 0, len(resources))
	for resource := range resources {
		resourceList = append(resourceList, resource)
	}
	sort.Slice(resourceList, func(i, j int) bool {
		return resources[resourceList[i]].TotalUsed > resources[resourceList[j]].TotalUsed
	})
	return resourceList
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// Report structures

type SimulationReport struct {
	Timestamp       time.Time                      `json:"timestamp"`
	TotalGames      int                           `json:"total_games"`
	CompletedGames  int                           `json:"completed_games"`
	FailedGames     int                           `json:"failed_games"`
	CompletionRate  float64                       `json:"completion_rate"`
	AverageDuration time.Duration                 `json:"average_duration"`
	GameResults     []GameStatistics              `json:"game_results"`
	StrategyStats   map[string]*StrategyStatistics `json:"strategy_stats"`
	PhaseAnalysis   map[string]*PhaseAnalysis      `json:"phase_analysis"`
	ResourceTrends  map[string]*ResourceAnalysis   `json:"resource_trends"`
	TopPerformers   []TopPerformer                `json:"top_performers"`
	Insights        []string                      `json:"insights"`
}

type StrategyStatistics struct {
	Strategy          string  `json:"strategy"`
	GamesPlayed       int     `json:"games_played"`
	Wins              int     `json:"wins"`
	WinRate           float64 `json:"win_rate"`
	AverageCities     float64 `json:"average_cities"`
	AveragePlants     float64 `json:"average_plants"`
	AverageActions    float64 `json:"average_actions"`
	AverageFinalMoney float64 `json:"average_final_money"`
}

type PhaseAnalysis struct {
	Phase            string        `json:"phase"`
	TotalOccurrences int           `json:"total_occurrences"`
	AveragePerGame   float64       `json:"average_per_game"`
	AverageDuration  time.Duration `json:"average_duration"`
}

type ResourceAnalysis struct {
	Resource       string  `json:"resource"`
	TotalUsed      int     `json:"total_used"`
	AveragePerGame float64 `json:"average_per_game"`
	PeakDemand     int     `json:"peak_demand"`
	PriceVolatility float64 `json:"price_volatility"`
}

type TopPerformer struct {
	Name     string `json:"name"`
	Strategy string `json:"strategy"`
	Wins     int    `json:"wins"`
	Games    int    `json:"games"`
}