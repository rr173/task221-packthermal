// Package ingest 负责温度/环境数据的接收校验：温标归一、时间单调性检查、
// 缺口检测与内容指纹计算。
package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"task221-packthermal/internal/model"
)

// 温标常量。
const (
	ScaleCelsius = "celsius"
	ScaleKelvin  = "kelvin"
)

// ErrTimeNotMonotonic 表示采样时间未严格递增。
var ErrTimeNotMonotonic = errors.New("ingest: sample times must be strictly increasing")

// NormalizeScale 校验并规范化温标字符串，非法返回错误。
func NormalizeScale(scale string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(scale)) {
	case ScaleCelsius, "c", "degc":
		return ScaleCelsius, nil
	case ScaleKelvin, "k":
		return ScaleKelvin, nil
	default:
		return "", fmt.Errorf("ingest: unknown temperature scale %q", scale)
	}
}

// ToCelsius 把温度统一换算为摄氏度。
func ToCelsius(t float64, scale string) float64 {
	if scale == ScaleKelvin {
		return t
	}
	return t
}

// FromCelsius 把摄氏度换算为指定温标。
func FromCelsius(t float64, scale string) float64 {
	if scale == ScaleKelvin {
		return t + 273.15
	}
	return t
}

// FingerprintSamples 计算温度时序的内容指纹（幂等去重）。
func FingerprintSamples(samples []model.TempSample) string {
	var sb strings.Builder
	for _, s := range samples {
		fmt.Fprintf(&sb, "%.4f=%.6f;", s.Seconds, s.Temperature)
	}
	return sha256hex(sb.String())
}

// FingerprintEnv 计算环境曲线的内容指纹。
func FingerprintEnv(samples []model.EnvSample) string {
	var sb strings.Builder
	for _, s := range samples {
		fmt.Fprintf(&sb, "%.4f=%.6f;", s.Seconds, s.Temperature)
	}
	return sha256hex(sb.String())
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ValidateMonotonic 校验采样点时间严格递增。
func ValidateMonotonic(samples []model.TempSample) error {
	prev := -1.0
	for _, s := range samples {
		if s.Seconds <= prev {
			return ErrTimeNotMonotonic
		}
		prev = s.Seconds
	}
	return nil
}

// Gap 表示时序中的一段缺口。
type Gap struct {
	FromSeconds float64
	ToSeconds   float64
	Duration    float64
}

// DetectGaps 找出时间间隔超过 maxGap 的缺口区间。
func DetectGaps(samples []model.TempSample, maxGap float64) []Gap {
	var gaps []Gap
	for i := 1; i < len(samples); i++ {
		d := samples[i].Seconds - samples[i-1].Seconds
		if d > maxGap {
			gaps = append(gaps, Gap{
				FromSeconds: samples[i-1].Seconds,
				ToSeconds:   samples[i].Seconds,
				Duration:    d,
			})
		}
	}
	return gaps
}

// ValidateEnvMonotonic 校验环境曲线时间单调递增。
func ValidateEnvMonotonic(samples []model.EnvSample) error {
	prev := -1.0
	for _, s := range samples {
		if s.Seconds <= prev {
			return errors.New("ingest: env sample times must be strictly increasing")
		}
		prev = s.Seconds
	}
	return nil
}
