package main

import (
	"bgp_dashboard/pkg"
	"testing"
	"time"
)

// TestBGPServiceInitialization verifies that the BGP service can be created
func TestBGPServiceInitialization(t *testing.T) {
	bgpService := pkg.NewBGPService()
	if bgpService == nil {
		t.Fatal("BGP service should not be nil")
	}
}

// TestBGPServiceStart tests the BGP server start functionality
func TestBGPServiceStart(t *testing.T) {
	bgpService := pkg.NewBGPService()

	tests := []struct {
		name        string
		routerId    string
		asn         uint32
		expectError bool
	}{
		{
			name:        "Valid configuration",
			routerId:    "192.168.1.213",
			asn:         65001,
			expectError: false,
		},
		{
			name:        "Invalid router ID",
			routerId:    "invalid.ip.address",
			asn:         65001,
			expectError: true,
		},
		{
			name:        "Invalid ASN",
			routerId:    "192.168.1.213",
			asn:         0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bgpService.Start(tt.routerId, tt.asn)
			if (err != nil) != tt.expectError {
				t.Errorf("Start() error = %v, expectError %v", err, tt.expectError)
			}
			// Clean up after each test
			bgpService.Stop()
		})
	}
}

// TestBGPNeighborConfiguration tests the neighbor configuration functionality
func TestBGPNeighborConfiguration(t *testing.T) {
	bgpService := pkg.NewBGPService()

	// Start the BGP service first
	err := bgpService.Start("192.168.1.213", 65001)
	if err != nil {
		t.Fatalf("Failed to start BGP service: %v", err)
	}
	defer bgpService.Stop()

	tests := []struct {
		name        string
		peerIP      string
		peerASN     uint32
		expectError bool
	}{
		{
			name:        "Valid neighbor configuration",
			peerIP:      "192.168.1.89",
			peerASN:     65002,
			expectError: false,
		},
		{
			name:        "Invalid peer IP",
			peerIP:      "invalid.ip",
			peerASN:     65002,
			expectError: true,
		},
		{
			name:        "Invalid peer ASN",
			peerIP:      "192.168.1.89",
			peerASN:     0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bgpService.AddNeighbor(tt.peerIP, tt.peerASN)
			if (err != nil) != tt.expectError {
				t.Errorf("AddNeighbor() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestMonitorPrefixes tests the prefix monitoring functionality
func TestMonitorPrefixes(t *testing.T) {
	bgpService := pkg.NewBGPService()

	// Start the BGP service
	err := bgpService.Start("192.168.1.213", 65001)
	if err != nil {
		t.Fatalf("Failed to start BGP service: %v", err)
	}
	defer bgpService.Stop()

	// Add a neighbor
	err = bgpService.AddNeighbor("192.168.1.89", 65002)
	if err != nil {
		t.Fatalf("Failed to add neighbor: %v", err)
	}

	// Start monitoring in a goroutine
	done := make(chan bool)
	go func() {
		bgpService.MonitorPrefixes()
		done <- true
	}()

	// Wait for a short period to ensure monitoring starts
	select {
	case <-done:
		t.Error("MonitorPrefixes() returned unexpectedly")
	case <-time.After(2 * time.Second):
		// Success - monitoring is still running
	}
}

// TestBGPServiceIntegration tests the full integration of all components
func TestBGPServiceIntegration(t *testing.T) {
	bgpService := pkg.NewBGPService()

	// Test the complete flow
	t.Run("Complete BGP setup flow", func(t *testing.T) {
		// 1. Start the service
		err := bgpService.Start("192.168.1.213", 65001)
		if err != nil {
			t.Fatalf("Failed to start BGP service: %v", err)
		}

		// 2. Add a neighbor
		err = bgpService.AddNeighbor("192.168.1.89", 65002)
		if err != nil {
			t.Fatalf("Failed to add neighbor: %v", err)
		}

		// 3. Start monitoring
		monitoringStarted := make(chan bool)
		go func() {
			bgpService.MonitorPrefixes()
			monitoringStarted <- true
		}()

		// 4. Verify monitoring started
		select {
		case <-monitoringStarted:
			t.Error("MonitorPrefixes() returned unexpectedly")
		case <-time.After(2 * time.Second):
			// Success - monitoring is running
		}

		// 5. Clean up
		bgpService.Stop()
	})
}
