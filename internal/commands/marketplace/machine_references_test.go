package marketplace

import (
	"1ctl/internal/api"
	"errors"
	"testing"
)

func TestMarketplaceHostnameResolutionUsesStableIDsAndDeduplicates(t *testing.T) {
	id := "bb98af6738d6f16a79092a5bae2ffdf3"
	calls := 0
	lookup := func(name string) (*api.Machine, error) {
		calls++
		if name != "worker-fixture" {
			t.Fatal("unexpected lookup")
		}
		return &api.Machine{MachineID: id}, nil
	}
	ids, err := resolveMarketplaceMachineReferences([]string{"worker-fixture", id}, lookup)
	if err != nil || len(ids) != 1 || ids[0] != id || calls != 1 {
		t.Fatalf("resolution failed: IDs=%v, calls=%d, err=%v", ids, calls, err)
	}
}

func TestMarketplaceHostnameResolutionFailsBeforeDeploymentOnLookupError(t *testing.T) {
	_, err := resolveMarketplaceMachineReferences([]string{"missing"}, func(string) (*api.Machine, error) { return nil, errors.New("not found") })
	if err == nil {
		t.Fatal("missing machine must fail")
	}
}
