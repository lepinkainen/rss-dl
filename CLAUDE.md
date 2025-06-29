# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Guidelines

This project follows the guidelines defined in `llm-shared/project_tech_stack.md`. Key points:
- Always run `gofmt -w .` after making Go code changes
- Always build the project using the taskfile before finishing a task
- Prefer standard library packages when possible
- Provide justification when adding new third-party dependencies

## Project Overview

RSS-DL is a Go-based RSS feed downloader that fetches RSS feeds and downloads linked files to a specified directory. The application is designed to run as a single-file executable with minimal configuration.

## Core Architecture

- **Single-file application**: All logic is contained in `main.go`
- **Configuration-driven**: Uses YAML configuration file (`config.yaml`) for RSS URL and output directory (note: per `llm-shared/project_tech_stack.md`, consider using `github.com/spf13/viper` for configuration management)
- **RSS parsing**: Parses RSS XML feeds and extracts download links from items
- **File downloading**: Downloads files from extracted links with proper error handling and logging
- **Structured logging**: Uses logrus for formatted logging with timestamps and context (note: per `llm-shared/project_tech_stack.md`, consider migrating to standard library `log/slog` for cron applications or `fmt.Println` for CLI applications)

## Key Components

- `RSS` and `Channel` structs handle RSS XML parsing
- `Config` struct manages YAML configuration loading
- `downloadFile()` function handles HTTP downloads with content-type validation
- `extractFileName()` utility extracts filenames from HTTP responses using Content-Disposition or URL path

## Development Commands

The project uses Taskfile for build automation. Common commands:

- `task` or `task build` - Build the application (runs tests and creates Linux binary)
- `task test` - Run tests with coverage reports (generates HTML coverage in `coverage/`)
- `task lint` - Run golangci-lint for code quality checks
- `task clean` - Remove build artifacts and coverage files
- `task upgrade-deps` - Update all Go dependencies
- `task publish` - Deploy binary to remote server (requires DESTINATION environment variable)

## Configuration

The application expects a `config.yaml` file in the same directory as the executable:
- `rss_url`: URL of the RSS feed to process
- `output_dir`: Directory where downloaded files will be saved

Copy `config-example.yaml` to `config.yaml` and modify as needed.

## Build Details

- Cross-compiles to Linux AMD64 by default
- Uses Git commit hash for version information via ldflags
- Strips symbols for smaller binary size (-w -s flags)
- Requires golangci-lint for linting tasks