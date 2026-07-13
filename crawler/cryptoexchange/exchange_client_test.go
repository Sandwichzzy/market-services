package cryptoexchange

import (
	"testing"
)

func TestBuildExchangeConfig_NoProxy(t *testing.T) {
	cfg, err := buildExchangeConfig("", "")
	if err != nil {
		t.Fatalf("buildExchangeConfig returned error: %v", err)
	}

	if _, ok := cfg["httpProxy"]; ok {
		t.Fatal("httpProxy should not be set when proxy is empty")
	}

	if _, ok := cfg["socksProxy"]; ok {
		t.Fatal("socksProxy should not be set when proxy is empty")
	}
}

func TestBuildExchangeConfig_HTTPProxy(t *testing.T) {
	cfg, err := buildExchangeConfig("http://127.0.0.1:7890", "http")
	if err != nil {
		t.Fatalf("buildExchangeConfig returned error: %v", err)
	}

	if got := cfg["httpProxy"]; got != "http://127.0.0.1:7890" {
		t.Fatalf("httpProxy = %v, want %q", got, "http://127.0.0.1:7890")
	}
}

func TestBuildExchangeConfig_Socks5Proxy(t *testing.T) {
	cfg, err := buildExchangeConfig("127.0.0.1:7890", "socks5")
	if err != nil {
		t.Fatalf("buildExchangeConfig returned error: %v", err)
	}

	if got := cfg["socksProxy"]; got != "127.0.0.1:7890" {
		t.Fatalf("socksProxy = %v, want %q", got, "127.0.0.1:7890")
	}
}

func TestBuildExchangeConfig_InvalidProxyType(t *testing.T) {
	if _, err := buildExchangeConfig("http://127.0.0.1:7890", "grpc"); err == nil {
		t.Fatal("buildExchangeConfig should reject unsupported proxy types")
	}
}
