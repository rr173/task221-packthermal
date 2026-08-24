// Package thermal 实现包装箱体的一维热网络模型：稳态热阻、热容与时间常数。
//
// 物理模型采用集总参数近似：箱体由外到内若干层串联，每层贡献一个传导热阻
// R_i = L_i / (k_i · A) 和一个热容 C_i = ρ_i · c_i · A · L_i。串联后得到等效
// 热阻 R_total 与等效热容 C_total，其一阶阶跃响应的时间常数为 τ = R·C。
package thermal

import (
	"errors"
	"math"

	"task221-packthermal/internal/model"
)

// Network 是反演所依赖的集总热网络。
type Network struct {
	Layers  []model.Layer
	RTotal  float64 // 等效热阻 K/W
	CTotal  float64 // 等效热容 J/K
	Tau     float64 // 时间常数 s
}

// BuildNetwork 由层序构造热网络，计算等效热阻、热容与时间常数。
func BuildNetwork(layers []model.Layer) (*Network, error) {
	if len(layers) == 0 {
		return nil, errors.New("thermal: empty layer stack")
	}
	n := &Network{Layers: layers}
	var rTotal, cTotal float64
	for _, l := range layers {
		if l.ThicknessMM <= 0 || l.Conductivity <= 0 || l.AreaM2 <= 0 {
			return nil, errors.New("thermal: layer has non-positive geometry or conductivity")
		}
		if l.Density <= 0 || l.SpecificHeat <= 0 {
			return nil, errors.New("thermal: layer has non-positive heat capacity parameters")
		}
		thickness := l.ThicknessMM / 1000.0 // mm -> m
		rTotal += thickness / (l.Conductivity * l.AreaM2)
		cTotal += l.Density * l.SpecificHeat * thickness * l.AreaM2
	}
	if rTotal <= 0 || cTotal <= 0 {
		return nil, errors.New("thermal: computed non-positive equivalent parameters")
	}
	n.RTotal = rTotal
	n.CTotal = cTotal
	n.Tau = rTotal * cTotal
	return n, nil
}

// LayerResistance 计算单层热阻 K/W。
func LayerResistance(l model.Layer) float64 {
	return (l.ThicknessMM / 1000.0) / (l.Conductivity * l.AreaM2)
}

// LayerCapacitance 计算单层热容 J/K。
func LayerCapacitance(l model.Layer) float64 {
	return l.Density * l.SpecificHeat * (l.ThicknessMM / 1000.0) * l.AreaM2
}

// Predict 返回一阶 RC 模型在 t 时刻对「环境温度从 init 阶跃到 env」的响应温度。
//
//	T(t) = env - (env - init) · exp(-t / τ)
func (n *Network) Predict(t, init, env float64) float64 {
	if n.Tau <= 0 {
		return env
	}
	if t < 0 {
		return init
	}
	return env - (env-init)*math.Exp(-t/n.Tau)
}

// SteadyRise 返回给定热流 Q（W）下箱体的稳态内外温差 = Q · R_total。
func (n *Network) SteadyRise(heatFlowW float64) float64 {
	return heatFlowW * n.RTotal
}
