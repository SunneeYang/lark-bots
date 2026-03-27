#!/bin/bash
# Start bot-service in background

set -e

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS] <bots>"
    echo ""
    echo "Arguments:"
    echo "  bots            Space-separated list of bot names to start (required)"
    echo ""
    echo "Options:"
    echo "  --all           Start all bots"
    echo "  --bots=LIST     Comma-separated list of bots to start (alternative syntax)"
    echo "  -h, --help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 dispatcher                        Start specific bot"
    echo "  $0 dispatcher audit-executor         Start multiple bots (space-separated)"
    echo "  $0 --bots=dispatcher,executor        Start multiple bots (comma-separated)"
    echo "  $0 --all                             Start all configured bots"
    exit 0
}

# Parse arguments
BOT_ARGS=""
if [[ $# -eq 0 ]]; then
    # No arguments, show usage
    echo "❌ Error: Missing required argument"
    echo ""
    show_usage
elif [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    show_usage
elif [[ "$1" == "--all" ]]; then
    # Explicitly start all bots
    BOT_ARGS="start --all"
elif [[ "$1" == "--bots="* ]]; then
    # --bots=bot1,bot2 format
    BOT_LIST="${1#--bots=}"
    BOT_ARGS="start --bots=$BOT_LIST"
else
    # Space-separated list of bot names (join with comma)
    BOT_LIST=$(echo "$*" | tr ' ' ',')
    BOT_ARGS="start --bots=$BOT_LIST"
fi

# Binary name (same for all platforms)
BINARY="bot-service"

# Check if binary exists
if [[ ! -f "$BINARY" ]]; then
    echo "❌ Error: Binary '$BINARY' not found. Please run ./build.sh first."
    exit 1
fi

# Function to find and display running bot-service processes
check_running_services() {
    # Use pgrep with exact match on process name
    local pids=$(pgrep -x "bot-service" 2>/dev/null | sort)

    if [[ -n "$pids" ]]; then
        echo "⚠️  Found $(echo "$pids" | wc -l) bot-service process(es) already running:"
        echo ""
        for pid in $pids; do
            local cmd=$(ps -p "$pid" -o command= 2>/dev/null | head -c 100)
            local cpu=$(ps -p "$pid" -o %cpu= 2>/dev/null | tr -d ' ')
            local mem=$(ps -p "$pid" -o %mem= 2>/dev/null | tr -d ' ')
            echo "   PID: $pid  CPU: ${cpu}%  MEM: ${mem}%"
            echo "   CMD: $cmd"
            echo ""
        done
        return 0
    fi
    return 1
}

# Check if any bot-service process is already running
echo "🔍 Checking for running services..."
if check_running_services; then
    echo "❌ Cannot start new service while another is running"
    echo ""
    echo "To stop the running service(s):"
    echo "  ./stop.sh              # Stop single process (auto-detected)"
    echo "  ./stop.sh <pid>        # Stop specific PID"
    echo ""
    echo "To view running services:"
    echo "  ./status.sh            # Show all running services"
    exit 1
fi

# Check PID file (for backward compatibility)
PID_FILE="bot-service.pid"
if [[ -f "$PID_FILE" ]]; then
    OLD_PID=$(cat "$PID_FILE")
    if ps -p "$OLD_PID" > /dev/null 2>&1; then
        echo "⚠️  Bot service is already running (PID: $OLD_PID from PID file)"
        echo "   Stop it first with: ./stop.sh"
        exit 1
    else
        echo "🧹 Cleaning up stale PID file..."
        rm -f "$PID_FILE"
    fi
fi

# Create logs directory if not exists
mkdir -p logs

# Start in background
echo "🚀 Starting bot-service in background..."
echo "   Binary: $BINARY"
echo "   Command: $BOT_ARGS"

nohup ./"$BINARY" $BOT_ARGS > logs/bot-service.log 2>&1 &
PID=$!

# Save PID
echo "$PID" > "$PID_FILE"

# Wait for process to initialize
echo "⏳ Waiting for service to initialize..."
sleep 3

# Check if process is still running
if ! ps -p "$PID" > /dev/null 2>&1; then
    echo ""
    echo "❌ Failed to start bot service!"
    echo ""
    echo "Process exited unexpectedly. Check the log file for details:"
    echo "  tail -n 50 logs/bot-service.log"
    echo ""
    echo "Recent log output:"
    echo "----------------------------------------"
    tail -n 30 logs/bot-service.log 2>/dev/null || echo "Log file not found"
    echo "----------------------------------------"
    rm -f "$PID_FILE"
    exit 1
fi

echo "✅ Bot service started successfully!"
echo "   PID: $PID"
echo "   Log file: logs/bot-service.log"
echo ""
echo "View logs: tail -f logs/bot-service.log"
echo "Stop service: ./stop.sh"
