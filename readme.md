# Basal Tracker

A simple command-line tool for tracking insulin basal rates over time. Store your basal rate history, view changes with graphs, and find information using natural language queries. All data stays on your local machine.

## Features

## Key Features

- **Comprehensive Tracking**: Record and manage daily basal rate schedules
- **Natural Language Queries**: Ask questions about your data in plain English
- **Historical Analysis**: View and compare basal rate changes over time
- **Insulin Calculations**: Automatically calculate daily insulin totals
- **User-Friendly Interface**: Intuitive command-line experience with interactive prompts
- **Privacy-Focused**: All your health data stays local on your machine

## Quick Start

### Installation

```bash
go install
```

### Basic Commands

```bash
basal add    # Add a new basal rate record
basal list   # View all records
basal show   # Display rates for a specific date
basal ask    # Query your data using natural language
basal help   # Display help information
```

## Detailed Usage


Get detailed historical information for a specific date:

```bash
basal show 2025-03-01
```

### AI-Powered Natural Language Queries

Ask questions about your basal rates in plain English:

```bash
basal ask "what was my total daily basal insulin on December 2, 2023?"
basal ask "when was I taking the most insulin?"
basal ask "how have my basal rates changed over the past year?"
basal ask "what time of day do I typically have my highest basal rate?"
```

### Configuration Options

Customize your database location:

```bash
basal config db
```

Configure the LLM settings for natural language processing:

```bash
basal config llm
```

