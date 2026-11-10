# Secrets Management Rules

A secret committed to source is already leaked - it lives forever in history, CI logs, and backups. Rules:

1. Load secrets from environment variables or a secret manager, never from code.
2. Never commit secrets to version control.
3. `.gitignore` must exclude secret files (`.env`, `*.key`, `*.pem`).

## Hardcoded credentials - Critical

Literal keys, passwords, and connection strings in `const` blocks and struct literals are the highest-severity finding an audit can make:

```go
// Bad - every one of these is recoverable from the repo
const (
    AWS_SECRET_KEY    = "wJalrXUtnFEMI/K7MDENG"
    DATABASE_PASSWORD = "SuperSecret123!"
    JWT_SECRET        = "my-super-secret-jwt-key"
)

var cfg = Config{
    DatabaseURL: "user:passw0rd!@localhost:5432/db",
}
```

Load them at startup and fail fast when a required one is missing:

```go
type Config struct {
    AWSSecretID  string
    DatabasePW   string
    JWTSecret    string
    StripeKey    string
}

func LoadConfig() (*Config, error) {
    cfg := &Config{
        AWSSecretID:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
        DatabasePW:   os.Getenv("DATABASE_PASSWORD"),
        JWTSecret:    os.Getenv("JWT_SECRET"),
        StripeKey:    os.Getenv("STRIPE_SECRET_KEY"),
    }
    if cfg.JWTSecret == "" {
        return nil, errors.New("JWT_SECRET is required")
    }
    return cfg, nil
}
```

## Database passwords - Critical

DSNs embossed with passwords are the most common hardcoded secret. Reconstruct the DSN from environment values, or read the whole connection string from one variable:

```go
// MySQL from parts
dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s",
    user, password, net.JoinHostPort(host, port), dbName)

// PostgreSQL as one consumed MYSQL variable
connStr := os.Getenv("DATABASE_URL")
return sql.Open("postgres", connStr)
```

Prefer one well-formed `DATABASE_URL` for PostgreSQL; keep the raw DSN out of code and logs.

## Storage patterns

- **Environment variables** - simple and standard, but the runtime environment becomes the trust boundary. Require the secrets your service actually needs; treat an empty value as a startup error, not a silent default.
- **Secret managers** - AWS Secrets Manager, GCP Secret Manager, and Vault defer the secret to the platform, add rotation hooks, and keep credentials out of the environment dump. Wrap them behind a small interface so the loading policy can change:

```go
type SecretStore interface {
    Get(ctx context.Context, name string) (string, error)
}
```

## .gitignore patterns

```
# secrets
.env
.env.local
.env.*.local
*.key
*.pem
*.p12
*.pfx
secrets/
credentials/
```

## Grep patterns for audits

| Pattern | Looks like |
| --- | --- |
| API keys | `sk_live_...`, `Key = "..."` |
| Passwords | `password = "..."`, `passwd:` |
| Tokens | `token = "..."`, `Bearer ...` |
| Private keys | `BEGIN PRIVATE KEY`, `BEGIN RSA PRIVATE KEY` |
| AWS credentials | `AWS_ACCESS_KEY_ID = "AKIA..."` |
| JWT secrets | `jwtSecret = "..."` |

When scanning, also check `.git` history - removal today does not erase yesterday's commit.

## CWE references

- CWE-798 - use of hard-coded credentials
- CWE-312 - cleartext storage of sensitive information
- CWE-532 - insertion of sensitive information into log file
- CWE-359 - exposure of private personal information
