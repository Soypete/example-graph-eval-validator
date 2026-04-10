package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	sparql "github.com/soypete/ontology-go/sparql"
	"github.com/soypete/ontology-go/store"
	"github.com/soypete/ontology-go/ttl"
)

func main() {
	ttlPath := flag.String("ttl", "../../genre.ttl", "Path to the genre TTL file")
	endpoint := flag.String("endpoint", "http://pedrogpt:8080", "OpenAI-compatible API endpoint (no trailing /v1)")
	flag.Parse()

	// 1. Discover model name
	fmt.Println("Discovering model...")
	modelID, err := discoverModel(*endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to discover model: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Using model: %s\n", modelID)

	// 2. Load TTL file
	fmt.Printf("Loading TTL from %s...\n", *ttlPath)
	parser := ttl.NewTurtleParser()
	parser.Graph = "http://thekgguys.bootcamp.ai/genres"
	triples, err := parser.ParseFile(*ttlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse TTL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d triples\n", len(triples))

	// 3. Store triples in memory
	memStore := store.NewMemoryStore()
	if err := memStore.Register("genres", triples); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register triples: %v\n", err)
		os.Exit(1)
	}

	// 4. Create SPARQL engine
	engine := sparql.NewEngine(memStore)

	// 5. Read TTL file content for the read_ontology tool
	ttlContent, err := os.ReadFile(*ttlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read TTL file: %v\n", err)
		os.Exit(1)
	}

	// 6. Create tools and agent
	sparqlTool := NewSPARQLTool(engine)
	agent := NewAgent(*endpoint, modelID, sparqlTool, string(ttlContent))

	// 7. Run eval
	fmt.Printf("\nStarting eval: %d questions x %d runs\n", len(AllQuestions()), runsPerQuestion)
	results := RunEval(agent, AllQuestions())

	// 8. Print summary
	PrintSummary(results)
}

// modelsResponse represents the /v1/models response.
type modelsResponse struct {
	Data []struct {
		ID     string `json:"id"`
		Status struct {
			Value string `json:"value"`
		} `json:"status"`
	} `json:"data"`
}

// discoverModel calls GET /v1/models and returns the first model ID.
func discoverModel(endpoint string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("%s/v1/models", endpoint)

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var models modelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}

	if len(models.Data) == 0 {
		return "", fmt.Errorf("no models found at %s", url)
	}

	// Prefer a loaded model
	for _, m := range models.Data {
		if m.Status.Value == "loaded" {
			return m.ID, nil
		}
	}

	return models.Data[0].ID, nil
}
