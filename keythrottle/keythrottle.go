package keythrottle

import (
	"fmt"
	"sync"
)

const TIER_UNAUTHENTICATED = 0
const TIER_A = 1
const TIER_B = 2

type Tier int

type AuthTierStorage struct {
	TierA map[string]string `json:"tier_a"`
	TierB map[string]string `json:"tier_b"`
}
type KeyThrottle struct {
	tierA     map[string]struct{}
	tierB     map[string]struct{}
	tierMutex sync.Mutex
}

func BuildKeyThrottle() *KeyThrottle {
	return &KeyThrottle{
		tierA:     make(map[string]struct{}),
		tierB:     make(map[string]struct{}),
		tierMutex: sync.Mutex{},
	}
}

func (kt *KeyThrottle) SetTiers(tiers AuthTierStorage) error {
	if tiers.TierB == nil {
		return fmt.Errorf("tier_b is nil")
	}
	if tiers.TierA == nil {
		return fmt.Errorf("tier_a is nil")
	}
	newTierA := make(map[string]struct{})
	for _, key := range tiers.TierA {
		newTierA[key] = struct{}{}
	}
	newTierB := make(map[string]struct{})
	for _, key := range tiers.TierB {
		newTierB[key] = struct{}{}
	}
	kt.tierMutex.Lock()
	kt.tierA = newTierA
	kt.tierB = newTierB
	kt.tierMutex.Unlock()
	return nil
}

func (kt *KeyThrottle) GetTierFromKey(key string) Tier {
	if key == "" {
		return TIER_UNAUTHENTICATED
	}
	// avoid panic
	if kt.tierA == nil || kt.tierB == nil {
		return TIER_UNAUTHENTICATED
	}
	if _, exists := kt.tierA[key]; exists {
		return TIER_A
	} else if _, exists := kt.tierB[key]; exists {
		return TIER_B
	}
	return TIER_UNAUTHENTICATED
}
