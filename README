# Intelligence Query Engine

A simple Go REST API that supports natural language processing for querying a database efficiently.

---

This project is my submission for the hng internship stage 2 backend track.
https://airtable.com/appZPpwy4dtvVBWU4/shrMH9P1zv4TPhvns?C3OrT=recCRAOnUwTulDtq6

## 📦 Tech Stack

- Go (net/http)
- UUID for IDs
- errgroup (concurrency handling)
- External APIs:
  - https://api.agify.io
  - https://api.genderize.io
  - https://api.nationalize.io

---

## ▶️ Running the Project Locally

Make sure Go is installed:

```bash
go version
git clone https://github.com/Taterbro/backendStageZero.git
cd backendStageZero
go mod tidy
go run cmd/api/main.go
```

server will start at http://localhost:8080

---

# Features

## 1. Natural Language Query Support

- Accepts a `q` parameter with plain English input (e.g. `"female users above 25 in nigeria"`).
- Parses input into structured filters using regex
- Enables non-technical users to perform searches without knowing query parameters.

---

## 2. Structured Filtering (Overrides / Complements NLP)

Supports direct query parameters:

- `gender`
- `country_id`
- `age_group`
- `min_age`, `max_age`
- `min_gender_probability`
- `min_country_probability`

These parameters can:

- Override NLP-derived filters
- Be used independently without NLP

---

## 3. Dynamic Query Construction

- SQL query is built incrementally based on provided filters.
- Only necessary conditions are included.
- Improves efficiency and flexibility for partial queries.

---

## 4. Pagination Support

- Parameters:
  - `page` (default: 1)
  - `limit` (default: 10, max: 50)
- Uses SQL `LIMIT` and `OFFSET` for efficient pagination.

---

## 5. Sorting & Ordering

Supports sorting by:

- `name`
- `age`
- `created_at`
- `gender_probability`
- `country_probability`

Supports ordering:

- `asc`
- `desc`

- Sorting and ordering are restricted via whitelists to prevent SQL injection.

---

## 6. Input Validation & Error Handling

- Validates numeric inputs (age, probabilities, pagination).
- Returns structured error responses for invalid inputs.
- Prevents malformed queries from reaching the database.

---

## 7. SQL Injection Protection (Partial)

- Uses parameterized queries (`?` placeholders).
- Restricts dynamic SQL parts (sorting/order) to predefined safe values.

---

## 8. Flexible Query Usage

Supports:

- NLP-only queries (`q`)
- Structured query parameters only
- Combination of both

---

# Limitations of NLP Queries

## 1. Rule-Based Parsing

- Relies entirely on predefined parsing rules (No AI/LLMs).
- Implemented with pattern matching or keyword rules.
- Cannot generalize beyond predefined patterns.

---

## 2. No True Language Understanding

- Does not understand context, intent, or semantics.
- Small variations in phrasing may break parsing.

---

## 3. No Fuzzy Matching

- Requires near-exact matches.
- Misspellings or variations are not handled.

---

## 4. No Conflict Resolution

- When both NLP and query parameters are provided:
  - Later values overwrite earlier ones silently.
- No explicit conflict handling logic.

---

## 5. No Complex Query Support

Cannot handle:

- OR conditions (`male OR female`)
- Nested logic (`(age > 20 AND female) OR country = US`)
- Comparative expressions (`closer to 30 than 20`)

---

## 6. Language Limitation

- Supports English only.
- No multilingual capability.

---

## 7. Schema Dependency

- NLP logic is tightly coupled to database fields.
- Any schema changes require updates to the parser.

---

# Summary

This system is a **rule-based natural language to SQL filter translator**.

### Strengths:

- Predictable behavior
- Fast execution
- Safe (controlled inputs and SQL construction)

### Trade-offs:

- Limited flexibility
- Fragile to language variation
- Requires manual updates to expand capabilities
