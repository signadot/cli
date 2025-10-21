package models

import (
	"fmt"
	"math/rand"
	"time"
)

// MockDataService provides mock HTTP request data for testing
type MockDataService struct {
	requests []HTTPRequest
}

// NewMockDataService creates a new mock data service
func NewMockDataService() *MockDataService {
	service := &MockDataService{
		requests: generateMockRequests(),
	}
	return service
}

// GetRequests returns mock HTTP requests
func (m *MockDataService) GetRequests() []HTTPRequest {
	return m.requests
}

// GetRequestByID returns a specific request by ID
func (m *MockDataService) GetRequestByID(id string) *HTTPRequest {
	for _, req := range m.requests {
		if req.ID == id {
			return &req
		}
	}
	return nil
}

// generateMockRequests creates sample HTTP requests for testing
func generateMockRequests() []HTTPRequest {
	now := time.Now()
	requests := []HTTPRequest{
		{
			ID:         "req-001",
			Method:     "GET",
			URL:        "https://api.example.com/users",
			Path:       "/users",
			Headers:    map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token123"},
			Body:       "",
			Timestamp:  now.Add(-5 * time.Minute),
			Duration:   150 * time.Millisecond,
			StatusCode: 200,
			ClientIP:   "192.168.1.100",
			UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			Response: &HTTPResponse{
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json", "Content-Length": "1024"},
				Body:       `{"users": [{"id": 1, "name": "John Doe"}]}`,
				Size:       1024,
				Duration:   150 * time.Millisecond,
			},
		},
		{
			ID:         "req-002",
			Method:     "POST",
			URL:        "https://api.example.com/users",
			Path:       "/users",
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"name": "Jane Smith", "email": "jane@example.com"}`,
			Timestamp:  now.Add(-4 * time.Minute),
			Duration:   300 * time.Millisecond,
			StatusCode: 201,
			ClientIP:   "192.168.1.101",
			UserAgent:  "curl/7.68.0",
			Response: &HTTPResponse{
				StatusCode: 201,
				Headers:    map[string]string{"Content-Type": "application/json", "Location": "/users/2"},
				Body:       `{"id": 2, "name": "Jane Smith", "email": "jane@example.com"}`,
				Size:       512,
				Duration:   300 * time.Millisecond,
			},
		},
		{
			ID:         "req-003",
			Method:     "PUT",
			URL:        "https://api.example.com/users/1",
			Path:       "/users/1",
			Headers:    map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token123"},
			Body:       `{"name": "John Updated", "email": "john.updated@example.com"}`,
			Timestamp:  now.Add(-3 * time.Minute),
			Duration:   200 * time.Millisecond,
			StatusCode: 200,
			ClientIP:   "192.168.1.100",
			UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			Response: &HTTPResponse{
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"id": 1, "name": "John Updated", "email": "john.updated@example.com"}`,
				Size:       768,
				Duration:   200 * time.Millisecond,
			},
		},
		{
			ID:         "req-004",
			Method:     "DELETE",
			URL:        "https://api.example.com/users/2",
			Path:       "/users/2",
			Headers:    map[string]string{"Authorization": "Bearer token123"},
			Body:       "",
			Timestamp:  now.Add(-2 * time.Minute),
			Duration:   100 * time.Millisecond,
			StatusCode: 204,
			ClientIP:   "192.168.1.102",
			UserAgent:  "PostmanRuntime/7.26.8",
			Response: &HTTPResponse{
				StatusCode: 204,
				Headers:    map[string]string{},
				Body:       "",
				Size:       0,
				Duration:   100 * time.Millisecond,
			},
		},
		{
			ID:         "req-005",
			Method:     "GET",
			URL:        "https://api.example.com/users/999",
			Path:       "/users/999",
			Headers:    map[string]string{"Authorization": "Bearer token123"},
			Body:       "",
			Timestamp:  now.Add(-1 * time.Minute),
			Duration:   50 * time.Millisecond,
			StatusCode: 404,
			ClientIP:   "192.168.1.103",
			UserAgent:  "curl/7.68.0",
			Response: &HTTPResponse{
				StatusCode: 404,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"error": "User not found"}`,
				Size:       256,
				Duration:   50 * time.Millisecond,
			},
		},
	}

	for i := 0; i < 10; i++ {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		paths := []string{"/api/users", "/api/orders", "/api/products", "/api/auth/login", "/api/health"}
		statusCodes := []int{200, 201, 400, 401, 403, 404, 500}
		
		req := HTTPRequest{
			ID:         fmt.Sprintf("req-%03d", i+6),
			Method:     methods[rand.Intn(len(methods))],
			URL:        "https://api.example.com" + paths[rand.Intn(len(paths))],
			Path:       paths[rand.Intn(len(paths))],
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       generateRandomBody(),
			Timestamp:  now.Add(-time.Duration(rand.Intn(60)) * time.Minute),
			Duration:   time.Duration(rand.Intn(1000)) * time.Millisecond,
			StatusCode: statusCodes[rand.Intn(len(statusCodes))],
			ClientIP:   fmt.Sprintf("192.168.1.%d", 100+rand.Intn(50)),
			UserAgent:  "MockAgent/1.0",
		}
		
		if req.StatusCode >= 200 && req.StatusCode < 300 {
			req.Response = &HTTPResponse{
				StatusCode: req.StatusCode,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"success": true}`,
				Size:       int64(rand.Intn(1000)),
				Duration:   req.Duration,
			}
		}
		
		requests = append(requests, req)
	}

	return requests
}

// generateRandomBody creates a random JSON body for mock requests
func generateRandomBody() string {
	bodies := []string{
		`{"name": "Test User"}`,
		`{"email": "test@example.com", "password": "secret"}`,
		`{"query": "search term"}`,
		`{"id": 123, "status": "active"}`,
		`{"data": {"key": "value"}}`,
		"",
	}
	return bodies[rand.Intn(len(bodies))]
}
