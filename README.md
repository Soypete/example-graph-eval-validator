# Example Graph Eval Validator

An example of testing whether an LLM can use tool calling to query an RDF knowledge graph via SPARQL. Built as a companion to [Unit Testing Your Agents](https://open.substack.com/pub/soypetetech/p/unit-testing-your-agents?utm_campaign=post-expanded-share&utm_medium=web).

## Setup

This project has three components:

### 1. Ontology (`genre.ttl`)

A music genre knowledge graph written in Turtle (TTL) using a hybrid SKOS + RDFS/OWL approach:

- **TBox** (schema): Class definitions (`mg:Songs`, `mg:MusicGenres`, `mg:Playlists`), object/data properties, genre class hierarchy via `rdfs:subClassOf`
- **CBox** (vocabulary): SKOS ConceptScheme with `skos:broader`/`skos:narrower` taxonomy, labels, definitions, Wikidata alignments
- **ABox** (data): Song instances, playlists, genre characteristics — all in a separate `sng:` namespace

The ontology uses multiple namespaces to separate concerns:
- `mg:` — schema (classes, properties, concept scheme)
- `sng:` — instance data (songs, playlists, platforms)
- `mkr:` — artist instances (from [musicKGartists](http://musicKGartists.com))
- `inst:` — instrument instances

### 2. SPARQL Engine ([ontology-go](https://github.com/Soypete/ontology-go))

A pure-Go RDF library that provides:
- **TTL parser**: Loads `.ttl` files into `[]types.Triple`
- **In-memory triple store**: `MemoryStore` with indexed pattern matching
- **SPARQL engine**: Supports SELECT, WHERE, OPTIONAL, FILTER, GROUP BY, aggregates, LIMIT/OFFSET
- **SKOS inference**: Optional broader/narrower/related transitive inference
- **SKOS validator**: Checks for hierarchy consistency, label conflicts, missing metadata

The eval loads `genre.ttl` into memory and exposes it to the LLM as a `sparql_query` tool.

### 3. Locally Hosted LLM

The eval targets an **OpenAI-compatible API** served locally via [llama.cpp](https://github.com/ggerganov/llama.cpp). The eval discovers the model dynamically by calling `GET /v1/models` at startup — no model name is hardcoded.

Tested with:
- **gpt-oss-20b** (Q4_K_M, 32k context) — a local 20B parameter model

Any model that supports [tool/function calling](https://platform.openai.com/docs/guides/function-calling) via the OpenAI chat completions API will work.

## How the Eval Works

The eval gives the LLM two tools:
1. **`read_ontology`** — Returns the TBox (schema) portion of the TTL file so the model can learn the class structure, properties, and namespaces
2. **`sparql_query`** — Executes a SPARQL SELECT query against the in-memory graph and returns JSON results

For each of 9 domain questions, the eval runs a [ReAct loop](https://arxiv.org/abs/2210.03629) (max 5 turns):
1. Send the question to the LLM with both tool definitions
2. If the LLM calls a tool, execute it and return the result
3. Repeat until the LLM gives a final text answer or runs out of turns

Each question is run **5 times** to measure reliability. The eval scores each run on:
- **Tool called**: Did the model invoke at least one tool?
- **Query parsed**: Did the SPARQL execute without error?
- **Data returned**: Did the query return non-empty results?
- **Question answered**: Did the model give a substantive final answer?

Results are printed as a summary table and logged to a JSONL file with full details (every SPARQL query, every result, every answer).

### The 9 Questions

1. Where is this genre most popular?
2. Where can I go to listen to this genre? (locations/platforms)
3. What are the top songs in this genre?
4. What are the main audience demographics? (no data — tests graceful handling)
5. Who are the main artists affiliated with the genre?
6. What are the main characteristics of this genre?
7. What types of instruments are used in this genre?
8. What are defining cultural moments? (no data — tests graceful handling)
9. What genres are related/similar?

## Usage

### Prerequisites

- Go 1.22+
- [ontology-go](https://github.com/Soypete/ontology-go) cloned locally at `~/code/misc/ontology-parser`
- An OpenAI-compatible LLM endpoint (default: `http://pedrogpt:8080`)

### Commands

```bash
# Validate the ontology
make validate

# Run the full eval (9 questions x 5 runs)
make eval

# Run eval against a different endpoint
make eval ENDPOINT=http://localhost:8080

# Show only validation errors
make validate-errors

# Clean build artifacts and logs
make clean
```

### Example Output

```
EVALUATION SUMMARY
==============================================================================
Question                  |  Runs | Tool% | Parse% | Data% |  Ans% | HasData
------------------------------------------------------------------------------
q1_popular_where          |     5 |  100% |   80% |   60% |  100% | yes
q2_listen_locations       |     5 |  100% |   60% |   60% |  100% | yes
q3_top_songs              |     5 |  100% |   80% |   80% |   60% | yes
...
```

## Project Structure

```
genre.ttl              # The music genre ontology (TBox + CBox + ABox)
Makefile               # validate, eval, build, clean targets
CLAUDE.md              # Coding assistant instructions
src/evals/
  main.go              # Entry point: load TTL, discover model, run eval
  agent.go             # ReAct agent loop with OpenAI-compatible tool calling
  sparql_tool.go       # SPARQL tool wrapper around ontology-go engine
  eval.go              # Eval runner: 5 runs per question, scoring, JSONL logging
  questions.go         # The 9 eval questions
  go.mod               # Go module (depends on ontology-go)
```
