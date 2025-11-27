package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

// mockNetcupAPIServer simulates the Netcup API for testing.
type mockNetcupAPIServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []APIRequest
	records  []netcupDNSRecord
	fail     bool
}

func newMockNetcupAPIServer() *mockNetcupAPIServer {
	mock := &mockNetcupAPIServer{
		records: []netcupDNSRecord{
			{ID: "1", Hostname: "@", Type: "A", Destination: "1.1.1.1"},
			{ID: "2", Hostname: "www", Type: "A", Destination: "1.1.1.1"},
			{ID: "3", Hostname: "*", Type: "A", Destination: "1.1.1.1"},
		},
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req APIRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(&testing.T{}, err)

		mock.mu.Lock()
		mock.requests = append(mock.requests, req)
		if mock.fail {
			w.WriteHeader(http.StatusInternalServerError)
			mock.mu.Unlock()
			return
		}
		mock.mu.Unlock()

		var resp APIResponse
		switch req.Action {
		case "login":
			resp = APIResponse{
				Status:       "success",
				StatusCode:   2000,
				ShortMessage: "Login successful",
				ResponseData: json.RawMessage(`{"apisessionid": "test-session-id"}`),
			}
		case "infoDnsRecords":
			resp = APIResponse{
				Status:       "success",
				StatusCode:   2000,
				ShortMessage: "DNS records successfully read",
				ResponseData: json.RawMessage(mock.marshalRecords()),
			}
		case "updateDnsRecords":
			var params UpdateDNSRecordsParams
			var paramBytes []byte
			switch value := req.Param.(type) {
			case json.RawMessage:
				paramBytes = value
			default:
				var err error
				paramBytes, err = json.Marshal(value)
				assert.NoError(&testing.T{}, err)
			}
			err := json.Unmarshal(paramBytes, &params)
			assert.NoError(&testing.T{}, err)
			mock.mu.Lock()
			mock.records = params.DNSRecordSet.DNSRecords
			mock.mu.Unlock()
			resp = APIResponse{
				Status:       "success",
				StatusCode:   2000,
				ShortMessage: "DNS records successfully updated",
			}

		case "logout":
			resp = APIResponse{
				Status:       "success",
				StatusCode:   2000,
				ShortMessage: "Logout successful",
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		assert.NoError(&testing.T{}, err)
	}))

	return mock
}

func (m *mockNetcupAPIServer) marshalRecords() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := json.Marshal(netcupDNSRecordSet{DNSRecords: m.records})
	assert.NoError(&testing.T{}, err)
	return string(data)
}

func (m *mockNetcupAPIServer) Close() {
	m.server.Close()
}

func TestNetcupProvider_GetRecord(t *testing.T) {
	mockServer := newMockNetcupAPIServer()
	defer mockServer.Close()

	client := NewNetcupClient("user", "key", "pass").WithEndpoint(mockServer.server.URL)

	// Test case: Record found
	record, err := client.GetRecord(context.Background(), "example.com", "www.example.com", "A")
	assert.NoError(t, err)
	assert.NotZero(t, record)
	assert.Equal(t, "www.example.com", record.Name)
	assert.Equal(t, "1.1.1.1", record.Value)

	// Test case: Record not found
	_, err = client.GetRecord(context.Background(), "example.com", "notfound.example.com", "A")
	assert.Error(t, err)
	assert.Equal(t, "record not found", err.Error())
}

func TestNetcupProvider_UpdateRecord(t *testing.T) {
	mockServer := newMockNetcupAPIServer()
	defer mockServer.Close()

	client := NewNetcupClient("user", "key", "pass").WithEndpoint(mockServer.server.URL)

	// Test case: Update existing record
	err := client.UpdateRecord(context.Background(), "example.com", &DNSRecord{
		Name:  "www.example.com",
		Type:  "A",
		Value: "2.2.2.2",
	})
	assert.NoError(t, err)

	// Verify the record was updated
	record, err := client.GetRecord(context.Background(), "example.com", "www.example.com", "A")
	assert.NoError(t, err)
	assert.Equal(t, "2.2.2.2", record.Value)

	// Test case: Create new record
	err = client.UpdateRecord(context.Background(), "example.com", &DNSRecord{
		Name:  "new.example.com",
		Type:  "A",
		Value: "3.3.3.3",
	})
	assert.NoError(t, err)

	// Verify the record was created
	record, err = client.GetRecord(context.Background(), "example.com", "new.example.com", "A")
	assert.NoError(t, err)
	assert.Equal(t, "3.3.3.3", record.Value)

	// Test case: No change needed
	mockServer.mu.Lock()
	initialRequestCount := len(mockServer.requests)
	mockServer.mu.Unlock()

	err = client.UpdateRecord(context.Background(), "example.com", &DNSRecord{
		Name:  "new.example.com",
		Type:  "A",
		Value: "3.3.3.3",
	})
	assert.NoError(t, err)

	// Verify no update API call was made
	mockServer.mu.Lock()
	// infoDnsRecords is called, but updateDnsRecords is not
	assert.Equal(t, initialRequestCount+1, len(mockServer.requests))
	mockServer.mu.Unlock()
}

func TestNetcupProvider_extractSubdomain(t *testing.T) {
	client := &NetcupClient{}
	testCases := []struct {
		hostname string
		domain   string
		expected string
	}{
		{"test.example.com", "example.com", "test"},
		{"*.example.com", "example.com", "*"},
		{"example.com", "example.com", "@"},
		{"sub.sub.example.com", "example.com", "sub.sub"},
		{"another.domain.net", "domain.net", "another"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.hostname, tc.domain), func(t *testing.T) {
			actual := client.extractSubdomain(tc.hostname, tc.domain)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestNetcupProvider_SessionHandling(t *testing.T) {
	mockServer := newMockNetcupAPIServer()
	defer mockServer.Close()

	client := NewNetcupClient("user", "key", "pass").WithEndpoint(mockServer.server.URL)
	client.sessionExpiry = time.Now().Add(-1 * time.Minute) // Force expiry

	// This call should trigger a login
	_, err := client.GetRecord(context.Background(), "example.com", "www.example.com", "A")
	assert.NoError(t, err)

	// Check that a login request was made
	mockServer.mu.Lock()
	assert.True(t, len(mockServer.requests) > 0)
	assert.Equal(t, "login", mockServer.requests[0].Action)
	mockServer.mu.Unlock()
}

func TestNetcupProvider_NewNetcupProvider(t *testing.T) {
	t.Setenv("NETCUP_CUSTOMER_NUMBER", "123")
	t.Setenv("NETCUP_API_KEY", "abc")
	t.Setenv("NETCUP_API_PASSWORD", "secret")
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	provider, err := NewNetcupProvider(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotZero(t, provider)
	assert.Equal(t, "netcup_ccp", provider.Name())

	// Test failure by clearing credentials and ensuring no credentials file
	t.Setenv("NETCUP_CUSTOMER_NUMBER", "")
	t.Setenv("NETCUP_API_KEY", "")
	t.Setenv("NETCUP_API_PASSWORD", "")
	_, err = NewNetcupProvider(context.Background(), nil)
	assert.Error(t, err)
}
