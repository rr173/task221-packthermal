// Package inverse 实现热阻/热容参数反演与接触异常检测。
//
// 反演核心：对内壁参考传感器记录的环境阶跃升温曲线，先用对数线性回归得到
// 一阶 RC 时间常数 τ 的初值，再用高斯-牛顿迭代精化；随后结合层序热容或稳态
// 热阻先验，把 τ 分离为等效热阻 R 与等效热容 C。最后对每只传感器计算拟合
// 残差均方根，用中位数绝对偏差（MAD）识别接触不良的离群传感器。
package inverse

import (
	"math"
	"sort"

	"task221-packthermal/internal/model"
)

const (
	maxIterations = 200
	tauTolerance  = 1e-6
)

// FitInput 是一次反演的输入。
type FitInput struct {
	InitTemp float64            // 阶跃前箱内温度
	EnvTemp  float64            // 阶跃后环境温度（稳态）
	Samples  []model.TempSample // 升温曲线采样点
	CapPrior float64            // 热容先验（J/K），>0 时用于分离 R=τ/C
	ResPrior float64            // 热阻先验（K/W），CapPrior<=0 且 ResPrior>0 时 C=τ/R
}

// FitResult 是一次反演的输出。
type FitResult struct {
	Tau       float64 // 时间常数 s
	REff      float64 // 等效热阻 K/W
	CEff      float64 // 等效热容 J/K
	Rms       float64 // 拟合残差均方根 K
	Converged bool
	Iterations int
}

// FitRC 执行时间常数反演并分离 R/C 参数。
func FitRC(in FitInput) FitResult {
	tau, converged, iters := fitTau(in)
	res := FitResult{Tau: tau, Converged: converged, Iterations: iters}
	res.Rms = RMS(Residuals(in, tau))
	switch {
	case in.CapPrior > 0:
		res.CEff = in.CapPrior
		res.REff = tau / in.CapPrior
	case in.ResPrior > 0:
		res.REff = in.ResPrior
		res.CEff = tau / in.ResPrior
	}
	return res
}

// fitTau 先对数线性回归得到 τ 初值，再高斯-牛顿精化。
func fitTau(in FitInput) (float64, bool, int) {
	if in.EnvTemp == in.InitTemp {
		return 0, false, 0
	}
	tau := logLinearTau(in)
	if tau <= 0 || math.IsInf(tau, 0) || math.IsNaN(tau) {
		return 0, false, 0
	}
	// 高斯-牛顿迭代：最小化残差平方和。
	for i := 0; i < maxIterations; i++ {
		var num, den float64
		for _, s := range in.Samples {
			if s.Seconds < 0 {
				continue
			}
			e := math.Exp(-s.Seconds / tau)
			// 模型值 T = Env - (Env - Init) * e
			r := s.Temperature - (in.EnvTemp - (in.EnvTemp-in.InitTemp)*e)
			// 雅可比 dT/dτ = -(Env-Init)*t/τ²*e
			j := -(in.EnvTemp - in.InitTemp) * s.Seconds / (tau * tau) * e
			num += r * j
			den += j * j
		}
		if den == 0 {
			return tau, true, i + 1
		}
		step := num / den
		if math.Abs(step) <= tauTolerance*math.Abs(tau) {
			return tau, true, i + 1
		}
		tau += step
		if tau <= 0 {
			return 0, false, i + 1
		}
	}
	return tau, false, maxIterations
}

// logLinearTau 用 ln(Env-T) = ln(Env-Init) - t/τ 的线性回归估计 τ。
func logLinearTau(in FitInput) float64 {
	var sx, sy, sxx, sxy float64
	var n float64
	lnA := math.Log(math.Abs(in.EnvTemp - in.InitTemp))
	for _, s := range in.Samples {
		if s.Seconds < 0 {
			continue
		}
		delta := in.EnvTemp - s.Temperature
		if delta <= 0 {
			continue // 已越过稳态或噪声点，跳过
		}
		y := math.Log(delta) - lnA
		sx += s.Seconds
		sy += y
		sxx += s.Seconds * s.Seconds
		sxy += s.Seconds * y
		n++
	}
	if n < 2 {
		return 0
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return 0
	}
	slope := (n*sxy - sx*sy) / den
	if slope >= 0 {
		return 0 // 无升温衰减趋势
	}
	return -1.0 / slope
}

// predict 返回给定 τ 下每个采样点的模型温度。
func predict(in FitInput, tau float64) []model.TempSample {
	out := make([]model.TempSample, 0, len(in.Samples))
	for _, s := range in.Samples {
		t := in.InitTemp
		if s.Seconds >= 0 && tau > 0 {
			t = in.EnvTemp - (in.EnvTemp-in.InitTemp)*math.Exp(-s.Seconds/tau)
		}
		out = append(out, model.TempSample{Seconds: s.Seconds, Temperature: t})
	}
	return out
}

// Residuals 返回每个采样点的残差（测量 - 模型）。
func Residuals(in FitInput, tau float64) []float64 {
	out := make([]float64, 0, len(in.Samples))
	modeled := predict(in, tau)
	for i, s := range in.Samples {
		out = append(out, s.Temperature-modeled[i].Temperature)
	}
	return out
}

// RMS 计算一组残差的均方根。
func RMS(residuals []float64) float64 {
	if len(residuals) == 0 {
		return 0
	}
	var sum float64
	for _, r := range residuals {
		sum += r * r
	}
	return math.Sqrt(sum / float64(len(residuals)))
}

// DetectAnomalies 基于残差 RMS 的 MAD 离群检测，返回异常传感器 ID。
//
// 正常传感器的升温曲线服从同一一阶模型，残差 RMS 集中在低值；接触不良的
// 传感器残差明显偏大。规则：rms > median + k*MAD 判定为接触异常。
func DetectAnomalies(sensorRMS map[int64]float64, k float64) []int64 {
	if len(sensorRMS) == 0 {
		return nil
	}
	vals := make([]float64, 0, len(sensorRMS))
	for _, v := range sensorRMS {
		vals = append(vals, v)
	}
	sort.Float64s(vals)
	median := vals[len(vals)/2]
	if len(vals)%2 == 0 {
		median = (vals[len(vals)/2-1] + vals[len(vals)/2]) / 2
	}
	devs := make([]float64, len(vals))
	for i, v := range vals {
		devs[i] = math.Abs(v - median)
	}
	sort.Float64s(devs)
	mad := devs[len(devs)/2]
	if len(devs)%2 == 0 {
		mad = (devs[len(devs)/2-1] + devs[len(devs)/2]) / 2
	}
	threshold := median + k*mad
	var out []int64
	for id, v := range sensorRMS {
		if v > threshold {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
