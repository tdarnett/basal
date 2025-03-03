# Basal Tracker

A simple command-line tool for tracking insulin basal rates over time. Built with Go, it helps you maintain a history of your basal rate adjustments and analyze changes.

## Features

- Track daily basal rate schedules
- View historical basal rate changes
- Calculate daily insulin totals
- Simple SQLite storage
- Easy-to-use command-line interface

## Installation

Build and install:
```bash
go install
```

## Usage

### Add Basal Rates

Add a new basal rate record interactively:

```bash
basal add
```

This will prompt you for:
- Date (defaults to today)
- Time intervals (supports HH:MM, H:MM, or HHMM formats)
- Units per hour for each interval

Note: Intervals must be continuous (end time of one interval is start time of next) and cover the full 24 hours (00:00 to 00:00).

### List Records

View all basal rate records:

```bash
basal list
```

### Show Basal Rates

View basal rates for a specific date:

```bash
basal show 2025-03-01
```

If no exact record exists for the date, it will show the closest previous record.

### Delete Records

Delete a basal rate record by its ID:

```bash
basal delete 123
```

### Natural Language Queries

Ask questions about your basal rates using natural language:

```bash
basal ask "what was my basal rate on Dec 2, 2023?"
basal ask "when was I taking the most insulin?"
```

Note: This requires Ollama to be running locally.

### Configure Database

Set the database location:

```bash
basal config db
```

### Configure LLM

Configure the LLM settings for natural language queries:

```bash
basal config llm
```

### Help

Display help information:

```bash
basal help
```

## Future Features

- Graph a record in terminal
- Export/import functionality
- Statistical analysis of basal patterns

