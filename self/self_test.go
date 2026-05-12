package self_test

import (
	"strings"
	"testing"

	"github.com/kaichao/gopkg/self"
)

// Named function for nested call
func nestedFunc() string {
	return self.GetCurrentGoroutineStack()
}

// Named function for deep recursive call
func deepStack(depth int) string {
	if depth == 0 {
		return self.GetCurrentGoroutineStack()
	}
	return deepStack(depth - 1)
}

// Struct for GetFunctionName demonstration
type demoStruct struct{}

func (d *demoStruct) PointerMethod() {}
func (d demoStruct)  ValueMethod()   {}

func topLevelFunc() {}

func TestGetFunctionName(t *testing.T) {
	// Basic: top-level function, use '.' to get the short name
	name := self.GetFunctionName(topLevelFunc, '.')
	if name != "topLevelFunc" {
		t.Errorf("expected 'topLevelFunc', got '%s'", name)
	}

	// Pointer method, use '.' to get method name only
	name = self.GetFunctionName((*demoStruct).PointerMethod, '.')
	if name != "PointerMethod" {
		t.Errorf("expected 'PointerMethod', got '%s'", name)
	}

	// Value method, use '.' to get method name only
	name = self.GetFunctionName(demoStruct.ValueMethod, '.')
	if name != "ValueMethod" {
		t.Errorf("expected 'ValueMethod', got '%s'", name)
	}

	// Custom separator: split by '/' returns the final path segment
	name = self.GetFunctionName(topLevelFunc, '/')
	if !strings.Contains(name, "topLevelFunc") {
		t.Errorf("expected name to contain 'topLevelFunc', got '%s'", name)
	}

	// Multiple separators
	name = self.GetFunctionName((*demoStruct).PointerMethod, '.', '-')
	if name != "PointerMethod" {
		t.Errorf("expected 'PointerMethod', got '%s'", name)
	}
}

func TestGetCurrentGoroutineStack(t *testing.T) {
	// Test 1: Basic call, check for current function name
	stack := self.GetCurrentGoroutineStack()
	if !strings.Contains(stack, "TestGetCurrentGoroutineStack") {
		t.Errorf("Expected stack to contain 'TestGetCurrentGoroutineStack', got:\n%s", stack)
	}

	// Test 2: Nested call, verify multi-layer stack
	stack = nestedFunc()
	if !strings.Contains(stack, "nestedFunc") || !strings.Contains(stack, "TestGetCurrentGoroutineStack") {
		t.Errorf("Expected stack to contain 'nestedFunc' and 'TestGetCurrentGoroutineStack', got:\n%s", stack)
	}

	// Test 3: Deep call, simulate complex stack before error
	stack = deepStack(5) // Simulate 5 layers of recursion
	if !strings.Contains(stack, "deepStack") {
		t.Errorf("Expected stack to contain 'deepStack', got:\n%s", stack)
	}
}
