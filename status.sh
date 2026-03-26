#!/bin/bash
# Check bot-service status

PID_FILE="bot-service.pid"

# Function to find bot-service processes
find_bot_processes() {
    pgrep -f "bot-service" | grep -v "bot-service-" | sort
}

# Function to get process command line
get_process_cmd() {
    local pid=$1
    ps -p "$pid" -o command= 2>/dev/null || echo "Unknown"
}

# Function to get process info
get_process_info() {
    local pid=$1
    local cmd=$(ps -p "$pid" -o command= 2>/dev/null | head -c 100)
    local cpu=$(ps -p "$pid" -o %cpu= 2>/dev/null | tr -d ' ')
    local mem=$(ps -p "$pid" -o %mem= 2>/dev/null | tr -d ' ')
    local time=$(ps -p "$pid" -o etime= 2>/dev/null | tr -d ' ')
    echo "   PID: $pid"
    echo "   CPU: ${cpu}%  MEM: ${mem}%  TIME: $time"
    echo "   CMD: $cmd"
}

# Try to get PID from:
# 1. Command line argument
# 2. PID file
# 3. Process name search
if [[ $# -ge 1 ]] && [[ "$1" =~ ^[0-9]+$ ]]; then
    # Manual PID specified
    PID=$1
elif [[ -f "$PID_FILE" ]]; then
    # Use PID file
    PID=$(cat "$PID_FILE")

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
    else
        echo "⚠️  Found multiple bot-service processes:"
        echo ""
        for p in $PIDS; do
            echo "Process $p:"
            get_process_info "$p"
            echo ""
        done
        exit 0
    fi
fi

# Check if process is running
if ps -p "$PID" > /dev/null 2>&1; then
    echo "✅ Bot service is running"
    get_process_info "$PID"
    echo ""
    echo "Recent logs:"
    echo "----------------------------------------"
    tail -n 20 logs/bot-service.log 2>/dev/null || echo "Log file not found"
else
    echo "❌ Bot service is not running"
    [[ -f "$PID_FILE" ]] && echo "   (Stale PID file: $PID_FILE)"
    exit 1
fi
