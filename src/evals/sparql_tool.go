package main

import (
	"encoding/json"
	"fmt"

	sparql "github.com/soypete/ontology-go/sparql"
)

// SPARQLTool wraps the ontology-go SPARQL engine as a callable tool.
type SPARQLTool struct {
	engine *sparql.Engine
}

// NewSPARQLTool creates a new SPARQL tool backed by the given engine.
func NewSPARQLTool(engine *sparql.Engine) *SPARQLTool {
	return &SPARQLTool{engine: engine}
}

// ToolDefinition returns the OpenAI-compatible tool definition for the SPARQL tool.
func (t *SPARQLTool) ToolDefinition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": "sparql_query",
			"description": `Execute a SPARQL SELECT query against the music genre ontology loaded in memory.

Available prefixes:
  mg:    <http://thekgguys.bootcamp.ai/genres#>
  skos:  <http://www.w3.org/2004/02/skos/core#>
  rdfs:  <http://www.w3.org/2000/01/rdf-schema#>
  owl:   <http://www.w3.org/2002/07/owl#>
  schema: <https://schema.org/>
  mka:   <http://musicKGartists.com/ontology/artists#>
  mkr:   <http://musicKGartists.com/resource/>
  inst:  <http://www.instrumental.org/2026/03/en/instruments_group#>
  kga:   <http://www.kga.org/us-cohort-two#>

IMPORTANT: This ontology uses TWO namespaces -- mg: for schema (TBox/CBox) and sng: for instance data (ABox).

TBox classes (use mg: prefix):
  mg:MusicGenres           - root class for all genres
  mg:Songs                 - song class
  mg:Playlists             - playlist class
  mg:StreamingPlatforms    - streaming service class
  Genre subclasses: mg:popular-music, mg:reggaeton, mg:k-pop, mg:rock-music, mg:pop-music, etc.

ABox instances (use sng: prefix -- NOT mg:):
  Songs:     sng:Gasolina, sng:Dakiti, sng:Dynamite, sng:Butter, sng:BohemianRhapsody, etc.
  Genres:    sng:Reggaeton, sng:KPop, sng:RockMusic (these have origin, popularity data)
  Playlists: sng:ReggaetonHits, sng:BailaReggaeton, sng:ThisIsBadBunny
  Platform:  sng:Spotify

Properties (always mg: prefix):
  Object: mg:hasGenre, mg:hasArtist, mg:usesInstrument, mg:hasOrigin, mg:isPopularIn,
          mg:availableOn, mg:isRemixOf, mg:hasSong, mg:hasExternalURL
  Data:   mg:hasLanguage, mg:popularDuring, mg:wasReleasedIn, mg:hasTempo

SKOS (CBox -- genre concepts use mg: prefix):
  skos:broader, skos:narrower, skos:prefLabel, skos:altLabel, skos:definition

Example: To find where Reggaeton is popular:
  SELECT ?place WHERE { sng:Reggaeton mg:isPopularIn ?place . }
Example: To find K-Pop songs:
  SELECT ?song ?label WHERE { ?song a mg:Songs . ?song mg:hasGenre mg:k-pop . ?song rdfs:label ?label . }
Example: To find subgenres of Pop Music:
  SELECT ?sub WHERE { mg:pop-music skos:narrower ?sub . }

Use full URIs or PREFIX declarations in your queries.`,
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "A SPARQL SELECT query string",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// Execute runs a SPARQL query and returns the result as a JSON string.
func (t *SPARQLTool) Execute(query string) (string, error) {
	result, err := t.engine.Execute(query)
	if err != nil {
		return "", fmt.Errorf("SPARQL execution error: %w", err)
	}

	if len(result.Bindings) == 0 {
		return `{"bindings": [], "count": 0, "message": "Query executed successfully but returned no results."}`, nil
	}

	out := map[string]any{
		"bindings": result.Bindings,
		"count":    len(result.Bindings),
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON marshal error: %w", err)
	}
	return string(b), nil
}
