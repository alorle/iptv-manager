#!/bin/bash
# scripts/validate-react.sh
# Validates React frontend for clean Logic/UI separation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_UI_DIR="$(dirname "$SCRIPT_DIR")/web-ui"

if [ ! -d "$WEB_UI_DIR" ]; then
    echo "❌ web-ui directory not found"
    exit 1
fi

cd "$WEB_UI_DIR"

echo "=============================================="
echo "   React Frontend Validation"
echo "=============================================="

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ERRORS=0

# 1. Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo ""
    echo "[0/7] Installing dependencies..."
    npm install
fi

# 2. ESLint
echo ""
echo "[1/7] Running ESLint..."
if npm run lint 2>&1; then
    echo -e "${GREEN}✓ ESLint + Prettier check passed${NC}"
else
    echo -e "${RED}❌ ESLint/Prettier found issues${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 3. TypeScript check
echo ""
echo "[2/7] TypeScript checking..."
if [ -f "tsconfig.json" ]; then
    if npx tsc --noEmit 2>&1; then
        echo -e "${GREEN}✓ TypeScript check passed${NC}"
    else
        echo -e "${RED}❌ TypeScript errors found${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${YELLOW}⚠️  No TypeScript config found${NC}"
fi

# 4. Tests
echo ""
echo "[3/7] Running tests..."
if npm test -- --run 2>&1; then
    echo -e "${GREEN}✓ Tests passed${NC}"
else
    echo -e "${RED}❌ Tests failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 5. Build
echo ""
echo "[4/7] Building application..."
if npm run build 2>&1; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 6. Check for common anti-patterns
echo ""
echo "[5/7] Checking for anti-patterns..."
CONSOLE_LOGS=$(grep -r "console.log" src/ --include="*.ts" --include="*.tsx" 2>/dev/null | wc -l || echo "0")
if [ "$CONSOLE_LOGS" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $CONSOLE_LOGS console.log statements${NC}"
    grep -r "console.log" src/ --include="*.ts" --include="*.tsx" 2>/dev/null | head -5 || true
else
    echo -e "${GREEN}✓ No console.log statements${NC}"
fi

# Check for large components (>100 lines)
echo ""
echo "[6/7] Checking component sizes..."
LARGE_COMPONENTS=""
if [ -d "src/components" ]; then
    while IFS= read -r file; do
        if [ -f "$file" ]; then
            lines=$(wc -l < "$file")
            if [ "$lines" -gt 100 ]; then
                LARGE_COMPONENTS="$LARGE_COMPONENTS$file ($lines lines)\n"
            fi
        fi
    done < <(find src/components -name "*.tsx" 2>/dev/null)
fi

if [ -n "$LARGE_COMPONENTS" ]; then
    echo -e "${YELLOW}⚠️  Large components found (>100 lines):${NC}"
    echo -e "$LARGE_COMPONENTS"
else
    echo -e "${GREEN}✓ All components reasonably sized${NC}"
fi

# 7. Check for TODO/FIXME comments
echo ""
echo "[7/7] Checking for technical debt markers..."
TODO_COUNT=$(grep -r "TODO\|FIXME\|HACK\|XXX" src/ --include="*.ts" --include="*.tsx" 2>/dev/null | wc -l || echo "0")
if [ "$TODO_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $TODO_COUNT TODO/FIXME/HACK comments${NC}"
    grep -r "TODO\|FIXME\|HACK\|XXX" src/ --include="*.ts" --include="*.tsx" 2>/dev/null | head -5 || true
else
    echo -e "${GREEN}✓ No technical debt markers found${NC}"
fi

# Summary
echo ""
echo "=============================================="
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✅ React validation PASSED${NC}"
    echo "=============================================="
    exit 0
else
    echo -e "${RED}❌ React validation FAILED with $ERRORS error(s)${NC}"
    echo "=============================================="
    exit 1
fi
