#!/bin/bash

# Parse command line arguments
FRESH=false
CODEBASE_PATH=""
for arg in "$@"; do
    case $arg in
        --fresh)
        FRESH=true
        shift
        ;;
        --codebase=*)
        CODEBASE_PATH="${arg#*=}"
        shift
        ;;
    esac
done

if [ "$FRESH" = true ]; then
    # Check if codebase path is provided
    if [ -z "$CODEBASE_PATH" ]; then
        echo "Error: Please provide the codebase path using --codebase=<path>"
        echo "Usage: ./run_agents.sh [--fresh] --codebase=<path>"
        exit 1
    fi

    # Check if the codebase directory exists
    if [ ! -d "indexing/$CODEBASE_PATH" ]; then
        echo "Error: Codebase directory '$CODEBASE_PATH' does not exist"
        exit 1
    fi
fi

# Create virtual environment if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
source venv/bin/activate

# Install requirements
echo "Installing requirements..."
pip install -r requirements.txt

# Install local package in development mode
echo "Installing local package in development mode..."
pip install -e .

# Set environment variables
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="password"

if [ "$FRESH" = true ]; then
    echo "Indexing the codebase..."
    cd indexing
    ./index_codebase.sh "$CODEBASE_PATH"
    cd ..
fi

# Run the app
echo "Starting the app..."
uvicorn app:app --reload --host 0.0.0.0 --port 8001 