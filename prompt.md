# Repository Setup for Ralph Wiggum Clean Architecture Loop

You are preparing this repository for automated technical debt resolution using the Ralph Wiggum technique. Your task is to set up ALL necessary infrastructure, validation scripts, configuration files, and documentation so the repository is ready for the infinite refactoring loop.

## Completion Criteria

When ALL setup tasks are complete and verified, output exactly:
<promise>SETUP_COMPLETE</promise>

## Your Mission

Analyze the current repository structure and create a complete Ralph Wiggum setup that includes:

1. Validation scripts for Go backend and React frontend
2. Configuration files for linters and tools
3. Ralph loop execution script
4. Architecture documentation
5. Stop hook checker
6. Git hooks (optional but recommended)

---

## PHASE 1: ANALYZE REPOSITORY

First, examine the repository:

```bash
# Check Go backend structure
ls -la
find . -name "*.go" -type f | head -20
cat go.mod 2>/dev/null || echo "No go.mod found"

# Check React frontend structure
ls -la web-ui/ 2>/dev/null || echo "No web-ui directory"
cat web-ui/package.json 2>/dev/null || echo "No package.json found"
```

Document findings:

- Current Go structure (packages, layout)
- Current React structure (directories, build tool)
- Existing linters/tools
- Testing framework
- Any existing configuration files

---

## PHASE 2: CREATE GO BACKEND VALIDATION

### File: `scripts/validate-go.sh`

Create comprehensive Go validation script:

```bash
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
UNFORMATTED=$(gofmt -l . 2>&1 | grep -v vendor || true)
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
if go test ./... -cover 2>&1; then
    echo -e "${GREEN}✓ All tests passed${NC}"
else
    echo -e "${RED}❌ Tests failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 5. Test coverage
echo ""
echo "[5/8] Checking test coverage..."
COVERAGE=$(go test ./... -coverprofile=coverage.out -covermode=atomic 2>&1 | grep -oP 'coverage: \K[0-9.]+' || echo "0")
rm -f coverage.out
echo "Overall coverage: ${COVERAGE}%"
if (( $(echo "$COVERAGE < 70" | bc -l) )); then
    echo -e "${YELLOW}⚠️  Coverage below 70% (currently ${COVERAGE}%)${NC}"
else
    echo -e "${GREEN}✓ Coverage adequate (${COVERAGE}%)${NC}"
fi

# 6. Circular dependencies
echo ""
echo "[6/8] Checking for circular dependencies..."
if go list -json ./... | jq -r '.ImportPath' | while read pkg; do
    go list -f '{{.ImportPath}}: {{join .Imports ", "}}' "$pkg" 2>/dev/null
done > /tmp/go-deps.txt 2>&1; then
    echo -e "${GREEN}✓ No circular dependency issues detected${NC}"
else
    echo -e "${YELLOW}⚠️  Could not fully analyze dependencies${NC}"
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
TODO_COUNT=$(grep -r "TODO\|FIXME\|HACK\|XXX" --include="*.go" . | grep -v vendor | wc -l || echo "0")
if [ "$TODO_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $TODO_COUNT TODO/FIXME/HACK comments${NC}"
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
```

Make it executable:

```bash
chmod +x scripts/validate-go.sh
```

---

## PHASE 3: CREATE REACT FRONTEND VALIDATION

### File: `scripts/validate-react.sh`

Create React validation script:

```bash
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
    echo "[0/6] Installing dependencies..."
    npm install
fi

# 2. ESLint
echo ""
echo "[1/6] Running ESLint..."
if npm run lint 2>&1 || npx eslint . --ext .js,.jsx,.ts,.tsx 2>&1; then
    echo -e "${GREEN}✓ ESLint passed${NC}"
else
    echo -e "${RED}❌ ESLint found issues${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 3. Prettier
echo ""
echo "[2/6] Checking code formatting (Prettier)..."
if npm run format:check 2>&1 || npx prettier --check "src/**/*.{js,jsx,ts,tsx,json,css}" 2>&1; then
    echo -e "${GREEN}✓ Code properly formatted${NC}"
else
    echo -e "${YELLOW}⚠️  Code formatting issues (run 'npm run format')${NC}"
fi

# 4. TypeScript check (if applicable)
echo ""
echo "[3/6] TypeScript checking..."
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

# 5. Build
echo ""
echo "[4/6] Building application..."
if npm run build 2>&1; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 6. Check for common anti-patterns
echo ""
echo "[5/6] Checking for anti-patterns..."
CONSOLE_LOGS=$(grep -r "console.log" src/ --include="*.js" --include="*.jsx" --include="*.ts" --include="*.tsx" | wc -l || echo "0")
if [ "$CONSOLE_LOGS" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $CONSOLE_LOGS console.log statements${NC}"
fi

# Check for large components (>100 lines)
LARGE_COMPONENTS=$(find src/components -name "*.jsx" -o -name "*.tsx" | while read file; do
    lines=$(wc -l < "$file")
    if [ "$lines" -gt 100 ]; then
        echo "$file ($lines lines)"
    fi
done)
if [ -n "$LARGE_COMPONENTS" ]; then
    echo -e "${YELLOW}⚠️  Large components found (>100 lines):${NC}"
    echo "$LARGE_COMPONENTS"
fi

# 7. Unused dependencies
echo ""
echo "[6/6] Checking for unused dependencies..."
if command -v depcheck &> /dev/null; then
    npx depcheck || echo -e "${YELLOW}⚠️  Unused dependencies found${NC}"
else
    echo -e "${YELLOW}⚠️  depcheck not available${NC}"
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
```

Make it executable:

```bash
chmod +x scripts/validate-react.sh
```

---

## PHASE 4: CREATE UNIFIED VALIDATION

### File: `scripts/validate-all.sh`

```bash
#!/bin/bash
# scripts/validate-all.sh
# Master validation script for entire repository

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "╔════════════════════════════════════════════╗"
echo "║  Full Repository Validation                ║"
echo "╔════════════════════════════════════════════╗"
echo ""

ERRORS=0

# Validate Go backend
if bash "$SCRIPT_DIR/validate-go.sh"; then
    echo ""
    echo "✅ Go backend validation PASSED"
else
    echo ""
    echo "❌ Go backend validation FAILED"
    ERRORS=$((ERRORS + 1))
fi

echo ""
echo "---"
echo ""

# Validate React frontend
if [ -d "web-ui" ]; then
    if bash "$SCRIPT_DIR/validate-react.sh"; then
        echo ""
        echo "✅ React frontend validation PASSED"
    else
        echo ""
        echo "❌ React frontend validation FAILED"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "⚠️  No web-ui directory found, skipping frontend validation"
fi

echo ""
echo "╔════════════════════════════════════════════╗"
if [ $ERRORS -eq 0 ]; then
    echo "║  ✅ ALL VALIDATIONS PASSED                 ║"
    echo "╔════════════════════════════════════════════╗"
    exit 0
else
    echo "║  ❌ VALIDATION FAILED ($ERRORS component(s))      ║"
    echo "╔════════════════════════════════════════════╗"
    exit 1
fi
```

Make it executable:

```bash
chmod +x scripts/validate-all.sh
```

---

## PHASE 5: CREATE RALPH WIGGUM LOOP SCRIPT

### File: `scripts/ralph-loop.sh`

```bash
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
echo "╔═══════════════════════════════════════════════════╗"
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
        if git log -1 --pretty=%B | grep -q "<promise>COMPLETE</promise>"; then
            echo ""
            echo "╔═══════════════════════════════════════════════════╗"
            echo "║           🎉 RALPH LOOP COMPLETE! 🎉              ║"
            echo "╔═══════════════════════════════════════════════════╗"
            echo ""
            echo "Repository successfully refactored after $ITERATION iterations"
            exit 0
        fi
    fi

    # Feed prompt to AI (this is where you'd integrate with Claude Code or API)
    echo ""
    echo "📝 Feeding prompt to AI agent..."
    echo "   (Integrate with Claude Code: claude-code --prompt-file $PROMPT_FILE)"
    echo ""

    # For now, this is a manual step - you would integrate with Claude Code CLI
    # Example: claude-code --prompt-file "$PROMPT_FILE" --auto-approve

    read -p "Press Enter after AI completes this iteration (or Ctrl+C to stop)..."

    # Check for completion
    if git log -1 --pretty=%B | grep -q "<promise>COMPLETE</promise>"; then
        echo ""
        echo "╔═══════════════════════════════════════════════════╗"
        echo "║           🎉 RALPH LOOP COMPLETE! 🎉              ║"
        echo "╔═══════════════════════════════════════════════════╗"
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
```

Make it executable:

```bash
chmod +x scripts/ralph-loop.sh
```

---

## PHASE 6: CREATE RALPH PROMPT

### File: `scripts/ralph-prompt.md`

`````markdown
    # Clean Architecture & SOLID Principles - Go Backend + React Frontend

    You are refactoring a monorepo with a Go backend (root) and React frontend (web-ui/). Apply idiomatic Go patterns with SOLID principles to the backend and clean UI/logic separation to the frontend. Work continues until both codebases are pristine.

    ## Completion Criteria

    When ALL conditions below are verified, output exactly:
    <promise>COMPLETE</promise>

    Otherwise, continue working.

    ---

    ## GO BACKEND ARCHITECTURE (Root Directory)

    ### Idiomatic Go Structure

    Follow "screaming architecture" with feature-based packages, not layer-based. Structure packages around problems they solve, not component types.

    **Standard Go Project Layout:**
    ```

    / (root - Go backend)
    ├── cmd/
    │ └── server/
    │ └── main.go # Application entry point
    ├── internal/ # Private application code
    │ ├── {feature}/ # Feature packages (user, product, order)
    │ │ ├── entity.go # Domain models
    │ │ ├── repository.go # Data access interfaces
    │ │ ├── repository_impl.go # Repository implementation
    │ │ ├── service.go # Business logic (use cases)
    │ │ └── handler.go # HTTP handlers (controllers)
    │ └── config/
    │ └── config.go
    ├── pkg/ # Public libraries (if any)
    ├── infrastructure/ # External concerns
    │ ├── database/
    │ │ └── postgres.go
    │ ├── http/
    │ │ ├── router.go
    │ │ └── middleware/
    │ └── logger/
    ├── migrations/ # Database migrations
    ├── go.mod
    └── go.sum
    ```

    ### Key Principles:

    - Each feature package contains entity, service, repository, and API handler
    - Dependencies flow inward: handler → service → repository → entity
    - Use interfaces for all cross-layer communication
    - Keep internal/ for private code, pkg/ for reusable libraries

    ### Go-Specific SOLID Implementation

    #### S - Single Responsibility (Go Style)

    ```go

    // ❌ BAD: God struct doing everything
    type UserService struct {
        db *sql.DB
        logger *log.Logger
        emailer *smtp.Client
    }

    // ✅ GOOD: Single responsibility per struct
    type UserService struct {
        repo   UserRepository    // interface
        notifier Notifier        // interface
        logger Logger            // interface
    }
    ````

    #### O - Open/Closed (Interfaces)

    ```go
    // ✅ Use interfaces for extension
    type PaymentProcessor interface {
        Process(amount float64) error
    }

    // Extend with new implementations
    type StripeProcessor struct{}
    type PayPalProcessor struct{}
    ```

    #### L - Liskov Substitution

    ```go
    // ✅ All implementations honor the contract
    type Storage interface {
        Save(key string, value []byte) error
    }

    // Both implementations can substitute the interface
    type FileStorage struct{}
    type S3Storage struct{}
    ```

    #### I - Interface Segregation (Small Interfaces)

    ```go
    // ❌ BAD: Fat interface
    type Repository interface {
        Create() error
        Read() error
        Update() error
        Delete() error
        Search() error
        Export() error
        Import() error
    }

    // ✅ GOOD: Small, focused interfaces
    type Creator interface { Create() error }
    type Reader interface { Read() error }
    type Updater interface { Update() error }
    ```

    #### D - Dependency Inversion (Constructor Injection)

    ```go// ✅ Depend on abstractions, inject via constructor
    type UserService struct {
        repo UserRepository  // interface, not concrete
    }

    func NewUserService(repo UserRepository) *UserService {
        return &UserService{repo: repo}
    }
    ```

    ### Go Backend Validation Checklist

    - All packages follow feature-based structure (not layer-based)
    - No package exceeds 500 lines per file
    - All cross-package dependencies use interfaces
    - Constructor injection used everywhere (NewX pattern)
    - No global variables (except constants)
    - Errors wrapped with context (fmt.Errorf("%w", err))
    - Context.Context passed as first parameter
    - go.mod dependencies are minimal and up-to-date
    - No circular dependencies (go mod graph)

    ### Code Quality:

    - gofmt applied to all files
    - golangci-lint passes with no errors
    - go vet passes
    - staticcheck passes
    - All exported functions have godoc comments
    - All tests pass (go test ./...)
    - Test coverage ≥ 70% for business logic
    - No naked returns in functions >5 lines

    ### Idiomatic Go Patterns:

    - Error handling: return errors, don't panic
    - Interfaces defined where used (not where implemented)
    - Accept interfaces, return structs
    - Keep interfaces small (1-3 methods ideal)
    - Use table-driven tests
    - Avoid getters/setters (export fields directly when appropriate)

    ---

    ## REACT FRONTEND ARCHITECTURE (web-ui/)

    ### Two-Layer Architecture: Logic + UI

    Separate business logic from UI using custom hooks for dependency injection. Keep components focused on presentation.

    **Structure:**

    ```
    web-ui/
    ├── src/
    │   ├── features/              # Feature-based organization
    │   │   ├── auth/
    │   │   │   ├── components/    # UI components (presentational)
    │   │   │   │   ├── LoginForm.jsx
    │   │   │   │   └── RegisterForm.jsx
    │   │   │   ├── hooks/         # Logic layer (custom hooks)
    │   │   │   │   ├── useAuth.js
    │   │   │   │   └── useLogin.js
    │   │   │   └── services/      # API calls
    │   │   │       └── authService.js
    │   │   ├── products/
    │   │   └── orders/
    │   ├── components/            # Shared UI components
    │   │   ├── Button.jsx
    │   │   └── Input.jsx
    │   ├── hooks/                 # Shared custom hooks
    │   ├── services/              # Shared API services
    │   │   └── api.js
    │   ├── utils/                 # Pure utility functions
    │   ├── App.jsx
    │   └── main.jsx
    ├── package.json
    └── vite.config.js
    Logic/UI Separation Pattern
    ❌ BAD - Logic mixed with UI:
    jsxfunction ProductList() {
      const [products, setProducts] = useState([]);
      const [loading, setLoading] = useState(true);

      useEffect(() => {
        fetch('/api/products')
          .then(res => res.json())
          .then(data => {
            const filtered = data.filter(p => p.stock > 0);
            setProducts(filtered);
            setLoading(false);
          });
      }, []);

      return loading ? <div>Loading...</div> : (
        <div>{products.map(p => <div key={p.id}>{p.name}</div>)}</div>
      );
    }
    ✅ GOOD - Logic separated:
    javascript// hooks/useProducts.js (Logic Layer)
    function useProducts() {
      const [products, setProducts] = useState([]);
      const [loading, setLoading] = useState(true);
      const [error, setError] = useState(null);

      useEffect(() => {
        productService.fetchProducts()
          .then(filterInStock)
          .then(setProducts)
          .catch(setError)
          .finally(() => setLoading(false));
      }, []);

      return { products, loading, error };
    }

    // utils/productUtils.js (Pure functions)
    const filterInStock = (products) => products.filter(p => p.stock > 0);

    // components/ProductList.jsx (UI Layer)
    function ProductList() {
      const { products, loading, error } = useProducts();

      if (loading) return <LoadingSpinner />;
      if (error) return <ErrorMessage error={error} />;

      return (
        <div>
          {products.map(p => <ProductCard key={p.id} product={p} />)}
        </div>
      );
    }
    React Validation Checklist
    bash✓ No business logic in component JSX files
    ✓ All stateful logic extracted to custom hooks
    ✓ Components are purely presentational (≤50 lines ideal)
    ✓ All API calls in dedicated service files
    ✓ Custom hooks follow "use" naming convention
    ✓ No prop drilling (use context/state management if needed)
    ✓ All components use PascalCase, functions use camelCase


    Code Quality:

    ✓ ESLint passes (no errors, minimal warnings)
    ✓ Prettier applied to all files
    ✓ No console.log statements in production code
    ✓ No unused imports or variables
    ✓ PropTypes or TypeScript types defined
    ✓ Key props on all list items
    ✓ useEffect dependencies correct (no lint warnings)
    ```

    ---

    ## ITERATION PROCESS

    **PRIORITY ORDER:**

    1. **Go: Architecture violations** (layer mixing, circular deps)
    2. **Go: SOLID violations** (fat structs, tight coupling)
    3. **React: Logic/UI mixing** (business logic in components)
    4. **Go: Idiomatic issues** (error handling, interfaces)
    5. **React: Code organization** (hooks, services)
    6. **Both: Unused code & duplication**
    7. **Both: Documentation gaps**

    **EACH ITERATION:**

    1. **SCAN**: Run `./scripts/validate-all.sh`
    2. **IDENTIFY**: Pick highest-priority issue from validation output
    3. **REFACTOR**: Make ONE focused improvement
    4. **TEST**: Run validation again
    5. **COMMIT**: Clear git commit with description
    6. **REPORT**: Document what was fixed

    ### Iteration Report Format

    ```
    === Iteration Report ===
    Component: [Go Backend | React Frontend]
    Status: [WORKING | COMPLETE]

    Fixed:
      Principle: [Clean Architecture | SOLID | Lean]
      Issue: [specific violation]
      Solution: [what you did]

    Validation:
      Go: [✓ pass / ❌ fail]
      React: [✓ pass / ❌ fail]

    Remaining Issues: [number or "None detected"]
    Next Target: [what you'll work on next, or "N/A"]
    ```

    ### Commit Message Format

    ```
    refactor(go/user): extract UserRepository interface

    - Apply Dependency Inversion Principle
    - Enable easier testing and mocking
    - Remove direct DB dependency from service
    ```

    ---

    ## CONSTRAINTS

    - ✓ Make incremental changes (one fix per iteration)
    - ✓ Always run tests after changes
    - ✓ Never skip tests to make validation pass
    - ✓ Preserve existing functionality
    - ✓ Document reasoning in commits

    ---

    ## BEGIN FIRST ITERATION

    Run `./scripts/validate-all.sh` and identify the single highest-priority issue to fix.
`````

---

## PHASE 7: CREATE CONFIGURATION FILES

### File: `.golangci.yml`

```yaml
# .golangci.yml
# golangci-lint configuration

linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - goconst
    - gocyclo
    - misspell
    - unparam
    - unconvert
    - goimports
    - revive

linters-settings:
  gofmt:
    simplify: true
  gocyclo:
    min-complexity: 15
  goconst:
    min-len: 3
    min-occurrences: 3
  misspell:
    locale: US
  revive:
    rules:
      - name: exported
        severity: warning
      - name: var-naming
        severity: warning

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0

run:
  timeout: 5m
  tests: true
```

### File: `web-ui/.eslintrc.json` (if doesn't exist)

```json
{
  "extends": [
    "eslint:recommended",
    "plugin:react/recommended",
    "plugin:react-hooks/recommended"
  ],
  "parserOptions": {
    "ecmaVersion": 2021,
    "sourceType": "module",
    "ecmaFeatures": {
      "jsx": true
    }
  },
  "env": {
    "browser": true,
    "es2021": true,
    "node": true
  },
  "settings": {
    "react": {
      "version": "detect"
    }
  },
  "rules": {
    "no-console": "warn",
    "no-unused-vars": "error",
    "react/prop-types": "warn",
    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn"
  }
}
```

### File: `web-ui/.prettierrc` (if doesn't exist)

```json
{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 80,
  "tabWidth": 2,
  "useTabs": false
}
```

### File: `web-ui/package.json` scripts (add if missing)

Check if these scripts exist, if not add them:

```json
{
  "scripts": {
    "lint": "eslint src --ext .js,.jsx,.ts,.tsx",
    "lint:fix": "eslint src --ext .js,.jsx,.ts,.tsx --fix",
    "format": "prettier --write \"src/**/*.{js,jsx,ts,tsx,json,css}\"",
    "format:check": "prettier --check \"src/**/*.{js,jsx,ts,tsx,json,css}\""
  }
}
```

---

## PHASE 8: CREATE DOCUMENTATION

### File: `RALPH_SETUP.md`

````markdown
# Ralph Wiggum Clean Architecture Setup

This repository is configured for automated technical debt resolution using the Ralph Wiggum technique.

## What is Ralph Wiggum?

Ralph Wiggum is an infinite loop technique that repeatedly feeds the same refactoring prompt to an AI coding agent until the codebase meets all quality criteria. Progress persists in files and git history, not in the AI's context window.

## Structure

- **Go Backend** (root): Clean Architecture with SOLID principles, idiomatic Go patterns
- **React Frontend** (web-ui/): Logic/UI separation with custom hooks

## Quick Start

### 1. Run Validation

Test current state:

```bash
./scripts/validate-all.sh
```

### 2. Start Ralph Loop

```bash
./scripts/ralph-loop.sh
```

This will continuously refactor until the completion promise is found.

### 3. Manual Iteration

For a single refactoring pass:

```bash
# Read the prompt
cat scripts/ralph-prompt.md

# Apply changes
# ... make your changes ...

# Validate
./scripts/validate-all.sh

# Commit with proper format
git commit -m "refactor(go/user): extract repository interface

- Apply Dependency Inversion Principle
- Enable easier testing"
```

## Validation Scripts

- `scripts/validate-go.sh` - Go backend validation
- `scripts/validate-react.sh` - React frontend validation
- `scripts/validate-all.sh` - Full repository validation

## Architecture Goals

### Go Backend

- Feature-based package structure
- Interface-driven design
- Constructor injection
- Repository pattern
- No circular dependencies
- 70%+ test coverage

### React Frontend

- Custom hooks for logic
- Presentational components
- Service layer for APIs
- No business logic in JSX
- Clean, readable components

## Completion Criteria

The loop completes when:

- All validations pass
- No TODO/FIXME/HACK comments
- All SOLID principles applied
- Clean Architecture enforced
- Code is idiomatic and maintainable

## Tools Required

### Go

- Go 1.21+ (or your version)
- golangci-lint (recommended)
- staticcheck (alternative)

### React

- Node.js 18+
- npm or yarn
- ESLint
- Prettier

## Manual Installation

Install tools if missing:

```bash
# Go tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install honnef.co/go/tools/cmd/staticcheck@latest

# React tools (in web-ui/)
cd web-ui
npm install --save-dev eslint prettier eslint-plugin-react eslint-plugin-react-hooks
```

## Git Commit Convention
````

refactor(component): brief description

- Why this change (principle violated)
- What was changed
- Impact/benefit

```

## Support

If validation fails, review the output for specific issues and refactor incrementally.
```

---

## PHASE 9: VERIFY SETUP

Run these verification steps:

```bash
# 1. Make all scripts executable
chmod +x scripts/*.sh

# 2. Test Go validation
./scripts/validate-go.sh || echo "Expected to fail initially"

# 3. Test React validation (if web-ui exists)
[ -d web-ui ] && ./scripts/validate-react.sh || echo "Expected to fail initially"

# 4. Test combined validation
./scripts/validate-all.sh || echo "Expected to fail initially"

# 5. Verify all files created
ls -la scripts/
cat scripts/ralph-prompt.md | head -20
cat .golangci.yml
[ -f web-ui/.eslintrc.json ] && cat web-ui/.eslintrc.json || echo "ESLint config created"
```

---

## PHASE 10: CREATE CHECKLIST & SUMMARY

Create final summary document:

### File: `SETUP_CHECKLIST.md`

````markdown
# Ralph Wiggum Setup Checklist

## ✅ Files Created

- [ ] `scripts/validate-go.sh` - Go validation script
- [ ] `scripts/validate-react.sh` - React validation script
- [ ] `scripts/validate-all.sh` - Combined validation
- [ ] `scripts/ralph-loop.sh` - Main Ralph loop
- [ ] `scripts/ralph-prompt.md` - Refactoring instructions
- [ ] `.golangci.yml` - Go linter configuration
- [ ] `web-ui/.eslintrc.json` - ESLint configuration
- [ ] `web-ui/.prettierrc` - Prettier configuration
- [ ] `RALPH_SETUP.md` - Documentation
- [ ] `SETUP_CHECKLIST.md` - This file

## ✅ Scripts Executable

- [ ] `chmod +x scripts/validate-go.sh`
- [ ] `chmod +x scripts/validate-react.sh`
- [ ] `chmod +x scripts/validate-all.sh`
- [ ] `chmod +x scripts/ralph-loop.sh`

## ✅ Tools Installed

### Go

- [ ] `go version` works
- [ ] `golangci-lint version` works (or staticcheck)
- [ ] `go test ./...` runs

### React

- [ ] `node --version` works (18+)
- [ ] `npm --version` works
- [ ] `cd web-ui && npm install` succeeds
- [ ] ESLint installed in web-ui
- [ ] Prettier installed in web-ui

## ✅ Validation Tests

- [ ] `./scripts/validate-go.sh` runs (may fail initially)
- [ ] `./scripts/validate-react.sh` runs (may fail initially)
- [ ] `./scripts/validate-all.sh` runs (may fail initially)

## ✅ Git Setup

- [ ] Repository has git initialized
- [ ] `.gitignore` includes `coverage.out`, `node_modules/`, etc.
- [ ] Can commit changes

## 🚀 Ready to Start

Once all checkboxes are ticked, you're ready to:

```bash
./scripts/ralph-loop.sh
```

Or run manual iterations with:

```bash
./scripts/validate-all.sh
# Review issues
# Make changes
# Commit
# Repeat
```
````

---

## FINAL VERIFICATION

Before outputting completion:

1. ✅ Verify all script files created in `scripts/` directory
2. ✅ Verify all scripts are executable (`chmod +x`)
3. ✅ Verify configuration files created (`.golangci.yml`, ESLint, Prettier)
4. ✅ Verify documentation created (`RALPH_SETUP.md`, `SETUP_CHECKLIST.md`)
5. ✅ Verify scripts have no syntax errors (run with `bash -n scriptname.sh`)
6. ✅ Test that validation scripts can run (even if they fail on current code)

---

## SUCCESS CRITERIA

The setup is complete when:

- ✅ All validation scripts exist and are executable
- ✅ All configuration files exist
- ✅ Ralph loop script exists
- ✅ Ralph prompt exists with complete instructions
- ✅ Documentation is in place
- ✅ Scripts can be run without errors (even if validation fails)

When all above criteria are met, output:

<promise>SETUP_COMPLETE</promise>

And provide a summary of:

1. Files created
2. Next steps for the user
3. How to start the Ralph loop
