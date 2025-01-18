package keythrottle

import (
	"reflect"
	"testing"
)

func TestKeyThrottle_GetTierFromKey(t *testing.T) {
	type fields struct {
		tierA map[string]struct{}
		tierB map[string]struct{}
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Tier
	}{
		{
			name: "no tiers",
			fields: fields{
				tierA: make(map[string]struct{}),
				tierB: make(map[string]struct{}),
			},
			args: args{
				key: "123456",
			},
			want: TIER_UNAUTHENTICATED,
		},
		{
			name: "not in any tier",
			fields: fields{
				tierA: map[string]struct{}{
					"1111": struct{}{},
				},
				tierB: map[string]struct{}{
					"2222": struct{}{},
				},
			},
			args: args{
				key: "123456",
			},
			want: TIER_UNAUTHENTICATED,
		},
		{
			name: "is b tier",
			fields: fields{
				tierA: map[string]struct{}{
					"1111": struct{}{},
				},
				tierB: map[string]struct{}{
					"2222": struct{}{},
				},
			},
			args: args{
				key: "2222",
			},
			want: TIER_B,
		},
		{
			name: "is a tier",
			fields: fields{
				tierA: map[string]struct{}{
					"1111": struct{}{},
				},
				tierB: map[string]struct{}{
					"2222": struct{}{},
				},
			},
			args: args{
				key: "1111",
			},
			want: TIER_A,
		},
		{
			name: "a not initialized",
			fields: fields{
				tierA: nil,
				tierB: map[string]struct{}{
					"1111": struct{}{},
				},
			},
			args: args{
				key: "1111",
			},
			want: TIER_UNAUTHENTICATED,
		},
		{
			name: "B not initialized",
			fields: fields{
				tierA: map[string]struct{}{
					"1111": struct{}{},
				},
				tierB: nil,
			},
			args: args{
				key: "1111",
			},
			want: TIER_UNAUTHENTICATED,
		},
		{
			name: "empty key",
			fields: fields{
				tierA: map[string]struct{}{
					"": struct{}{},
				},
				tierB: map[string]struct{}{
					"2222": struct{}{},
				},
			},
			args: args{
				key: "",
			},
			want: TIER_UNAUTHENTICATED,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kt := &KeyStore{
				tierA: tt.fields.tierA,
				tierB: tt.fields.tierB,
			}
			if got := kt.GetTierFromKey(tt.args.key); got != tt.want {
				t.Errorf("GetTierFromKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyThrottle_BuildKeyThrottle(t *testing.T) {
	kt := BuildKeyStore()
	if kt.tierA == nil {
		t.Errorf("BuildKeyStore() tierA is nil")
	}
	if kt.tierB == nil {
		t.Errorf("BuildKeyStore() tierB is nil")
	}
}

func TestKeyThrottle_SetTiers(t *testing.T) {
	type fields struct {
		tierA map[string]struct{}
		tierB map[string]struct{}
	}
	type args struct {
		tiers AuthTierStorage
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		checkKey string
		wantTier Tier
	}{
		{
			name: "both nil",
			fields: fields{
				tierA: nil,
				tierB: nil,
			},
			args: args{
				tiers: AuthTierStorage{
					TierA: nil,
					TierB: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "a nil",
			fields: fields{
				tierA: nil,
				tierB: nil,
			},
			args: args{
				tiers: AuthTierStorage{
					TierA: map[string]string{"a": "b"},
					TierB: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "b nil",
			fields: fields{
				tierA: nil,
				tierB: nil,
			},
			args: args{
				tiers: AuthTierStorage{
					TierA: nil,
					TierB: map[string]string{"a": "b"},
				},
			},
			wantErr: true,
		},
		{
			name: "a: b, c: d",
			fields: fields{
				tierA: nil,
				tierB: nil,
			},
			args: args{
				tiers: AuthTierStorage{
					TierA: map[string]string{"a": "b"},
					TierB: map[string]string{"c": "d"},
				},
			},
			wantErr:  false,
			checkKey: "b",
			wantTier: TIER_A,
		},
		{
			name: "a: b, c: d, get b",
			fields: fields{
				tierA: nil,
				tierB: nil,
			},
			args: args{
				tiers: AuthTierStorage{
					TierA: map[string]string{"a": "b"},
					TierB: map[string]string{"c": "d"},
				},
			},
			wantErr:  false,
			checkKey: "d",
			wantTier: TIER_B,
		},
		{
			name: "a: b, c: d, get d",
			fields: fields{
				tierA: nil,
				tierB: nil,
			},
			args: args{
				tiers: AuthTierStorage{
					TierA: map[string]string{"a": "b"},
					TierB: map[string]string{"c": "d"},
				},
			},
			wantErr:  false,
			checkKey: "d",
			wantTier: TIER_B,
		},
		{
			name: "a: b, c: d, get z",
			fields: fields{
				tierA: nil,
				tierB: nil,
			},
			args: args{
				tiers: AuthTierStorage{
					TierA: map[string]string{"a": "b"},
					TierB: map[string]string{"c": "d"},
				},
			},
			wantErr:  false,
			checkKey: "z",
			wantTier: TIER_UNAUTHENTICATED,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kt := &KeyStore{
				tierA: tt.fields.tierA,
				tierB: tt.fields.tierB,
			}
			if err := kt.SetTiers(tt.args.tiers); (err != nil) != tt.wantErr {
				t.Errorf("SetTiers() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				result := kt.GetTierFromKey(tt.checkKey)
				if !reflect.DeepEqual(result, tt.wantTier) {
					t.Errorf("GetTierFromKey() = %v, want %v", result, tt.wantTier)
				}
			}
		})
	}
}
