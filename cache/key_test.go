package cache

import (
	"os"
	"reflect"
	"testing"
)

// Test structs for key generation
type TestUser struct {
	ID       int    `json:"id" cache:""`
	Name     string `json:"name" cache:""`
	Email    string `json:"email" cache:"optional"`
	Age      int    `json:"age" cache:"optional"`
	IsActive bool   `json:"is_active" cache:""`
}

type TestNestedUser struct {
	ID      int      `json:"id" cache:""`
	Profile TestUser `json:"profile" cache:""`
}

type TestUserNoDive struct {
	ID      int      `json:"id" cache:""`
	Profile TestUser `json:"profile" cache:"nodive,custom_key"`
}

type TestEmptyStruct struct{}

func TestParseTagsFast(t *testing.T) {
	tests := []struct {
		name         string
		cacheTag     string
		jsonTag      string
		fieldName    string
		fieldKind    reflect.Kind
		expectedInfo fieldInfo
	}{
		{
			name:         "basic cache tag",
			cacheTag:     "",
			jsonTag:      "user_id",
			fieldName:    "UserID",
			fieldKind:    reflect.Int,
			expectedInfo: fieldInfo{name: "user_id", isDive: false, optional: false},
		},
		{
			name:         "optional tag",
			cacheTag:     "optional",
			jsonTag:      "email",
			fieldName:    "Email",
			fieldKind:    reflect.String,
			expectedInfo: fieldInfo{name: "email", isDive: false, optional: true},
		},
		{
			name:         "struct with dive (empty cache tag)",
			cacheTag:     "",
			jsonTag:      "profile",
			fieldName:    "Profile",
			fieldKind:    reflect.Struct,
			expectedInfo: fieldInfo{name: "profile", isDive: true, optional: false},
		},
		{
			name:         "struct with nodive",
			cacheTag:     "nodive,custom",
			jsonTag:      "profile",
			fieldName:    "Profile",
			fieldKind:    reflect.Struct,
			expectedInfo: fieldInfo{name: "profile", isDive: false, optional: false},
		},
		{
			name:         "empty json tag uses field name",
			cacheTag:     "",
			jsonTag:      "",
			fieldName:    "UserName",
			fieldKind:    reflect.String,
			expectedInfo: fieldInfo{name: "username", isDive: false, optional: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTagsFast(tt.cacheTag, tt.jsonTag, tt.fieldName, tt.fieldKind)

			if result.name != tt.expectedInfo.name {
				t.Errorf("parseTagsFast() name = %v, want %v", result.name, tt.expectedInfo.name)
			}
			if result.isDive != tt.expectedInfo.isDive {
				t.Errorf("parseTagsFast() isDive = %v, want %v", result.isDive, tt.expectedInfo.isDive)
			}
			if result.optional != tt.expectedInfo.optional {
				t.Errorf("parseTagsFast() optional = %v, want %v", result.optional, tt.expectedInfo.optional)
			}
		})
	}
}

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero string", "", true},
		{"non-zero string", "hello", false},
		{"zero bool", false, true},
		{"non-zero bool", true, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"nil pointer", (*int)(nil), true},
		{"zero slice", []int{}, true},
		{"non-zero slice", []int{1, 2, 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := isZeroValue(v, v.Type())

			if result != tt.expected {
				t.Errorf("isZeroValue() = %v, want %v for value %v", result, tt.expected, tt.value)
			}
		})
	}
}

func TestGetKey(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		expectedKey string
		expectError bool
	}{
		{
			name:        "nil data",
			data:        nil,
			expectedKey: "",
			expectError: false,
		},
		{
			name: "simple struct",
			data: TestUser{
				ID:       123,
				Name:     "John Doe",
				IsActive: true,
			},
			expectedKey: "#id:123#name:John Doe#is_active:true",
			expectError: false,
		},
		{
			name: "struct with optional fields",
			data: TestUser{
				ID:   456,
				Name: "Jane Doe",
				// Email is empty but optional
				Age:      25,
				IsActive: false,
			},
			expectedKey: "#id:456#name:Jane Doe#age:25#is_active:false",
			expectError: false,
		},
		{
			name: "struct with missing required field",
			data: TestUser{
				ID: 789,
				// Name is required but empty
				IsActive: true,
			},
			expectedKey: "",
			expectError: true,
		},
		{
			name: "nested struct",
			data: TestNestedUser{
				ID: 111,
				Profile: TestUser{
					ID:       222,
					Name:     "Nested User",
					IsActive: true,
				},
			},
			expectedKey: "#id:111#profile:#id:222#name:Nested User#is_active:true",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := getKey(tt.data)

			if tt.expectError && err == nil {
				t.Errorf("getKey() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("getKey() unexpected error: %v", err)
			}
			if key != tt.expectedKey {
				t.Errorf("getKey() = %v, want %v", key, tt.expectedKey)
			}
		})
	}
}

func TestKey(t *testing.T) {
	// Set up environment
	_ = os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		_ = os.Unsetenv("SERVICE_NAME")
	}()

	tests := []struct {
		name        string
		serviceName string
		data        interface{}
		prefixes    []string
		expectedKey string
		expectError bool
	}{
		{
			name:        "nil data with no prefixes",
			serviceName: "user-service",
			data:        nil,
			prefixes:    nil,
			expectedKey: "user-service",
			expectError: false,
		},
		{
			name:        "nil data with prefixes",
			serviceName: "user-service",
			data:        nil,
			prefixes:    []string{"cache", "v1"},
			expectedKey: "user-service#cache#v1",
			expectError: false,
		},
		{
			name:        "struct data with prefixes",
			serviceName: "user-service",
			data: TestUser{
				ID:       123,
				Name:     "Test User",
				IsActive: true,
			},
			prefixes:    []string{"users", "active"},
			expectedKey: "user-service#TestUser#users#active#id:123#name:Test User#is_active:true",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := key(tt.serviceName, tt.data, tt.prefixes...)

			if tt.expectError && err == nil {
				t.Errorf("key() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("key() unexpected error: %v", err)
			}
			if key != tt.expectedKey {
				t.Errorf("key() = %v, want %v", key, tt.expectedKey)
			}
		})
	}
}

func TestKeyFunction(t *testing.T) {
	// Test the public Key function
	_ = os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		_ = os.Unsetenv("SERVICE_NAME")
	}()

	data := TestUser{
		ID:       123,
		Name:     "Test User",
		IsActive: true,
	}

	key := Key(data, "users")
	expected := "test-service#TestUser#users#id:123#name:Test User#is_active:true"

	if key != expected {
		t.Errorf("Key() = %v, want %v", key, expected)
	}

	// Test with missing SERVICE_NAME
	_ = os.Unsetenv("SERVICE_NAME")
	key = Key(data)
	if key != "" {
		t.Errorf("Key() should return empty string when SERVICE_NAME is not set, got %v", key)
	}
}

func TestExternalKey(t *testing.T) {
	data := TestUser{
		ID:       456,
		Name:     "External User",
		IsActive: false,
	}

	key := ExternalKey("external-service", data, "cache")
	expected := "external-service#TestUser#cache#id:456#name:External User#is_active:false"

	if key != expected {
		t.Errorf("ExternalKey() = %v, want %v", key, expected)
	}
}

// Benchmark tests
func BenchmarkGetKey(b *testing.B) {
	data := TestUser{
		ID:       123,
		Name:     "Benchmark User",
		Email:    "bench@example.com",
		Age:      30,
		IsActive: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getKey(data)
	}
}

func BenchmarkParseTagsFast(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseTagsFast("optional", "user_id", "UserID", reflect.String)
	}
}

func BenchmarkIsZeroValue(b *testing.B) {
	value := "test"
	v := reflect.ValueOf(value)
	t := v.Type()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isZeroValue(v, t)
	}
}
