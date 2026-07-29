# Graph Report - .  (2026-07-30)

## Corpus Check
- Corpus is ~3,446 words - fits in a single context window. You may not need a graph.

## Summary
- 81 nodes · 125 edges · 8 communities
- Extraction: 79% EXTRACTED · 21% INFERRED · 0% AMBIGUOUS · INFERRED: 26 edges (avg confidence: 0.8)
- Token cost: 0 input · 84,870 output

## Community Hubs (Navigation)
- Wallet Graph & On-chain Entities
- Risk Scoring & Detection
- Trading Engine & Execution
- Backtesting & Trading Modes
- Design System & Discovery
- Product Lifecycle Loop
- RAG/CAG, Telegram & Services
- Strategy Signals & Overview

## God Nodes (most connected - your core abstractions)
1. `Wallet Graph / On-chain Entity Graph` - 21 edges
2. `Trading Engine` - 15 edges
3. `Creator Reputation / Trust Score` - 9 edges
4. `Backtesting Framework` - 9 edges
5. `MVP Scope (Phase 1)` - 8 edges
6. `Opportunity Score` - 7 edges
7. `Token Detail (Screen 3)` - 7 edges
8. `New Token Discovery Engine` - 6 edges
9. `Strategy Contract (evaluate/Decision)` - 6 edges
10. `Token Safety Score` - 5 edges

## Surprising Connections (you probably didn't know these)
- `Score Components` --references--> `Opportunity Score`  [INFERRED]
  docs/design/sentinel-ui-ux-design.md → ROADMAP.md
- `Live Feed (Screen 2)` --references--> `New Token Discovery Engine`  [INFERRED]
  docs/design/sentinel-ui-ux-design.md → ROADMAP.md
- `Portfolio and Positions (Screen 8)` --conceptually_related_to--> `Trading Engine`  [INFERRED]
  docs/design/sentinel-ui-ux-design.md → ROADMAP.md
- `Trading Terminal (Screen 7)` --references--> `Trading Engine`  [INFERRED]
  docs/design/sentinel-ui-ux-design.md → ROADMAP.md
- `Trading Terminal (Screen 7)` --references--> `Pre-trade Safety Layer`  [INFERRED]
  docs/design/sentinel-ui-ux-design.md → ROADMAP.md

## Hyperedges (group relationships)
- **Sentinel Product Lifecycle** — concept_discover, concept_enrich, concept_score_stage, concept_alert_stage, concept_decide, concept_execute, concept_observe, concept_learn [EXTRACTED 1.00]
- **Sentinel Multi-Score System** — concept_creator_reputation_score, concept_token_safety_score, concept_market_quality_score, concept_momentum_score, concept_manipulation_risk_score, concept_execution_risk_score, concept_opportunity_score [EXTRACTED 1.00]
- **On-chain Entity Graph Schema** — concept_wallet, concept_token, concept_liquidity_pool, concept_program, concept_launchpad, concept_funding_source, concept_edge_funded, concept_edge_created, concept_edge_provided_liquidity [EXTRACTED 1.00]

## Communities (8 total, 0 thin omitted)

### Community 0 - "Wallet Graph & On-chain Entities"
Cohesion: 0.15
Nodes (15): CREATED (Graph Edge Type), FUNDED (Graph Edge Type), PROVIDED_LIQUIDITY (Graph Edge Type), SHARES_FUNDER_WITH (Graph Edge Type), Funding Source (Graph Node Type), Launchpad (Graph Node Type), Liquidity Pool (Graph Node Type), Program (Graph Node Type) (+7 more)

### Community 1 - "Risk Scoring & Detection"
Cohesion: 0.24
Nodes (13): Behavioral Risks, Creator Profile (Screen 4), Creator Reputation / Trust Score, Manipulation Risk Score, Market Quality Score, MVP Scope (Phase 1), Opportunity Score, Phase 3 (Later) (+5 more)

### Community 2 - "Trading Engine & Execution"
Cohesion: 0.21
Nodes (13): Circuit Breakers, Execution Risk Score, Jupiter Swap API, Phase 2 (Later), Portfolio and Positions (Screen 8), Stop-loss Order, Strategy Platform / Strategies, System Health (Screen 12) (+5 more)

### Community 3 - "Backtesting & Trading Modes"
Cohesion: 0.31
Nodes (10): Backtesting Framework, Emergency Close, Event Replay, Live Trading, Look-ahead Bias Prevention, Loop Engineering, Paper Trading, Security UX (+2 more)

### Community 4 - "Design System & Discovery"
Cohesion: 0.20
Nodes (10): Sentinel Visual Design Language, Live Feed (Screen 2), Sentinel Platform, Token (Graph Node Type), New Token Discovery Engine, Token Intelligence Profile (TokenProfile), Sentinel Dark Theme Color Tokens, Sentinel Component System (+2 more)

### Community 5 - "Product Lifecycle Loop"
Cohesion: 0.22
Nodes (9): Alert (Lifecycle Stage), Decide / Trade (Lifecycle Stage), Discover (Lifecycle Stage), Enrich / Analyze (Lifecycle Stage), Execute (Lifecycle Stage), Learn (Lifecycle Stage), Monitor (Lifecycle Stage), Observe (Lifecycle Stage) (+1 more)

### Community 6 - "RAG/CAG, Telegram & Services"
Cohesion: 0.33
Nodes (6): Alerts (Screen 10), CAG (Cache-Augmented Generation), Go Services (Ingestion / Low-latency), RAG (Retrieval-Augmented Generation), Research Assistant (Screen 11), Telegram Bot / Operations Interface

### Community 7 - "Strategy Signals & Overview"
Cohesion: 0.40
Nodes (5): Decision Object, Momentum Score, Opportunity Radar, Overview Dashboard (Screen 1), Strategy Contract (evaluate/Decision)

## Knowledge Gaps
- **23 isolated node(s):** `Discover (Lifecycle Stage)`, `Monitor (Lifecycle Stage)`, `Learn (Lifecycle Stage)`, `Market Quality Score`, `Execution Risk Score` (+18 more)
  These have ≤1 connection - possible missing edges or undocumented components.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Wallet Graph / On-chain Entity Graph` connect `Wallet Graph & On-chain Entities` to `Risk Scoring & Detection`, `Trading Engine & Execution`, `Design System & Discovery`, `RAG/CAG, Telegram & Services`, `Strategy Signals & Overview`?**
  _High betweenness centrality (0.339) - this node is a cross-community bridge._
- **Why does `Trading Engine` connect `Trading Engine & Execution` to `Backtesting & Trading Modes`, `Design System & Discovery`, `RAG/CAG, Telegram & Services`?**
  _High betweenness centrality (0.202) - this node is a cross-community bridge._
- **Why does `MVP Scope (Phase 1)` connect `Risk Scoring & Detection` to `Wallet Graph & On-chain Entities`, `Backtesting & Trading Modes`, `Design System & Discovery`, `RAG/CAG, Telegram & Services`?**
  _High betweenness centrality (0.116) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `Wallet Graph / On-chain Entity Graph` (e.g. with `CAG (Cache-Augmented Generation)` and `Creator Reputation / Trust Score`) actually correct?**
  _`Wallet Graph / On-chain Entity Graph` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 5 inferred relationships involving `Trading Engine` (e.g. with `Execution Risk Score` and `Go Services (Ingestion / Low-latency)`) actually correct?**
  _`Trading Engine` has 5 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Discover (Lifecycle Stage)`, `Monitor (Lifecycle Stage)`, `Learn (Lifecycle Stage)` to the rest of the system?**
  _23 weakly-connected nodes found - possible documentation gaps or missing edges._