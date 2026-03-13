package client_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/grafana/nanogit/protocol/client"
)

// Example_protocolV1Detection demonstrates the error message
// users will see when attempting to connect to a Git server that only
// supports protocol v1.
func Example_protocolV1Detection() {
	// Create a test server that returns a v1-style response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a Git protocol v1 response with ref advertisements
		v1Response := "003f1234567890abcdef1234567890abcdef12345678 refs/heads/main\000capabilities\n0000"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(v1Response))
	}))
	defer server.Close()

	// Attempt to connect to the v1-only server
	rawClient, err := client.NewRawClient(server.URL + "/repo")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	err = rawClient.SmartInfo(context.Background(), "git-upload-pack")
	if err != nil {
		// Check if it's the protocol v1 error
		if errors.Is(err, client.ErrProtocolV1NotSupported) {
			fmt.Println("Error: Git protocol v1 detected")
			fmt.Println("The server only supports protocol v1")
			fmt.Println("Modern Git servers support protocol v2")
		}
		return
	}

	fmt.Println("Connection successful")
	// Output:
	// Error: Git protocol v1 detected
	// The server only supports protocol v1
	// Modern Git servers support protocol v2
}

// Example_protocolV2Success demonstrates successful connection
// to a Git server that supports protocol v2.
func Example_protocolV2Success() {
	// Create a test server that returns a v2-style response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a Git protocol v2 response
		v2Response := "000eversion 2\n0000"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(v2Response))
	}))
	defer server.Close()

	// Connect to the v2 server
	rawClient, err := client.NewRawClient(server.URL + "/repo")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	err = rawClient.SmartInfo(context.Background(), "git-upload-pack")
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		return
	}

	fmt.Println("Successfully connected to Git server with protocol v2 support")
	// Output:
	// Successfully connected to Git server with protocol v2 support
}
