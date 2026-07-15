package security

import "fmt"

// AuthenticatorFactory 是创建 Authenticator 的工厂函数类型。
type AuthenticatorFactory func(cfg AuthConfig) (Authenticator, error)

// AuthorizerFactory 是创建 Authorizer 的工厂函数类型。
type AuthorizerFactory func(cfg AuthzConfig) (Authorizer, error)

// BillingServiceFactory 是创建 BillingService 的工厂函数类型。
type BillingServiceFactory func(cfg BillConfig) (BillingService, error)

// TokenAuthenticatorFactory 是创建 TokenAuthenticator 的工厂函数类型。
type TokenAuthenticatorFactory func(cfg AuthConfig) (TokenAuthenticator, error)

var (
	authFactories      = map[string]AuthenticatorFactory{}
	authzFactories     = map[string]AuthorizerFactory{}
	billFactories      = map[string]BillingServiceFactory{}
	tokenAuthFactories = map[string]TokenAuthenticatorFactory{}
)

// RegisterAuthenticator 注册一个 Authenticator 工厂。
// 通常在 .so 插件的 init() 中调用，重复注册会 panic。
func RegisterAuthenticator(name string, fn AuthenticatorFactory) {
	if _, exists := authFactories[name]; exists {
		panic("security: duplicate authenticator registration: " + name)
	}
	authFactories[name] = fn
}

// RegisterAuthorizer 注册一个 Authorizer 工厂。
func RegisterAuthorizer(name string, fn AuthorizerFactory) {
	if _, exists := authzFactories[name]; exists {
		panic("security: duplicate authorizer registration: " + name)
	}
	authzFactories[name] = fn
}

// RegisterBillingService 注册一个 BillingService 工厂。
func RegisterBillingService(name string, fn BillingServiceFactory) {
	if _, exists := billFactories[name]; exists {
		panic("security: duplicate billing service registration: " + name)
	}
	billFactories[name] = fn
}

// RegisterTokenAuthenticator 注册一个 TokenAuthenticator 工厂。
// 通常在 .so 插件的 init() 中调用，重复注册会 panic。
func RegisterTokenAuthenticator(name string, fn TokenAuthenticatorFactory) {
	if _, exists := tokenAuthFactories[name]; exists {
		panic("security: duplicate token authenticator registration: " + name)
	}
	tokenAuthFactories[name] = fn
}

// NewAuthenticator 按名称创建 Authenticator。
// mode 为空或 "noop" 返回 NoopAuthenticator。
func NewAuthenticator(mode string, cfg AuthConfig) (Authenticator, error) {
	if mode == "" || mode == "noop" {
		return &NoopAuthenticator{}, nil
	}
	fn, ok := authFactories[mode]
	if !ok {
		return nil, fmt.Errorf("security: unknown authenticator mode: %s (available: %s)", mode, authModes())
	}
	return fn(cfg)
}

// NewAuthorizer 按名称创建 Authorizer。
func NewAuthorizer(mode string, cfg AuthzConfig) (Authorizer, error) {
	if mode == "" || mode == "noop" {
		return &NoopAuthorizer{}, nil
	}
	fn, ok := authzFactories[mode]
	if !ok {
		return nil, fmt.Errorf("security: unknown authorizer mode: %s (available: %s)", mode, authzModes())
	}
	return fn(cfg)
}

// NewBillingService 按名称创建 BillingService。
func NewBillingService(mode string, cfg BillConfig) (BillingService, error) {
	if mode == "" || mode == "noop" {
		return &NoopBillingService{}, nil
	}
	fn, ok := billFactories[mode]
	if !ok {
		return nil, fmt.Errorf("security: unknown billing service mode: %s (available: %s)", mode, billModes())
	}
	return fn(cfg)
}

// NewTokenAuthenticator 按名称创建 TokenAuthenticator。
// mode 为空或 "noop" 时返回 nil（调用方自行处理默认行为）。
func NewTokenAuthenticator(mode string, cfg AuthConfig) (TokenAuthenticator, error) {
	if mode == "" || mode == "noop" {
		return nil, nil
	}
	fn, ok := tokenAuthFactories[mode]
	if !ok {
		return nil, fmt.Errorf("security: unknown token authenticator mode: %s (available: %s)", mode, tokenAuthModes())
	}
	return fn(cfg)
}

// AvailableModes 返回所有已注册的模式名称（用于调试和日志）。
func AvailableModes() (auths, tokenAuths, authzs, bills []string) {
	auths = keys(authFactories)
	tokenAuths = keys(tokenAuthFactories)
	authzs = keys(authzFactories)
	bills = keys(billFactories)
	return
}

func authModes() string      { return listKeys(authFactories) }
func tokenAuthModes() string { return listKeys(tokenAuthFactories) }
func authzModes() string     { return listKeys(authzFactories) }
func billModes() string      { return listKeys(billFactories) }

func listKeys[T any](m map[string]T) string {
	s := ""
	for k := range m {
		if s != "" {
			s += ", "
		}
		s += k
	}
	if s == "" {
		s = "<none>"
	}
	return s
}

func keys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
