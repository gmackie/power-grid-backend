package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"powergrid/internal/ai"
	"powergrid/pkg/logger"
)

var (
	serverURL     = flag.String("server", "ws://localhost:4080/ws", "WebSocket server URL")
	numPlayers    = flag.Int("players", 4, "Number of AI players (2-6)")
	strategies    = flag.String("strategies", "balanced,aggressive,conservative,random", "Comma-separated list of strategies")
	gamePrefix    = flag.String("game-prefix", "Simulation", "Prefix for game names")
	thinkTime     = flag.Duration("think-time", 500*time.Millisecond, "AI think time between moves")
	logLevel      = flag.String("log-level", "info", "Log level: debug, info, warn, error")
	showCaller    = flag.Bool("show-caller", false, "Show caller information in logs")
	iterations    = flag.Int("iterations", 1, "Number of game iterations to run")
	concurrent    = flag.Bool("concurrent", false, "Run multiple games concurrently")
	reportStats   = flag.Bool("stats", true, "Generate statistics report")
	maxGameTime   = flag.Duration("max-game-time", 30*time.Minute, "Maximum time per game")
)

type GameResult struct {
	GameID      string
	Duration    time.Duration
	Winner      string
	PlayerStats map[string]PlayerStats
	Completed   bool
	Error       error
	monitor     *ai.GameMonitor
}

type PlayerStats struct {
	Strategy     string
	FinalMoney   int
	FinalCities  int
	FinalPlants  int
	TotalMoves   int
}

type SimulationStats struct {
	TotalGames      int
	CompletedGames  int
	FailedGames     int
	AverageDuration time.Duration
	StrategyWins    map[string]int
	StrategyStats   map[string][]PlayerStats
	gameResults     []GameResult
}

func main() {
	flag.Parse()

	// Validate parameters
	if *numPlayers < 2 || *numPlayers > 6 {
		fmt.Printf("Number of players must be between 2 and 6\n")
		os.Exit(1)
	}

	// Parse log level
	var level logger.LogLevel
	switch *logLevel {
	case "debug":
		level = logger.DEBUG
	case "info":
		level = logger.INFO
	case "warn":
		level = logger.WARN
	case "error":
		level = logger.ERROR
	default:
		level = logger.INFO
	}

	// Initialize loggers
	logger.InitLoggers(level, *showCaller)

	logger.TestLogger.Info("Starting Power Grid simulation")
	logger.TestLogger.Info("Players: %d, Iterations: %d, Concurrent: %v", *numPlayers, *iterations, *concurrent)

	// Handle shutdown gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx := &SimulationContext{
		results: make(chan GameResult, *iterations),
		stats:   &SimulationStats{
			StrategyWins:  make(map[string]int),
			StrategyStats: make(map[string][]PlayerStats),
			gameResults:   make([]GameResult, 0),
		},
	}

	go func() {
		<-c
		logger.TestLogger.Info("Shutting down simulation...")
		os.Exit(0)
	}()

	// Run simulations
	if *concurrent {
		runConcurrentSimulations(ctx)
	} else {
		runSequentialSimulations(ctx)
	}

	// Generate report if requested
	if *reportStats {
		generateReport(ctx.stats)
	}

	logger.TestLogger.Info("Simulation completed")
}

type SimulationContext struct {
	results chan GameResult
	stats   *SimulationStats
	mu      sync.RWMutex
}

func runSequentialSimulations(ctx *SimulationContext) {
	for i := 0; i < *iterations; i++ {
		logger.TestLogger.Info("Starting game %d/%d", i+1, *iterations)
		
		result := runSingleGame(fmt.Sprintf("%s_%d", *gamePrefix, i+1))
		ctx.results <- result
		
		updateStats(ctx, result)
		
		if !result.Completed {
			logger.TestLogger.Error("Game %d failed: %v", i+1, result.Error)
		} else {
			logger.TestLogger.Info("Game %d completed in %v, winner: %s", 
				i+1, result.Duration, result.Winner)
		}
		
		// Brief pause between games
		time.Sleep(2 * time.Second)
	}
	close(ctx.results)
}

func runConcurrentSimulations(ctx *SimulationContext) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Limit concurrent games
	
	for i := 0; i < *iterations; i++ {
		wg.Add(1)
		go func(gameNum int) {
			defer wg.Done()
			
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release
			
			logger.TestLogger.Info("Starting concurrent game %d/%d", gameNum+1, *iterations)
			
			result := runSingleGame(fmt.Sprintf("%s_concurrent_%d", *gamePrefix, gameNum+1))
			ctx.results <- result
			
			updateStats(ctx, result)
			
			if !result.Completed {
				logger.TestLogger.Error("Game %d failed: %v", gameNum+1, result.Error)
			} else {
				logger.TestLogger.Info("Game %d completed in %v, winner: %s", 
					gameNum+1, result.Duration, result.Winner)
			}
		}(i)
	}
	
	go func() {
		wg.Wait()
		close(ctx.results)
	}()
}

func runSingleGame(gameName string) GameResult {
	startTime := time.Now()
	
	// Create game monitor
	monitor := ai.NewGameMonitor(gameName)
	
	result := GameResult{
		GameID:      gameName,
		PlayerStats: make(map[string]PlayerStats),
		monitor:     monitor,
	}
	
	// Set completion callback
	completionChan := make(chan *ai.GameMonitor, 1)
	monitor.SetCompletionCallback(func(m *ai.GameMonitor) {
		completionChan <- m
	})
	
	// Create AI clients
	strategyList := parseStrategies(*strategies)
	clients := make([]*ai.MonitoredAIClient, 0, *numPlayers)
	
	defer func() {
		// Cleanup clients
		for _, client := range clients {
			if client != nil {
				client.Disconnect()
			}
		}
	}()
	
	// Create and connect clients
	for i := 0; i < *numPlayers; i++ {
		strategy := strategyList[i%len(strategyList)]
		playerName := fmt.Sprintf("%s_Player_%d", strategy, i+1)
		
		config := ai.ClientConfig{
			ServerURL:   *serverURL,
			Strategy:    strategy,
			PlayerName:  playerName,
			PlayerColor: getPlayerColor(i),
			AutoPlay:    true,
			ThinkTime:   *thinkTime,
			Interactive: false,
		}
		
		client, err := ai.NewMonitoredAIClient(config, monitor)
		if err != nil {
			result.Error = fmt.Errorf("failed to create client %d: %w", i, err)
			return result
		}
		
		clients = append(clients, client)
		
		if err := client.Connect(); err != nil {
			result.Error = fmt.Errorf("failed to connect client %d: %w", i, err)
			return result
		}
		
		// Stagger connections slightly
		time.Sleep(100 * time.Millisecond)
	}
	
	// Wait for game to complete or timeout
	gameTimer := time.NewTimer(*maxGameTime)
	defer gameTimer.Stop()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	logger.TestLogger.Info("Waiting for game %s to complete...", gameName)
	
	select {
	case <-gameTimer.C:
		result.Error = fmt.Errorf("game timed out after %v", *maxGameTime)
		result.Duration = time.Since(startTime)
		return result
		
	case completedMonitor := <-completionChan:
		// Game completed!
		stats := completedMonitor.GetStatistics()
		
		result.Completed = true
		result.Duration = stats.Duration
		result.Winner = stats.Winner
		
		// Populate player statistics
		for playerID, actions := range stats.PlayerActions {
			result.PlayerStats[playerID] = PlayerStats{
				Strategy:     "unknown", // Would need to track this
				FinalCities:  stats.CitiesBuilt[playerID],
				TotalMoves:   actions,
			}
		}
		
		logger.TestLogger.Info("Game %s completed in %v, winner: %s", 
			gameName, result.Duration, result.Winner)
		
		return result
		
	case <-ticker.C:
		// Periodic status check
		logger.TestLogger.Debug("Game %s still running after %v", 
			gameName, time.Since(startTime))
		
		// Check monitor status
		if monitor.IsCompleted() {
			stats := monitor.GetStatistics()
			result.Completed = true
			result.Duration = stats.Duration
			result.Winner = stats.Winner
			return result
		}
	}
	
	// Continue waiting in a loop
	for {
		select {
		case <-gameTimer.C:
			result.Error = fmt.Errorf("game timed out after %v", *maxGameTime)
			result.Duration = time.Since(startTime)
			return result
			
		case completedMonitor := <-completionChan:
			stats := completedMonitor.GetStatistics()
			result.Completed = true
			result.Duration = stats.Duration
			result.Winner = stats.Winner
			return result
			
		case <-ticker.C:
			logger.TestLogger.Debug("Game %s still running after %v", 
				gameName, time.Since(startTime))
		}
	}
}

func parseStrategies(strategiesStr string) []string {
	// Parse comma-separated strategies
	if strategiesStr == "" {
		return []string{"balanced", "aggressive", "conservative", "random"}
	}
	
	strategies := []string{}
	for _, s := range strings.Split(strategiesStr, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			strategies = append(strategies, trimmed)
		}
	}
	
	if len(strategies) == 0 {
		return []string{"balanced"}
	}
	
	return strategies
}

func getPlayerColor(index int) string {
	colors := []string{"red", "blue", "green", "yellow", "purple", "orange"}
	return colors[index%len(colors)]
}

func updateStats(ctx *SimulationContext, result GameResult) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	
	ctx.stats.TotalGames++
	ctx.stats.gameResults = append(ctx.stats.gameResults, result)
	
	if result.Completed {
		ctx.stats.CompletedGames++
		
		// Update average duration
		if ctx.stats.AverageDuration == 0 {
			ctx.stats.AverageDuration = result.Duration
		} else {
			ctx.stats.AverageDuration = (ctx.stats.AverageDuration + result.Duration) / 2
		}
		
		// Track strategy wins
		if result.Winner != "" {
			// Extract strategy from winner name (simplified)
			ctx.stats.StrategyWins["unknown"]++
		}
	} else {
		ctx.stats.FailedGames++
	}
}

func generateReport(stats *SimulationStats) {
	// Convert to game statistics for reporter
	gameStats := make([]ai.GameStatistics, len(stats.gameResults))
	for i, result := range stats.gameResults {
		monitor := result.monitor
		if monitor != nil {
			gameStats[i] = monitor.GetStatistics()
		}
	}
	
	// Create reporter and generate report
	reporter := ai.NewStatsReporter(logger.TestLogger)
	report := reporter.GenerateReport(gameStats)
	
	// Add insights
	report.Insights = generateInsights(report)
	
	// Print report
	reporter.PrintReport(report)
	
	// Save to file
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("logs/simulation_report_%s.json", timestamp)
	if err := reporter.SaveReport(report, filename); err != nil {
		logger.TestLogger.Error("Failed to save report: %v", err)
	}
}

func generateInsights(report *ai.SimulationReport) []string {
	insights := []string{}
	
	// Find dominant strategy
	var bestStrategy string
	var bestWinRate float64
	for strategy, stats := range report.StrategyStats {
		if stats.WinRate > bestWinRate {
			bestWinRate = stats.WinRate
			bestStrategy = strategy
		}
	}
	
	if bestStrategy != "" {
		insights = append(insights, fmt.Sprintf("%s strategy dominated with %.1f%% win rate", 
			bestStrategy, bestWinRate))
	}
	
	// Check completion rate
	if report.CompletionRate < 90 {
		insights = append(insights, fmt.Sprintf("Low completion rate (%.1f%%) suggests potential stability issues", 
			report.CompletionRate))
	}
	
	// Resource analysis
	var mostUsedResource string
	var maxUsage int
	for resource, analysis := range report.ResourceTrends {
		if analysis.TotalUsed > maxUsage {
			maxUsage = analysis.TotalUsed
			mostUsedResource = resource
		}
	}
	
	if mostUsedResource != "" {
		insights = append(insights, fmt.Sprintf("%s was the most consumed resource (%d total)", 
			mostUsedResource, maxUsage))
	}
	
	// Game duration
	if report.AverageDuration > 30*time.Minute {
		insights = append(insights, "Games are taking longer than expected, consider optimizing AI decision times")
	} else if report.AverageDuration < 10*time.Minute {
		insights = append(insights, "Games are completing very quickly, AI might be too aggressive")
	}
	
	return insights
}