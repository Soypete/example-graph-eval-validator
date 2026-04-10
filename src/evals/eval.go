package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const runsPerQuestion = 5

// RunResult holds the results from all runs of a single question.
type RunResult struct {
	Question Question
	Runs     []AgentResult
}

// LogEntry is one line in the JSONL log file.
type LogEntry struct {
	Timestamp        string        `json:"timestamp"`
	QuestionID       string        `json:"question_id"`
	Question         string        `json:"question"`
	RunNumber        int           `json:"run"`
	Status           string        `json:"status"`
	ToolCalled       bool          `json:"tool_called"`
	QueryParsed      bool          `json:"query_parsed"`
	DataReturned     bool          `json:"data_returned"`
	QuestionAnswered bool          `json:"question_answered"`
	TurnsUsed        int           `json:"turns_used"`
	QueriesAttempted int           `json:"queries_attempted"`
	LatencyMS        int64         `json:"latency_ms"`
	FinalAnswer      string        `json:"final_answer"`
	ToolCalls        []ToolCallLog `json:"tool_calls"`
	Error            string        `json:"error,omitempty"`
}

// RunEval executes the full evaluation: 9 questions x 5 runs each.
// Writes detailed JSONL logs to a timestamped file.
func RunEval(agent *Agent, questions []Question) []RunResult {
	results := make([]RunResult, 0, len(questions))

	// Open log file
	logName := fmt.Sprintf("eval_%s.log", time.Now().Format("20060102_150405"))
	logFile, err := os.Create(logName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create log file %s: %v\n", logName, err)
	} else {
		fmt.Printf("Logging to %s\n", logName)
		defer logFile.Close()
	}

	for i, q := range questions {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(questions), q.ID)
		fmt.Printf("  Question: %s\n", q.Text)

		rr := RunResult{Question: q}

		for run := 0; run < runsPerQuestion; run++ {
			fmt.Printf("  Run %d/%d ... ", run+1, runsPerQuestion)
			result := agent.Run(q.Text)

			status := "PASS"
			if result.Error != "" {
				status = "ERROR"
			} else if !result.QuestionAnswered {
				status = "NO_ANSWER"
			}
			fmt.Printf("%s (turns=%d, queries=%d, tool=%v, parsed=%v, data=%v, latency=%dms)\n",
				status, result.TurnsUsed, result.QueriesAttempted,
				result.ToolCalled, result.QueryParsed, result.DataReturned, result.LatencyMS)

			if result.Error != "" {
				fmt.Printf("    Error: %s\n", result.Error)
			}

			// Write log entry
			if logFile != nil {
				entry := LogEntry{
					Timestamp:        time.Now().Format(time.RFC3339),
					QuestionID:       q.ID,
					Question:         q.Text,
					RunNumber:        run + 1,
					Status:           status,
					ToolCalled:       result.ToolCalled,
					QueryParsed:      result.QueryParsed,
					DataReturned:     result.DataReturned,
					QuestionAnswered: result.QuestionAnswered,
					TurnsUsed:        result.TurnsUsed,
					QueriesAttempted: result.QueriesAttempted,
					LatencyMS:        result.LatencyMS,
					FinalAnswer:      result.FinalAnswer,
					ToolCalls:        result.ToolCalls,
					Error:            result.Error,
				}
				if b, err := json.Marshal(entry); err == nil {
					logFile.Write(b)
					logFile.WriteString("\n")
				}
			}

			rr.Runs = append(rr.Runs, result)
		}

		results = append(results, rr)
	}

	return results
}

// PrintSummary prints a summary table of eval results.
func PrintSummary(results []RunResult) {
	fmt.Println("\n" + strings.Repeat("=", 110))
	fmt.Println("EVALUATION SUMMARY")
	fmt.Println(strings.Repeat("=", 110))

	fmt.Printf("%-25s | %5s | %5s | %5s | %5s | %5s | %s\n",
		"Question", "Runs", "Tool%", "Parse%", "Data%", "Ans%", "HasData")
	fmt.Println(strings.Repeat("-", 110))

	totalRuns := 0
	totalToolCalled := 0
	totalQueryParsed := 0
	totalDataReturned := 0
	totalAnswered := 0

	for _, rr := range results {
		n := len(rr.Runs)
		toolCalled := 0
		queryParsed := 0
		dataReturned := 0
		answered := 0

		for _, r := range rr.Runs {
			if r.ToolCalled {
				toolCalled++
			}
			if r.QueryParsed {
				queryParsed++
			}
			if r.DataReturned {
				dataReturned++
			}
			if r.QuestionAnswered {
				answered++
			}
		}

		totalRuns += n
		totalToolCalled += toolCalled
		totalQueryParsed += queryParsed
		totalDataReturned += dataReturned
		totalAnswered += answered

		hasData := "yes"
		if !rr.Question.HasData {
			hasData = "no*"
		}

		fmt.Printf("%-25s | %5d | %4.0f%% | %4.0f%% | %4.0f%% | %4.0f%% | %s\n",
			rr.Question.ID,
			n,
			pct(toolCalled, n),
			pct(queryParsed, n),
			pct(dataReturned, n),
			pct(answered, n),
			hasData,
		)
	}

	fmt.Println(strings.Repeat("-", 110))
	fmt.Printf("%-25s | %5d | %4.0f%% | %4.0f%% | %4.0f%% | %4.0f%% |\n",
		"TOTAL",
		totalRuns,
		pct(totalToolCalled, totalRuns),
		pct(totalQueryParsed, totalRuns),
		pct(totalDataReturned, totalRuns),
		pct(totalAnswered, totalRuns),
	)
	fmt.Println(strings.Repeat("=", 110))
	fmt.Println("* HasData=no means genre.ttl doesn't contain data for this question (valid if model reports 'no data').")
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
