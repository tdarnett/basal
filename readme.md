# Basal - Insulin Basal Rate Tracker

A command-line tool for tracking insulin basal rates over time. The application stores data in a SQLite database.

## Features

- Interactive basal rate updates with time intervals
- Query basal rates by date
- Natural language queries using local LLM (Ollama)
- Simple command-line interface

## Prerequisites

- Go 1.21 or later
- Ollama (for natural language queries)

## Installation

Build and install:
```bash
go install
```

## Usage

### Update Basal Rates

Create or update basal rates interactively:

```bash
basal update
```

This will prompt you for:
- Date (defaults to today)
- Time intervals (HH:MM format)
- Units per hour for each interval

### Query by Date

View basal rates for a specific date:

```bash
basal date 2025-03-01
```

If no exact record exists for the date, it will show the closest previous record.

### Natural Language Queries

Query your basal rates using natural language:

```bash
basal query "what was my basal rate on Dec 2, 2023?"
basal query "when was I taking the most insulin?"
```

Note: This requires Ollama to be running locally.

### Configure LLM

Configure the LLM endpoint and model for natural language queries:

```bash
basal llm configure
```

### Help

Display help information:

```bash
basal help
```



Future Features:

- graph a record in terminal.

