/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package delete

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
)

type mockTransport struct {
	RoundTripFunc func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (errorReader) Close() error {
	return nil
}

func TestDelete_Success(t *testing.T) {
	mockResponse := `{"message": "Deleted successfully"}`
	mockTransport := &mockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://example.com/resource" && req.Method == http.MethodDelete {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(mockResponse)),
				}, nil
			}
			return nil, fmt.Errorf("unexpected URL or method")
		},
	}
	client := &http.Client{Transport: mockTransport}
	tool := &DeleteRequestTool{
		config: &Config{
			Headers: make(map[string]string),
		},
		client: client,
	}

	req := &DeleteRequest{URL: "https://example.com/resource"}
	result, err := tool.Delete(context.Background(), req)
	assert.NoError(t, err)

	assert.Equal(t, mockResponse, result)
}

func TestDelete_InvalidURL(t *testing.T) {
	tool := &DeleteRequestTool{
		config: &Config{
			Headers: make(map[string]string),
		},
		client: &http.Client{},
	}
	req := &DeleteRequest{URL: "http://:invalid"}
	_, err := tool.Delete(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestDelete_RequestError(t *testing.T) {
	mockTransport := &mockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	client := &http.Client{Transport: mockTransport}
	tool := &DeleteRequestTool{
		config: &Config{
			Headers: make(map[string]string),
		},
		client: client,
	}
	req := &DeleteRequest{URL: "https://example.com/resource"}
	_, err := tool.Delete(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute request")
}

func TestDelete_ReadBodyError(t *testing.T) {
	mockTransport := &mockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       errorReader{},
			}, nil
		},
	}
	client := &http.Client{Transport: mockTransport}
	tool := &DeleteRequestTool{
		config: &Config{
			Headers: make(map[string]string),
		},
		client: client,
	}
	req := &DeleteRequest{URL: "https://example.com/resource"}
	_, err := tool.Delete(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response body")
}

func TestConfig_Validate_Defaults(t *testing.T) {
	config := &Config{}
	err := config.validate()
	assert.NoError(t, err)
	assert.Equal(t, "requests_delete", config.ToolName)
	assert.NotEmpty(t, config.ToolDesc)
	assert.NotNil(t, config.Headers)
	assert.NotNil(t, config.HttpClient)
	assert.Equal(t, 30*time.Second, config.HttpClient.Timeout)
}

func TestConfig_Validate_WithValues(t *testing.T) {
	customClient := &http.Client{}
	config := &Config{
		ToolName:   "custom_delete",
		ToolDesc:   "Custom description",
		Headers:    map[string]string{"Authorization": "Bearer token"},
		HttpClient: customClient,
	}
	err := config.validate()
	assert.NoError(t, err)
	assert.Equal(t, "custom_delete", config.ToolName)
	assert.Equal(t, "Custom description", config.ToolDesc)
	assert.Equal(t, map[string]string{"Authorization": "Bearer token"}, config.Headers)
	assert.Equal(t, customClient, config.HttpClient)
}

func TestNewTool_Config(t *testing.T) {
	mockey.PatchConvey("NilConfig", t, func() {
		_, err := NewTool(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request tool configuration is required")
	})

	mockey.PatchConvey("WithConfig", t, func() {
		tool, err := NewTool(context.Background(), &Config{})
		assert.NoError(t, err)

		info, err := tool.Info(context.Background())
		assert.Nil(t, err)

		doc, err := info.ParamsOneOf.ToOpenAPIV3()
		assert.Nil(t, err)
		assert.Len(t, doc.Properties, 1)
		for _, v := range doc.Properties {
			assert.NotEqual(t, "", v.Value.Description)
		}
	})
}

func TestDelete_WithHeaders(t *testing.T) {
	var receivedHeaders http.Header
	mockTransport := &mockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			receivedHeaders = req.Header
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}
	client := &http.Client{Transport: mockTransport}
	tool := &DeleteRequestTool{
		config: &Config{
			Headers: map[string]string{
				"Authorization": "Bearer token",
				"User-Agent":    "test-agent",
			},
		},
		client: client,
	}

	req := &DeleteRequest{URL: "https://example.com/resource"}
	_, err := tool.Delete(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer token", receivedHeaders.Get("Authorization"))
	assert.Equal(t, "test-agent", receivedHeaders.Get("User-Agent"))
}
