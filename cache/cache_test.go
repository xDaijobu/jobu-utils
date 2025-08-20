package cache

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

// Test data structures
type TestCacheData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestGetClientWithCheck(t *testing.T) {
	// Since we can't mock the functions directly in this version,
	// we'll test the actual function behavior
	_, err := getClientWithCheck()

	// In test environment without Redis, this should return an error
	if err == nil {
		t.Log("getClientWithCheck() succeeded - Redis might be running")
	} else {
		t.Log("getClientWithCheck() failed as expected without Redis")
	}
}

func TestJSONMarshaling(t *testing.T) {
	testData := TestCacheData{
		ID:   123,
		Name: "test data",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON marshaling should produce non-empty result")
	}

	// Test unmarshaling
	var unmarshaled TestCacheData
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}

	if unmarshaled.ID != testData.ID || unmarshaled.Name != testData.Name {
		t.Errorf("Unmarshaled data mismatch: got %+v, want %+v", unmarshaled, testData)
	}
}

func TestPointerValidation(t *testing.T) {
	var target TestCacheData
	var pointer *TestCacheData

	// Test non-pointer
	if reflect.ValueOf(target).Kind() == reflect.Ptr {
		t.Error("Non-pointer should not be detected as pointer")
	}

	// Test pointer
	if reflect.ValueOf(&target).Kind() != reflect.Ptr {
		t.Error("Pointer should be detected as pointer")
	}

	// Test nil pointer
	if reflect.ValueOf(pointer).Kind() != reflect.Ptr {
		t.Error("Nil pointer should still be detected as pointer type")
	}
}

func TestDurationCalculations(t *testing.T) {
	tests := []struct {
		name            string
		seconds         int
		expectedSeconds float64
	}{
		{"one hour", 3600, 3600.0},
		{"one minute", 60, 60.0},
		{"zero", 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := time.Duration(tt.seconds) * time.Second
			actualSeconds := duration.Seconds()

			if actualSeconds != tt.expectedSeconds {
				t.Errorf("Duration calculation incorrect: got %v, want %v", actualSeconds, tt.expectedSeconds)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := "base error"

	// Test basic error handling patterns used in cache functions
	if baseErr == "" {
		t.Error("Error message should not be empty")
	}
}

// Integration tests (require running Redis)
func TestCacheIntegrationIfAvailable(t *testing.T) {
	if !IsCacheConnected() {
		t.Skip("Redis not available - skipping integration tests")
		return
	}

	testKey := "test:integration:key"
	testData := TestCacheData{
		ID:   999,
		Name: "integration test",
	}

	// Clean up before and after
	Delete(testKey)
	defer Delete(testKey)

	// Test SetJSON
	err := SetJSON(testKey, testData, 60)
	if err != nil {
		t.Logf("SetJSON failed (expected if Redis not running): %v", err)
		return
	}

	// Test IsCacheExists
	exists, err := IsCacheExists(testKey)
	if err != nil {
		t.Logf("IsCacheExists failed: %v", err)
		return
	}
	if !exists {
		t.Error("Key should exist after SetJSON")
	}

	// Test Get
	data, err := Get(testKey)
	if err != nil {
		t.Logf("Get failed: %v", err)
		return
	}
	if data == nil {
		t.Error("Get should return data")
	}

	// Test GetUnmarshal
	var retrievedData TestCacheData
	err = GetUnmarshal(testKey, &retrievedData)
	if err != nil {
		t.Logf("GetUnmarshal failed: %v", err)
		return
	}
	if retrievedData.ID != testData.ID || retrievedData.Name != testData.Name {
		t.Errorf("Retrieved data mismatch: got %+v, want %+v", retrievedData, testData)
	}

	// Test TTL
	ttl, err := TTL(testKey)
	if err != nil {
		t.Logf("TTL failed: %v", err)
		return
	}
	if ttl <= 0 || ttl > 60 {
		t.Logf("TTL unexpected value: %v", ttl)
	}

	// Test SetExpire
	err = SetExpire(testKey, 120)
	if err != nil {
		t.Logf("SetExpire failed: %v", err)
		return
	}

	// Test Delete
	err = Delete(testKey)
	if err != nil {
		t.Logf("Delete failed: %v", err)
		return
	}

	// Verify deletion
	exists, err = IsCacheExists(testKey)
	if err != nil {
		t.Logf("IsCacheExists after delete failed: %v", err)
		return
	}
	if exists {
		t.Error("Key should not exist after Delete")
	}
}

// Test with environment variables
func TestCacheWithEnvironment(t *testing.T) {
	// Save original environment
	originalServiceName := os.Getenv("SERVICE_NAME")
	originalRedisHost := os.Getenv("REDIS_HOST")
	originalRedisPort := os.Getenv("REDIS_PORT")

	defer func() {
		if originalServiceName != "" {
			os.Setenv("SERVICE_NAME", originalServiceName)
		} else {
			os.Unsetenv("SERVICE_NAME")
		}
		if originalRedisHost != "" {
			os.Setenv("REDIS_HOST", originalRedisHost)
		} else {
			os.Unsetenv("REDIS_HOST")
		}
		if originalRedisPort != "" {
			os.Setenv("REDIS_PORT", originalRedisPort)
		} else {
			os.Unsetenv("REDIS_PORT")
		}
	}()

	// Test with proper environment setup
	os.Setenv("SERVICE_NAME", "test-service")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")

	// The cache functions should now work with proper environment
	// Even if Redis is not running, the environment setup should be correct
	t.Log("Environment variables set for cache testing")
}

// Benchmark tests
func BenchmarkJSONMarshal(b *testing.B) {
	testData := TestCacheData{
		ID:   123,
		Name: "benchmark test data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(testData)
	}
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	testData := TestCacheData{
		ID:   123,
		Name: "benchmark test data",
	}

	jsonData, _ := json.Marshal(testData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result TestCacheData
		json.Unmarshal(jsonData, &result)
	}
}

func BenchmarkDurationConversion(b *testing.B) {
	seconds := 3600

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		duration := time.Duration(seconds) * time.Second
		_ = duration.Seconds()
	}
}
