package main

import "testing"

func TestShouldFailDeterministic(t *testing.T) {
	// Same input always yields the same decision.
	for i := 0; i < 5; i++ {
		if shouldFail("order-42", 8) != shouldFail("order-42", 8) {
			t.Fatal("shouldFail must be deterministic")
		}
	}
	// 0% never fails, 100% always fails.
	if shouldFail("anything", 0) {
		t.Fatal("0%% must never fail")
	}
	if !shouldFail("anything", 100) {
		t.Fatal("100%% must always fail")
	}
}

func TestShouldFailRateRoughlyMatches(t *testing.T) {
	fails := 0
	const n = 10000
	for i := 0; i < n; i++ {
		if shouldFail("order-"+string(rune(i)), 8) {
			fails++
		}
	}
	rate := float64(fails) / float64(n)
	if rate < 0.05 || rate > 0.11 {
		t.Fatalf("fail rate %.3f outside expected band for 8%%", rate)
	}
}
