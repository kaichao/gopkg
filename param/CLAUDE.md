# CLAUDE.md

## param Package

Unified command line parameter management for Cobra.

### Priority
CLI args > env vars > static defaults > dynamic defaults

### Type Support
`int`, `string`, `bool`, `time.Duration`, `int64`, `float64`, `[]string`

### Key Functions
```go
func GetInt(cmd *cobra.Command, name string, opts ...Option) (int, error)
func GetString(cmd *cobra.Command, name string, opts ...Option) (string, error)
func GetBool(cmd *cobra.Command, name string, opts ...Option) (bool, error)
func GetDuration(cmd *cobra.Command, name string, opts ...Option) (time.Duration, error)
func GetInt64(cmd *cobra.Command, name string, opts ...Option) (int64, error)
func GetFloat64(cmd *cobra.Command, name string, opts ...Option) (float64, error)
func GetStringSlice(cmd *cobra.Command, name string, opts ...Option) ([]string, error)
```

### Options
- `WithEnvKey(key string)` — Custom env var name
- `WithDefault(val interface{})` — Static default
- `WithDefaultFunc(f DefaultValueFunc)` — Dynamic default
- `WithRequired()` — Must be set
- `WithValidator(v func(interface{}) error)` — Custom validation

### Env Var Derivation
`app-id` → `APP_ID`, `cluster-name` → `CLUSTER_NAME`

### Usage Example
```go
appID, err := param.GetInt(cmd, "app-id")
cluster, err := param.GetString(cmd, "cluster",
    param.WithRequired(),
    param.WithDefault("default-cluster"),
)
```
