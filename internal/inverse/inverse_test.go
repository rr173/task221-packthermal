package inverse

import (
	"math"
	"testing"

	"task221-packthermal/internal/model"
)

func genRC(initTemp, envTemp, tau float64, n int, step float64) []model.TempSample {
	out := make([]model.TempSample, 0, n)
	for i := 0; i < n; i++ {
		t := float64(i) * step
		v := envTemp - (envTemp-initTemp)*math.Exp(-t/tau)
		out = append(out, model.TempSample{Seconds: t, Temperature: v})
	}
	return out
}

func TestFitRCRecoversTau(t *testing.T) {
	const (
		init  = 5.0
		env   = 25.0
		tau   = 4500.0
		cPrior = 12000.0
	)
	samples := genRC(init, env, tau, 61, 60)
	res := FitRC(FitInput{InitTemp: init, EnvTemp: env, Samples: samples, CapPrior: cPrior})
	if !res.Converged {
		t.Fatalf("fit did not converge: %+v", res)
	}
	if math.Abs(res.Tau-tau)/tau > 0.02 {
		t.Fatalf("tau recovery off: got %.2f want %.2f", res.Tau, tau)
	}
	if math.Abs(res.CEff-cPrior) > 1e-9 {
		t.Fatalf("c_eff should equal prior: got %v want %v", res.CEff, cPrior)
	}
	wantR := tau / cPrior
	if math.Abs(res.REff-wantR)/wantR > 0.02 {
		t.Fatalf("r_eff off: got %.4f want %.4f", res.REff, wantR)
	}
	// 无噪声拟合残差应接近零。
	if res.Rms > 0.5 {
		t.Fatalf("rms too large: %v", res.Rms)
	}
}

func TestFitRCUsesResPrior(t *testing.T) {
	const (
		init    = 5.0
		env     = 25.0
		tau     = 3000.0
		resPrior = 2.5
	)
	samples := genRC(init, env, tau, 61, 60)
	res := FitRC(FitInput{InitTemp: init, EnvTemp: env, Samples: samples, ResPrior: resPrior})
	if math.Abs(res.REff-resPrior) > 1e-9 {
		t.Fatalf("r_eff should equal prior: got %v want %v", res.REff, resPrior)
	}
	wantC := tau / resPrior
	if math.Abs(res.CEff-wantC)/wantC > 0.02 {
		t.Fatalf("c_eff off: got %.4f want %.4f", res.CEff, wantC)
	}
}

func TestDetectAnomalies(t *testing.T) {
	rms := map[int64]float64{
		1: 0.05, // 正常
		2: 0.06, // 正常
		3: 3.40, // 接触不良
	}
	anomalies := DetectAnomalies(rms, 3.0)
	if len(anomalies) != 1 || anomalies[0] != 3 {
		t.Fatalf("expected sensor 3 as anomaly, got %v", anomalies)
	}
}

func TestDetectAnomaliesNoOutlier(t *testing.T) {
	rms := map[int64]float64{1: 0.05, 2: 0.06, 3: 0.07}
	if anomalies := DetectAnomalies(rms, 3.0); len(anomalies) != 0 {
		t.Fatalf("expected no anomaly, got %v", anomalies)
	}
}

func TestRMS(t *testing.T) {
	if got := RMS([]float64{3, 4}); math.Abs(got-math.Sqrt(12.5)) > 1e-9 {
		t.Fatalf("RMS([3,4]) = %v want sqrt(12.5)", got)
	}
	if got := RMS(nil); got != 0 {
		t.Fatalf("RMS(nil) = %v want 0", got)
	}
}
