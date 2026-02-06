#!/bin/bash
# scripts/ralph-loop.sh
# Infinite loop for Ralph Wiggum technique

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROMPT_FILE="$SCRIPT_DIR/ralph-prompt.md"
MAX_ITERATIONS=${MAX_ITERATIONS:-1000}
ITERATION=0

cd "$PROJECT_ROOT"

echo "╔═══════════════════════════════════════════════════╗"
echo "║       Ralph Wiggum Clean Architecture Loop        ║"
echo "╚═══════════════════════════════════════════════════╝"
echo ""
echo "Max iterations: $MAX_ITERATIONS"
echo "Prompt file: $PROMPT_FILE"
echo ""

if [ ! -f "$PROMPT_FILE" ]; then
    echo "❌ Prompt file not found: $PROMPT_FILE"
    exit 1
fi

while [ $ITERATION -lt $MAX_ITERATIONS ]; do
    ITERATION=$((ITERATION + 1))

    echo ""
    echo "═══════════════════════════════════════════════════"
    echo "  Iteration $ITERATION"
    echo "═══════════════════════════════════════════════════"
    echo ""

    # Run validation first
    if bash "$SCRIPT_DIR/validate-all.sh"; then
        echo ""
        echo "✅ All validations passed on iteration $ITERATION"
        echo ""
        echo "Checking for completion signal..."

        # Check if last commit or output contains COMPLETE promise
        if git log -1 --pretty=%B 2>/dev/null | grep -q "<promise>COMPLETE</promise>"; then
            echo ""
            echo "╔═══════════════════════════════════════════════════╗"
            echo "║           🎉 RALPH LOOP COMPLETE! 🎉              ║"
            echo "╚═══════════════════════════════════════════════════╝"
            echo ""
            echo "Repository successfully refactored after $ITERATION iterations"
            exit 0
        fi
    fi

    # Feed prompt to AI (this is where you'd integrate with Claude Code or API)
    echo ""
    echo "🤖 Running Claude Code agent..."
    echo ""

    # Run claude with the prompt file and capture output
    OUTPUT=$(claude --dangerously-skip-permissions --print < "$PROMPT_FILE" 2>&1 | tee /dev/stderr) || true
  
    # Check for completion signal
    if echo "$OUTPUT" | grep -q "<promise>COMPLETE</promise>"; then
        echo ""
        echo "╔═══════════════════════════════════════════════════╗"
        echo "║           🎉 RALPH LOOP COMPLETE! 🎉              ║"
        echo "╚═══════════════════════════════════════════════════╝"
        echo ""
        echo "Repository successfully refactored after $ITERATION iterations"
        exit 0
    fi

    sleep 1
done

echo ""
echo "⚠️  Reached maximum iterations ($MAX_ITERATIONS)"
echo "   Consider reviewing progress and continuing manually"
exit 1
