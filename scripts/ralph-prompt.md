# Clean Architecture & SOLID Principles - Go Backend + React Frontend

You are refactoring a monorepo with a Go backend (root) and React/TypeScript frontend (web-ui/). Apply idiomatic Go patterns with SOLID principles to the backend and clean UI/logic separation to the frontend. Work continues until both codebases are pristine.

## Completion Criteria

When ALL conditions below are verified, output exactly:
<promise>COMPLETE</promise>

Otherwise, continue working.

---

## GO BACKEND ARCHITECTURE (Root Directory)

### Idiomatic Go Structure

Follow "screaming architecture" with feature-based packages, not layer-based. Structure packages around problems they solve, not component types.

**Current Structure:**
```
/ (root - Go backend)
├── cmd/            # Application entry points (if needed)
├── api/            # HTTP handlers and API routes
├── cache/          # File caching with TTL
├── fetcher/        # HTTP client with cache fallback
├── multiplexer/    # Stream fan-out to clients
├── circuitbreaker/ # Resilience for upstream
├── overrides/      # Channel metadata customization
├── metrics/        # Prometheus metrics
├── logging/        # Structured logging
├── aceproxy/       # Acestream client
├── ui/             # Embedded React SPA (go:embed)
├── main.go
├── go.mod
└── go.sum
```

### Key Principles:

- Each package contains its own entity, service, and interfaces
- Dependencies flow inward: handler → service → repository → entity
- Use interfaces for all cross-package communication
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
```

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

```go
// ✅ Depend on abstractions, inject via constructor
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
- Test coverage ≥ 50% for business logic
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
│   │   ├── channels/
│   │   │   ├── components/    # UI components (presentational)
│   │   │   ├── hooks/         # Logic layer (custom hooks)
│   │   │   └── services/      # API calls
│   ├── components/            # Shared UI components
│   ├── hooks/                 # Shared custom hooks
│   ├── services/              # Shared API services
│   ├── utils/                 # Pure utility functions
│   ├── types/                 # TypeScript type definitions
│   ├── App.tsx
│   └── main.tsx
├── package.json
└── vite.config.ts
```

### Logic/UI Separation Pattern

❌ BAD - Logic mixed with UI:
```tsx
function ProductList() {
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
```

✅ GOOD - Logic separated:
```tsx
// hooks/useProducts.ts (Logic Layer)
function useProducts() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    productService.fetchProducts()
      .then(filterInStock)
      .then(setProducts)
      .catch(setError)
      .finally(() => setLoading(false));
  }, []);

  return { products, loading, error };
}

// utils/productUtils.ts (Pure functions)
const filterInStock = (products: Product[]) => products.filter(p => p.stock > 0);

// components/ProductList.tsx (UI Layer)
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
```

### React Validation Checklist

```
✓ No business logic in component TSX files
✓ All stateful logic extracted to custom hooks
✓ Components are purely presentational (≤100 lines ideal)
✓ All API calls in dedicated service files
✓ Custom hooks follow "use" naming convention
✓ No prop drilling (use context/state management if needed)
✓ All components use PascalCase, functions use camelCase
```

### Code Quality:

```
✓ ESLint passes (no errors, minimal warnings)
✓ Prettier applied to all files
✓ No console.log statements in production code
✓ No unused imports or variables
✓ TypeScript types defined (no `any` where avoidable)
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
