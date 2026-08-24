package thermal

import (
	"math"
	"testing"

	"task221-packthermal/internal/model"
)

func sampleLayers() []model.Layer {
	return []model.Layer{
		{Seq: 1, Material: "outer", ThicknessMM: 2.0, Conductivity: 0.45, Density: 950, SpecificHeat: 1900, AreaM2: 0.6},
		{Seq: 2, Material: "foam", ThicknessMM: 40.0, Conductivity: 0.022, Density: 35, SpecificHeat: 1400, AreaM2: 0.6},
		{Seq: 3, Material: "inner", ThicknessMM: 2.0, Conductivity: 0.45, Density: 950, SpecificHeat: 1900, AreaM2: 0.6},
	}
}

func TestBuildNetwork(t *testing.T) {
	n, err := BuildNetwork(sampleLayers())
	if err != nil {
		t.Fatalf("BuildNetwork: %v", err)
	}
	if n.RTotal <= 0 || n.CTotal <= 0 {
		t.Fatalf("expected positive R/C, got R=%v C=%v", n.RTotal, n.CTotal)
	}
	if got := n.RTotal * n.CTotal; math.Abs(got-n.Tau) > 1e-9 {
		t.Fatalf("tau mismatch: R*C=%v tau=%v", got, n.Tau)
	}
	// 保温层热阻应显著大于两侧薄壁。
	foamR := LayerResistance(sampleLayers()[1])
	outerR := LayerResistance(sampleLayers()[0])
	if foamR <= outerR*10 {
		t.Fatalf("foam resistance should dominate, foamR=%v outerR=%v", foamR, outerR)
	}
}

func TestBuildNetworkRejectsInvalid(t *testing.T) {
	if _, err := BuildNetwork(nil); err == nil {
		t.Fatal("expected error for empty stack")
	}
	bad := sampleLayers()
	bad[0].ThicknessMM = 0
	if _, err := BuildNetwork(bad); err == nil {
		t.Fatal("expected error for non-positive thickness")
	}
}

func TestPredict(t *testing.T) {
	n, err := BuildNetwork(sampleLayers())
	if err != nil {
		t.Fatalf("BuildNetwork: %v", err)
	}
	const init, env = 5.0, 25.0
	if got := n.Predict(0, init, env); math.Abs(got-init) > 1e-9 {
		t.Fatalf("Predict(0) should equal init, got %v", got)
	}
	// 长时间后趋近环境温度。
	if got := n.Predict(1e9, init, env); math.Abs(got-env) > 1e-3 {
		t.Fatalf("Predict(inf) should approach env, got %v", got)
	}
	// 单调上升。
	prev := n.Predict(0, init, env)
	for _, ts := range []float64{10, 100, 1000, 10000} {
		cur := n.Predict(ts, init, env)
		if cur < prev {
			t.Fatalf("response not monotonic: %v -> %v", prev, cur)
		}
		prev = cur
	}
}
