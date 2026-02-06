#!/bin/bash
# scripts/validate-all.sh
# Master validation script for entire repository

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "╔════════════════════════════════════════════╗"
echo "║  Full Repository Validation                ║"
echo "╚════════════════════════════════════════════╝"
echo ""

ERRORS=0

# Validate Go backend
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Go Backend"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if bash "$SCRIPT_DIR/validate-go.sh"; then
    echo ""
    GO_STATUS="PASSED"
else
    echo ""
    GO_STATUS="FAILED"
    ERRORS=$((ERRORS + 1))
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  React Frontend"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Validate React frontend
if [ -d "web-ui" ]; then
    if bash "$SCRIPT_DIR/validate-react.sh"; then
        echo ""
        REACT_STATUS="PASSED"
    else
        echo ""
        REACT_STATUS="FAILED"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "⚠️  No web-ui directory found, skipping frontend validation"
    REACT_STATUS="SKIPPED"
fi

echo ""
echo "╔════════════════════════════════════════════╗"
echo "║  Validation Summary                        ║"
echo "╠════════════════════════════════════════════╣"
echo "║  Go Backend:     $GO_STATUS"
echo "║  React Frontend: $REACT_STATUS"
echo "╠════════════════════════════════════════════╣"
if [ $ERRORS -eq 0 ]; then
    echo "║  ✅ ALL VALIDATIONS PASSED                 ║"
    echo "╚════════════════════════════════════════════╝"
    exit 0
else
    echo "║  ❌ VALIDATION FAILED ($ERRORS component(s))     ║"
    echo "╚════════════════════════════════════════════╝"
    exit 1
fi
