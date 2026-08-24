package service

import (
	"context"
	"fmt"
	"math"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/thermal"
)

// RunDemo 执行确定性的端到端演示：登记层序与环境 → 布置传感器 → 上传温度
// 时序 → 冻结 → 反演热阻/热容 → 识别接触异常 → 发布快照。用于 --smoke-test。
func (s *Service) RunDemo(ctx context.Context) error {
	// 1. 创建验证试验。
	trial, err := s.CreateTrial(ctx, "TRL-0001", "冷链包装回温验证样例")
	if err != nil {
		return fmt.Errorf("create trial: %w", err)
	}

	// 2. 登记箱体层序（外壁 → 保温芯 → 内壁）。
	layers := []model.Layer{
		{Seq: 1, Material: "HDPE-outer", ThicknessMM: 2.0, Conductivity: 0.45, Density: 950, SpecificHeat: 1900, AreaM2: 0.6},
		{Seq: 2, Material: "PU-foam", ThicknessMM: 40.0, Conductivity: 0.022, Density: 35, SpecificHeat: 1400, AreaM2: 0.6},
		{Seq: 3, Material: "HDPE-inner", ThicknessMM: 2.0, Conductivity: 0.45, Density: 950, SpecificHeat: 1900, AreaM2: 0.6},
	}
	for i := range layers {
		if _, err := s.AddLayer(ctx, trial.ID, &layers[i]); err != nil {
			return fmt.Errorf("add layer: %w", err)
		}
	}
	net, err := thermal.BuildNetwork(layers)
	if err != nil {
		return fmt.Errorf("build network: %w", err)
	}

	// 3. 登记环境阶跃曲线：从 5°C（冷库）阶跃到 25°C（室温）。
	const (
		envInit  = 5.0
		envFinal = 25.0
	)
	envSamples := []model.EnvSample{
		{Seconds: 0, Temperature: envInit},
		{Seconds: 60, Temperature: envFinal},
		{Seconds: 3600, Temperature: envFinal},
	}
	if _, err := s.AddEnv(ctx, trial.ID, "cold-room-to-ambient", envSamples); err != nil {
		return fmt.Errorf("add env: %w", err)
	}

	// 4. 布置传感器：一只内壁参考、一只内壁复检、一只接触不良的界面传感器。
	inner, err := s.AddSensor(ctx, trial.ID, "S-IN", "inner", "celsius")
	if err != nil {
		return fmt.Errorf("add sensor inner: %w", err)
	}
	inner2, err := s.AddSensor(ctx, trial.ID, "S-IN2", "inner", "celsius")
	if err != nil {
		return fmt.Errorf("add sensor inner2: %w", err)
	}
	contact, err := s.AddSensor(ctx, trial.ID, "S-CT", "interface-1", "celsius")
	if err != nil {
		return fmt.Errorf("add sensor contact: %w", err)
	}

	// 5. 上传温度时序（升温曲线，一阶 RC 响应）。
	innerSamples := generateRC(envInit, envFinal, net.Tau, 61, 60)
	inner2Samples := generateRC(envInit, envFinal, net.Tau, 61, 60)
	// 接触不良：传感器与壁面脱开，等效时间常数明显偏小（响应异常快）。
	contactSamples := generateRC(envInit, envFinal, net.Tau*0.12, 61, 60)

	if _, err := s.ImportSeries(ctx, trial.ID, inner.ID, "celsius", innerSamples); err != nil {
		return fmt.Errorf("import inner series: %w", err)
	}
	if _, err := s.ImportSeries(ctx, trial.ID, inner2.ID, "celsius", inner2Samples); err != nil {
		return fmt.Errorf("import inner2 series: %w", err)
	}
	if _, err := s.ImportSeries(ctx, trial.ID, contact.ID, "celsius", contactSamples); err != nil {
		return fmt.Errorf("import contact series: %w", err)
	}
	// 重复上传同一条时序，验证幂等去重。
	if _, err := s.ImportSeries(ctx, trial.ID, inner.ID, "celsius", innerSamples); err == nil {
		return fmt.Errorf("expected duplicate series to be rejected")
	}

	// 6. 开始采集 → 冻结 → 进入待反演。
	if _, err := s.StartCollecting(ctx, trial.ID); err != nil {
		return fmt.Errorf("start collecting: %w", err)
	}
	if _, err := s.FreezeTrial(ctx, trial.ID); err != nil {
		return fmt.Errorf("freeze trial: %w", err)
	}

	// 7. 创建并确认热模型。
	m, err := s.CreateModel(ctx, trial.ID, "nominal-3-layer")
	if err != nil {
		return fmt.Errorf("create model: %w", err)
	}
	if _, err := s.ConfirmModel(ctx, m.ID); err != nil {
		return fmt.Errorf("confirm model: %w", err)
	}

	// 8. 执行反演。
	inv, err := s.RunInversion(ctx, trial.ID, m.ID)
	if err != nil {
		return fmt.Errorf("run inversion: %w", err)
	}
	if !inv.Converged {
		return fmt.Errorf("inversion did not converge: tau=%.4f", inv.Tau)
	}
	if inv.Tau <= 0 || inv.REff <= 0 || inv.CEff <= 0 {
		return fmt.Errorf("inversion produced non-positive parameters: R=%.4f C=%.4f", inv.REff, inv.CEff)
	}

	// 9. 校验接触异常被识别。
	contactSeries, err := s.repos.Series.ListSeriesBySensor(ctx, trial.ID, contact.ID)
	if err != nil {
		return err
	}
	found := false
	for _, srs := range contactSeries {
		if srs.State == model.SeriesContact {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("contact anomaly not detected for sensor %d", contact.ID)
	}

	// 10. 发布快照。
	snap, err := s.PublishSnapshot(ctx, trial.ID)
	if err != nil {
		return fmt.Errorf("publish snapshot: %w", err)
	}
	if snap.Version != 1 || snap.State != model.SnapshotPublished {
		return fmt.Errorf("unexpected snapshot: version=%d state=%s", snap.Version, snap.State)
	}

	// 11. 确认试验。
	if _, err := s.ConfirmTrial(ctx, trial.ID); err != nil {
		return fmt.Errorf("confirm trial: %w", err)
	}

	// 12. 自检完整性。
	check, err := s.SelfCheck(ctx)
	if err != nil {
		return err
	}
	if check["integrity_check"] != 1 {
		return fmt.Errorf("integrity check failed: %+v", check)
	}
	return nil
}

// generateRC 生成一阶 RC 升温响应 T(t) = Env - (Env-Init)·exp(-t/τ)。
func generateRC(initTemp, envTemp, tau float64, n int, step float64) []model.TempSample {
	out := make([]model.TempSample, 0, n)
	for i := 0; i < n; i++ {
		t := float64(i) * step
		v := envTemp - (envTemp-initTemp)*math.Exp(-t/tau)
		out = append(out, model.TempSample{Seconds: t, Temperature: v})
	}
	return out
}
