package ingest

import (
	"testing"

	"task221-packthermal/internal/model"
)

func TestNormalizeScale(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"celsius", ScaleCelsius},
		{"C", ScaleCelsius},
		{"kelvin", ScaleKelvin},
		{"K", ScaleKelvin},
	} {
		got, err := NormalizeScale(c.in)
		if err != nil || got != c.want {
			t.Fatalf("NormalizeScale(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := NormalizeScale("fahrenheit"); err == nil {
		t.Fatal("expected error for unknown scale")
	}
}

func TestTemperatureConversion(t *testing.T) {
	if got := ToCelsius(273.15, ScaleKelvin); got != 0 {
		t.Fatalf("ToCelsius(273.15 K) = %v", got)
	}
	if got := FromCelsius(0, ScaleKelvin); got != 273.15 {
		t.Fatalf("FromCelsius(0) = %v", got)
	}
}

func TestFingerprintStable(t *testing.T) {
	a := []model.TempSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 10}}
	b := []model.TempSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 10}}
	c := []model.TempSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 10.1}}
	if FingerprintSamples(a) != FingerprintSamples(b) {
		t.Fatal("identical samples should share fingerprint")
	}
	if FingerprintSamples(a) == FingerprintSamples(c) {
		t.Fatal("different samples should differ in fingerprint")
	}
}

func TestValidateMonotonic(t *testing.T) {
	ok := []model.TempSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 10}}
	if err := ValidateMonotonic(ok); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bad := []model.TempSample{{Seconds: 60, Temperature: 5}, {Seconds: 60, Temperature: 10}}
	if err := ValidateMonotonic(bad); err == nil {
		t.Fatal("expected non-monotonic error")
	}
}

func TestDetectGaps(t *testing.T) {
	samples := []model.TempSample{
		{Seconds: 0, Temperature: 5},
		{Seconds: 60, Temperature: 6},
		{Seconds: 600, Temperature: 12}, // 大间隔
		{Seconds: 660, Temperature: 13},
	}
	gaps := DetectGaps(samples, 360)
	if len(gaps) != 1 || gaps[0].FromSeconds != 60 || gaps[0].ToSeconds != 600 {
		t.Fatalf("unexpected gaps: %+v", gaps)
	}
}
