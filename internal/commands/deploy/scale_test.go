package deploy

import (
	"reflect"
	"strings"
	"testing"

	"1ctl/internal/api"
)

func TestPrepareScaleDeploymentIntentFailsClosedWithoutLosingStoredState(t *testing.T) {
	period := int32(10)
	stored := api.DeploymentIntent{
		Deployment: api.Deployment{
			Namespace: "tenant-a", AppLabel: "api", Image: "registry.example/api:v7",
			Port: 9090, Replicas: 2, CpuRequest: "300m", CPULimit: "2",
			MemoryRequest: "512Mi", MemoryLimit: "1Gi",
		},
		Environment: []api.KeyValuePair{{Key: "LOG_LEVEL", Value: "debug"}},
		Config: api.DeploymentDesiredStateConfig{
			ReadinessProbe: &api.DeploymentProbe{
				HTTPGet:       &api.DeploymentHTTPGetProbe{Path: "/ready", Port: 9090},
				PeriodSeconds: &period,
			},
			RequiredSecrets: []api.DeploymentRequiredSecret{{Key: "DATABASE_URL"}},
		},
		Volumes: []api.DeploymentIntentVolume{{
			VolumeName: "data", ClaimName: "api-data-pvc", StorageClass: "ceph-block",
			StorageSize: "20Gi", MountPath: "/data",
		}},
		Service:     &api.DeploymentIntentService{Name: "api", Port: 9090},
		PublicRoute: &api.DeploymentIntentPublicRoute{Kind: "default_dns"},
	}
	want := stored

	intent, err := prepareScaleDeploymentIntent(stored.Deployment, 5)
	if err == nil {
		t.Fatal("prepareScaleDeploymentIntent error = nil, want fail-closed error")
	}
	if !strings.Contains(err.Error(), "replica-only mutation") || !strings.Contains(err.Error(), "refusing to submit a partial deployment intent") {
		t.Fatalf("prepareScaleDeploymentIntent error = %q", err)
	}
	if !reflect.DeepEqual(intent, api.DeploymentIntent{}) {
		t.Fatalf("partial scale intent was produced: %+v", intent)
	}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored environment/config/volumes changed\n got: %+v\nwant: %+v", stored, want)
	}
}
