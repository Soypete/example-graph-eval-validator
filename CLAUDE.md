# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a knowledge graph ontology for the **music domain**, written in Turtle (TTL) format. It uses a hybrid SKOS + RDF/RDFS approach: SKOS for controlled vocabulary and labels, RDFS for class hierarchy and schema structure.

## Key File

- `genres-skos-v8.ttl` — The music genre concept scheme (being expanded to include RDFS class typing).

## Namespaces

- `mg:` → `<http://thekgguyes.bootcamp.ai/musicgenres#>` — music genres
- `skos:`, `rdf:`, `rdfs:` — W3C standard vocabularies

## Ontology Design Patterns

- **Dual typing**: Genre concepts are typed as both `skos:Concept` and `rdfs:Class` (e.g., `mg:Reggaeton a skos:Concept, rdfs:Class`)
- **Parallel hierarchies**: Use `skos:broader`/`skos:narrower` for the SKOS taxonomy AND `rdfs:subClassOf` for the RDFS class hierarchy
- **Labels**: Include both `skos:prefLabel` (with language tag) and `rdfs:label`
- **External alignment**: `skos:exactMatch` links to Wikidata entities or musicmap.info
- **Schema.org mappings**: New classes follow schema.org type hierarchy where applicable

## CBox / TBox / ABox Structure

- **CBox**: SKOS metadata — labels, definitions, scope notes, controlled vocabulary
- **TBox**: RDFS classes, properties, relationships — the conceptual schema
- **ABox**: Data instances/individuals (e.g., "Bad Bunny is an artist", a specific song)

## Working with TTL Files

- Turtle syntax: statements end with `.`, property lists use `;` to share the same subject, object lists use `,`
- Validate TTL syntax with a tool like `rapper` (from Raptor RDF library): `rapper -i turtle -c genres-skos-v8.ttl`
- Ensure every concept includes `skos:inScheme` and has both `broader`/`narrower` links maintained bidirectionally
- When adding RDFS, mirror `skos:broader` with `rdfs:subClassOf`
