#!/bin/bash
# Stop bot-service running in background

PID_FILE="bot-service.pid"

# Function to find bot-service processes
find_bot_processes() {
    # Find processes matching bot-service (but not bot-service- which is old pattern)
    pgrep -f "bot-service" | grep -v "bot-service-" | sort
}

# Function to get process command line
get_process_cmd() {
    local pid=$1
    ps -p "$pid" -o command= 2>/dev/null || echo "Unknown"
}

# Try to get PID from:
# 1. Command line argument
# 2. PID file
# 3. Process name search
if [[ $# -ge 1 ]] && [[ "$1" =~ ^[0-9]+$ ]]; then
    # Manual PID specified
    PID=$1
    echo "📌 Using manual PID: $PID"
elif [[ -f "$PID_FILE" ]]; then
    # Use PID file
    PID=$(cat "$PID_FILE")
    echo "📄 Using PID from file: $PID"

    # Check if the PID file process is still running
    if ! ps -p "$PID" > /dev/null 2>&1; then
        echo "⚠️  Process $PID from PID file is not running"
        echo "🔍 Searching for running bot-service processes..."
        rm -f "$PID_FILE"
        PIDS=$(find_bot_processes)
        if [[ -z "$PIDS" ]]; then
            echo "❌ No bot-service processes found"
            exit 1
        fi
        PID=$(echo "$PIDS" | head -n 1)
    fi
else
    # Search by process name
    echo "🔍 No PID file found, searching for bot-service processes..."
    PIDS=$(find_bot_processes)

    if [[ -z "$PIDS" ]]; then
        echo "❌ No bot-service processes found"
        echo "   Is the service running?"
        exit 1
    fi

    PID_COUNT=$(echo "$PIDS" | wc -l | tr -d ' ')

    if [[ $PID_COUNT -eq 1 ]]; then
        PID=$(echo "$PIDS" | head -n 1)
        echo "✅ Found one bot-service process: $PID"
    else
        echo "⚠️  Found multiple bot-service processes:"
        echo ""
        for p in $PIDS; do
            CMD=$(get_process_cmd "$p")
            echo "   [$p] $CMD"
        done
        echo ""
        echo "Please specify which PID to stop:"
        echo "  $0 <pid>"
        exit 1
    fi
fi

# Verify the PID is actually running
if ! ps -p "$PID" > /dev/null 2>&1; then
    echo "❌ Process $PID is not running"
    [[ -f "$PID_FILE" ]] && rm -f "$PID_FILE"
    exit 1
fi

# Kill process
echo "🛑 Stopping bot-service (PID: $PID)..."
kill "$PID"

# Wait for process to terminate
for i in {1..10}; do
    if ! ps -p "$PID" > /dev/null 2>&1; then
        echo "✅ Bot service stopped successfully"
        rm -f "$PID_FILE"
        exit 0
    fi
    sleep 1
done

# Force kill if still running
echo "⚠️  Process did not stop gracefully. Force killing..."
kill -9 "$PID"
rm -f "$PID_FILE"
echo "✅ Bot service force stopped"
