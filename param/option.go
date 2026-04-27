package param

// DefaultValueFunc defines the signature for default value functions
type DefaultValueFunc func() (interface{}, error)

// Option defines parameter option functions
type Option func(*options)

type options struct {
	envKey         string
	defaultVal     interface{}
	defaultValFunc DefaultValueFunc
	required       bool
	validator      func(interface{}) error
	separator      string // for StringSlice env parsing
}

// WithEnvKey specifies environment variable key
func WithEnvKey(key string) Option {
	return func(o *options) {
		o.envKey = key
	}
}

// WithDefault specifies static default value
func WithDefault(val interface{}) Option {
	return func(o *options) {
		o.defaultVal = val
	}
}

// WithDefaultFunc specifies dynamic default value function
func WithDefaultFunc(f DefaultValueFunc) Option {
	return func(o *options) {
		o.defaultValFunc = f
	}
}

// WithRequired marks parameter as required
func WithRequired() Option {
	return func(o *options) {
		o.required = true
	}
}

// WithValidator adds custom validator
func WithValidator(v func(interface{}) error) Option {
	return func(o *options) {
		o.validator = v
	}
}

// WithSeparator specifies the separator for string slice environment variable parsing.
// Default is "," if not specified.
func WithSeparator(sep string) Option {
	return func(o *options) {
		o.separator = sep
	}
}
