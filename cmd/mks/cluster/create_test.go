package cluster

import (
	"strings"
	"testing"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
)

func TestCreateOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    createOptions
		wantErr string
	}{
		{
			name:    "missing sizing",
			opts:    createOptions{Name: "c", VolumeGB: 50, VolumeType: "fast"},
			wantErr: "nodegroup sizing is required",
		},
		{
			name:    "cpus without ram",
			opts:    createOptions{Name: "c", CPUs: 4, VolumeGB: 50, VolumeType: "fast"},
			wantErr: "nodegroup sizing is required",
		},
		{
			name:    "missing volume",
			opts:    createOptions{Name: "c", CPUs: 4, RAMMB: 8192},
			wantErr: "nodegroup volume is required",
		},
		{
			name: "local volume ok",
			opts: createOptions{Name: "c", CPUs: 4, RAMMB: 8192, LocalVolume: true},
		},
		{
			name: "flavor and volume ok",
			opts: createOptions{Name: "c", FlavorID: "f1", VolumeGB: 50, VolumeType: "fast"},
		},
		{
			name:    "invalid cluster type",
			opts:    createOptions{Name: "c", FlavorID: "f1", VolumeGB: 50, VolumeType: "fast", ClusterType: "BOGUS"},
			wantErr: "invalid --cluster-type",
		},
		{
			name:    "invalid cni type",
			opts:    createOptions{Name: "c", CPUs: 4, RAMMB: 8192, LocalVolume: true, CNIType: "WEIRD"},
			wantErr: "invalid --cni-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate() expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate() error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCreateOptionsClusterOpts(t *testing.T) {
	opts := createOptions{
		Name:           "my-cluster",
		KubeVersion:    "1.29.1",
		Region:         "ru-3",
		ClusterType:    "high_availability",
		CNIType:        "cilium",
		PrivateKubeAPI: false,
		Count:          2,
		CPUs:           4,
		RAMMB:          8192,
		VolumeGB:       50,
		VolumeType:     "fast.ru-3a",
	}

	co := opts.clusterOpts()

	if co.Name != "my-cluster" {
		t.Errorf("Name = %q", co.Name)
	}
	if co.KubeVersion != "1.29.1" {
		t.Errorf("KubeVersion = %q", co.KubeVersion)
	}
	if len(co.Nodegroups) != 1 {
		t.Fatalf("expected 1 nodegroup, got %d", len(co.Nodegroups))
	}
	ng := co.Nodegroups[0]
	if ng.Count != 2 || ng.CPUs != 4 || ng.RAMMB != 8192 || ng.VolumeType != "fast.ru-3a" {
		t.Errorf("nodegroup mismatch: %+v", ng)
	}
	if co.ClusterType == nil || *co.ClusterType != mkscluster.ClusterTypeHighAvailability {
		t.Errorf("ClusterType = %v", co.ClusterType)
	}
	if co.CNIType != mkscluster.CNITypeCilium {
		t.Errorf("CNIType = %q", co.CNIType)
	}
	if co.PrivateKubeAPI == nil || *co.PrivateKubeAPI {
		t.Errorf("PrivateKubeAPI = %v", co.PrivateKubeAPI)
	}
}
