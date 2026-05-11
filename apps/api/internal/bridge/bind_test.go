package bridge

import "testing"

func TestValidateBindAddressAcceptsLoopback(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:4761", "localhost:4761", "[::1]:4761"} {
		if err := ValidateBindAddress(addr, false); err != nil {
			t.Fatalf("expected %s to be accepted: %v", addr, err)
		}
	}
}

func TestValidateBindAddressRejectsWildcardAndPublic(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:4761", "[::]:4761", "8.8.8.8:4761", "example.com:4761"} {
		if err := ValidateBindAddress(addr, false); err == nil {
			t.Fatalf("expected %s to be rejected", addr)
		}
	}
}

func TestValidateBindAddressPrivateOptIn(t *testing.T) {
	if err := ValidateBindAddress("172.17.0.1:4761", false); err == nil {
		t.Fatal("expected private non-loopback bind to require opt-in")
	}
	if err := ValidateBindAddress("172.17.0.1:4761", true); err != nil {
		t.Fatalf("expected private bind with opt-in to pass: %v", err)
	}
}

func TestLocalhostURLForBind(t *testing.T) {
	got, err := LocalhostURLForBind("127.0.0.1:4761")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://127.0.0.1:4761" {
		t.Fatalf("expected http://127.0.0.1:4761, got %q", got)
	}
}
