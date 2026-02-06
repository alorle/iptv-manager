#!/bin/bash
# scripts/validate-go.sh
# Validates Go backend against Clean Architecture & SOLID principles

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "=============================================="
echo "   Go Backend Validation"
echo "=============================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ERRORS=0

# 1. Format check
echo ""
echo "[1/8] Checking code formatting (gofmt)..."
UNFORMATTED=$(gofmt -l . 2>&1 | grep -v vendor | grep -v '.direnv' || true)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}❌ Code not formatted:${NC}"
    echo "$UNFORMATTED"
    ERRORS=$((ERRORS + 1))
else
    echo -e "${GREEN}✓ All Go files properly formatted${NC}"
fi

# 2. Go vet
echo ""
echo "[2/8] Running go vet..."
if go vet ./... 2>&1; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${RED}❌ go vet found issues${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 3. Linter (golangci-lint if available, otherwise staticcheck)
echo ""
echo "[3/8] Running linter..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run ./... 2>&1; then
        echo -e "${GREEN}✓ golangci-lint passed${NC}"
    else
        echo -e "${RED}❌ golangci-lint found issues${NC}"
        ERRORS=$((ERRORS + 1))
    fi
elif command -v staticcheck &> /dev/null; then
    if staticcheck ./... 2>&1; then
        echo -e "${GREEN}✓ staticcheck passed${NC}"
    else
        echo -e "${RED}❌ staticcheck found issues${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${YELLOW}⚠️  No linter found (install golangci-lint or staticcheck)${NC}"
fi

# 4. Tests
echo ""
echo "[4/8] Running tests..."
if go test ./... 2>&1; then
    echo -e "${GREEN}✓ All tests passed${NC}"
else
    echo -e "${RED}❌ Tests failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 5. Test coverage
echo ""
echo "[5/8] Checking test coverage..."
COVERAGE_OUTPUT=$(go test ./... -cover 2>&1 || true)
echo "$COVERAGE_OUTPUT" | grep -E "coverage:|ok|FAIL" || true
# Extract average coverage (simplified check)
COVERAGE_LINES=$(echo "$COVERAGE_OUTPUT" | grep -oP 'coverage: \K[0-9.]+' || echo "")
if [ -n "$COVERAGE_LINES" ]; then
    AVG_COVERAGE=$(echo "$COVERAGE_LINES" | awk '{sum+=$1; count++} END {if(count>0) print sum/count; else print 0}')
    echo "Average coverage: ${AVG_COVERAGE}%"
    if (( $(echo "$AVG_COVERAGE < 50" | bc -l 2>/dev/null || echo "1") )); then
        echo -e "${YELLOW}⚠️  Coverage below 50% (currently ${AVG_COVERAGE}%)${NC}"
    else
        echo -e "${GREEN}✓ Coverage adequate (${AVG_COVERAGE}%)${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Could not calculate coverage${NC}"
fi

# 6. Circular dependencies check
echo ""
echo "[6/8] Checking for circular dependencies..."
if go list ./... > /dev/null 2>&1; then
    echo -e "${GREEN}✓ No circular dependency issues detected${NC}"
else
    echo -e "${RED}❌ Circular dependency or import issues detected${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 7. go.mod tidiness
echo ""
echo "[7/8] Checking go.mod tidiness..."
cp go.mod go.mod.backup
cp go.sum go.sum.backup 2>/dev/null || true
go mod tidy
if diff go.mod go.mod.backup > /dev/null && diff go.sum go.sum.backup > /dev/null 2>&1; then
    echo -e "${GREEN}✓ go.mod is tidy${NC}"
    rm go.mod.backup go.sum.backup 2>/dev/null || true
else
    echo -e "${RED}❌ go.mod is not tidy (run 'go mod tidy')${NC}"
    mv go.mod.backup go.mod
    mv go.sum.backup go.sum 2>/dev/null || true
    ERRORS=$((ERRORS + 1))
fi

# 8. Architecture checks (basic)
echo ""
echo "[8/8] Checking architecture basics..."
# Check for TODO/FIXME/HACK comments
TODO_COUNT=$(grep -r "TODO\|FIXME\|HACK\|XXX" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v '.direnv' | wc -l || echo "0")
if [ "$TODO_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $TODO_COUNT TODO/FIXME/HACK comments${NC}"
    grep -r "TODO\|FIXME\|HACK\|XXX" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v '.direnv' | head -5 || true
else
    echo -e "${GREEN}✓ No technical debt markers found${NC}"
fi

# Summary
echo ""
echo "=============================================="
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✅ Go validation PASSED${NC}"
    echo "=============================================="
    exit 0
else
    echo -e "${RED}❌ Go validation FAILED with $ERRORS error(s)${NC}"
    echo "=============================================="
    exit 1
fi
