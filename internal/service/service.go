// Package service 编排冷链包装热阻反演的完整业务闭环。
package service

import (
	"context"
	"errors"
	"fmt"

	"task221-packthermal/internal/ingest"
	"task221-packthermal/internal/inverse"
	"task221-packthermal/internal/model"
	"task221-packthermal/internal/store"
	"task221-packthermal/internal/thermal"
)

// Service 是应用层编排入口。
type Service struct {
	repos *store.Repositories
}

// New 构造服务实例。
func New(repos *store.Repositories) *Service {
	return &Service{repos: repos}
}

// ensureTrialState 校验试验处于允许的状态之一，否则返回状态冲突错误。
func (s *Service) ensureTrialState(ctx context.Context, id int64, allowed ...string) (*model.Trial, error) {
	t, err := s.repos.Trial.GetTrial(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, a := range allowed {
		if t.State == a {
			return t, nil
		}
	}
	return nil, &model.StateError{Entity: "trial", ID: id, From: t.State, To: allowed[0]}
}

// rejectArchived 拒绝封存试验的写操作。
func (s *Service) rejectArchived(ctx context.Context, id int64) error {
	t, err := s.repos.Trial.GetTrial(ctx, id)
	if err != nil {
		return err
	}
	if t.State == model.TrialArchived {
		return model.ErrImmutable
	}
	return nil
}

// ---- 试验 ----

// CreateTrial 创建验证试验。
func (s *Service) CreateTrial(ctx context.Context, code, title string) (*model.Trial, error) {
	if code == "" {
		return nil, fmt.Errorf("%w: code is required", model.ErrInvalidInput)
	}
	t, err := s.repos.Trial.CreateTrial(ctx, code, title)
	if err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, t.ID, "trial.create", code)
	return t, nil
}

// ListTrials 列出全部试验。
func (s *Service) ListTrials(ctx context.Context) ([]*model.Trial, error) {
	return s.repos.Trial.ListTrials(ctx)
}

// GetTrial 读取试验。
func (s *Service) GetTrial(ctx context.Context, id int64) (*model.Trial, error) {
	return s.repos.Trial.GetTrial(ctx, id)
}

// StartCollecting 开始采集（planned → collecting）。
func (s *Service) StartCollecting(ctx context.Context, id int64) (*model.Trial, error) {
	if err := s.repos.Trial.TransitionState(ctx, id, model.TrialPlanned, model.TrialCollecting); err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, id, "trial.start", "planned -> collecting")
	return s.repos.Trial.GetTrial(ctx, id)
}

// FreezeTrial 冻结采集，进入待反演。
func (s *Service) FreezeTrial(ctx context.Context, id int64) (*model.Trial, error) {
	if err := s.repos.Trial.TransitionState(ctx, id, model.TrialCollecting, model.TrialPendingInversion); err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, id, "trial.freeze", "collecting -> pending_inversion")
	return s.repos.Trial.GetTrial(ctx, id)
}

// ConfirmTrial 确认试验结果。
func (s *Service) ConfirmTrial(ctx context.Context, id int64) (*model.Trial, error) {
	if err := s.repos.Trial.TransitionState(ctx, id, model.TrialPendingInversion, model.TrialConfirmed); err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, id, "trial.confirm", "pending_inversion -> confirmed")
	return s.repos.Trial.GetTrial(ctx, id)
}

// ArchiveTrial 封存试验（只读）。
func (s *Service) ArchiveTrial(ctx context.Context, id int64) (*model.Trial, error) {
	if err := s.repos.Trial.TransitionState(ctx, id, model.TrialConfirmed, model.TrialArchived); err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, id, "trial.archive", "confirmed -> archived")
	return s.repos.Trial.GetTrial(ctx, id)
}

// ---- 层序 ----

// AddLayer 追加一层层序。
func (s *Service) AddLayer(ctx context.Context, trialID int64, l *model.Layer) (*model.Layer, error) {
	if err := s.rejectArchived(ctx, trialID); err != nil {
		return nil, err
	}
	l.TrialID = trialID
	created, err := s.repos.Layer.CreateLayer(ctx, l)
	if err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, trialID, "layer.add", l.Material)
	return created, nil
}

// ListLayers 列出层序。
func (s *Service) ListLayers(ctx context.Context, trialID int64) ([]*model.Layer, error) {
	return s.repos.Layer.ListLayers(ctx, trialID)
}

// ---- 环境曲线 ----

// AddEnv 保存环境阶跃曲线（校验单调 + 指纹幂等）。
func (s *Service) AddEnv(ctx context.Context, trialID int64, name string, samples []model.EnvSample) (*model.EnvProfile, error) {
	if err := s.rejectArchived(ctx, trialID); err != nil {
		return nil, err
	}
	if err := ingest.ValidateEnvMonotonic(samples); err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrInvalidInput, err)
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("%w: env samples empty", model.ErrInvalidInput)
	}
	e := &model.EnvProfile{TrialID: trialID, Name: name, Fingerprint: ingest.FingerprintEnv(samples), Samples: samples}
	created, err := s.repos.Env.CreateEnv(ctx, e)
	if err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, trialID, "env.add", name)
	return created, nil
}

// ListEnvs 列出环境曲线。
func (s *Service) ListEnvs(ctx context.Context, trialID int64) ([]*model.EnvProfile, error) {
	return s.repos.Env.ListEnvs(ctx, trialID)
}

// ---- 传感器 ----

// AddSensor 登记传感器（校验温标）。
func (s *Service) AddSensor(ctx context.Context, trialID int64, code, position, scale string) (*model.Sensor, error) {
	if err := s.rejectArchived(ctx, trialID); err != nil {
		return nil, err
	}
	norm, err := ingest.NormalizeScale(scale)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrInvalidInput, err)
	}
	if position == "" {
		return nil, fmt.Errorf("%w: position required", model.ErrInvalidInput)
	}
	sn := &model.Sensor{TrialID: trialID, Code: code, Position: position, Scale: norm}
	created, err := s.repos.Sensor.CreateSensor(ctx, sn)
	if err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, trialID, "sensor.add", code)
	return created, nil
}

// ListSensors 列出传感器。
func (s *Service) ListSensors(ctx context.Context, trialID int64) ([]*model.Sensor, error) {
	return s.repos.Sensor.ListSensors(ctx, trialID)
}

// ---- 温度时序 ----

// ImportSeries 接收温度时序：校验温标、时间单调性、缺口，并按内容指纹幂等。
func (s *Service) ImportSeries(ctx context.Context, trialID, sensorID int64, scale string, samples []model.TempSample) (*model.TempSeries, error) {
	if err := s.rejectArchived(ctx, trialID); err != nil {
		return nil, err
	}
	norm, err := ingest.NormalizeScale(scale)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrInvalidInput, err)
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("%w: samples empty", model.ErrInvalidInput)
	}
	if err := ingest.ValidateMonotonic(samples); err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrInvalidInput, err)
	}
	// 统一到摄氏度后再落库，避免温标混用污染反演。
	normed := make([]model.TempSample, len(samples))
	for i, sm := range samples {
		normed[i] = model.TempSample{Seconds: sm.Seconds, Temperature: ingest.ToCelsius(sm.Temperature, norm)}
	}
	fp := ingest.FingerprintSamples(normed)
	state := model.SeriesPending
	// 缺口检测：存在明显缺口则标记为 gap 状态。
	if len(ingest.DetectGaps(normed, maxGapSeconds(normed))) > 0 {
		state = model.SeriesGap
	}
	srs := &model.TempSeries{TrialID: trialID, SensorID: sensorID, Fingerprint: fp, Scale: ingest.ScaleCelsius, State: state}
	created, err := s.repos.Series.CreateSeries(ctx, srs, normed)
	if err != nil {
		return nil, fmt.Errorf("%w: series storage failed", model.ErrInvalidInput)
	}
	_ = s.repos.Audit.Record(ctx, trialID, "series.import", fmt.Sprintf("sensor=%d samples=%d", sensorID, len(samples)))
	return created, nil
}

// maxGapSeconds 基于采样跨度估算的缺口阈值。
func maxGapSeconds(samples []model.TempSample) float64 {
	if len(samples) < 2 {
		return 0
	}
	span := samples[len(samples)-1].Seconds - samples[0].Seconds
	// 平均间隔的 6 倍以上视为缺口。
	avg := span / float64(len(samples)-1)
	return avg * 6
}

// ListSeries 列出温度时序。
func (s *Service) ListSeries(ctx context.Context, trialID int64) ([]*model.TempSeries, error) {
	return s.repos.Series.ListSeries(ctx, trialID)
}

// MarkSeriesContact 把时序标记为接触异常。
func (s *Service) MarkSeriesContact(ctx context.Context, seriesID int64) (*model.TempSeries, error) {
	srs, err := s.repos.Series.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Series.UpdateSeriesState(ctx, seriesID, srs.State, model.SeriesContact); err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, srs.TrialID, "series.contact", fmt.Sprintf("series=%d", seriesID))
	return s.repos.Series.GetSeries(ctx, seriesID)
}

// MarkSeriesValid 把时序标记为有效。
func (s *Service) MarkSeriesValid(ctx context.Context, seriesID int64) (*model.TempSeries, error) {
	srs, err := s.repos.Series.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Series.UpdateSeriesState(ctx, seriesID, srs.State, model.SeriesValid); err != nil {
		return nil, err
	}
	return s.repos.Series.GetSeries(ctx, seriesID)
}

// ---- 热模型 ----

// CreateModel 新建热网络模型。
func (s *Service) CreateModel(ctx context.Context, trialID int64, name string) (*model.ThermalModel, error) {
	if err := s.rejectArchived(ctx, trialID); err != nil {
		return nil, err
	}
	m := &model.ThermalModel{TrialID: trialID, Name: name, State: model.ModelDraft}
	created, err := s.repos.Model.CreateModel(ctx, m)
	if err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, trialID, "model.create", name)
	return created, nil
}

// ListModels 列出热模型。
func (s *Service) ListModels(ctx context.Context, trialID int64) ([]*model.ThermalModel, error) {
	return s.repos.Model.ListModels(ctx, trialID)
}

// ConfirmModel 确认模型。
func (s *Service) ConfirmModel(ctx context.Context, modelID int64) (*model.ThermalModel, error) {
	m, err := s.repos.Model.GetModel(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Model.UpdateModelState(ctx, modelID, m.State, model.ModelConfirmed); err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, m.TrialID, "model.confirm", m.Name)
	return s.repos.Model.GetModel(ctx, modelID)
}

// ---- 反演 ----

// RunInversion 执行热阻/热容反演，并识别接触异常传感器。
func (s *Service) RunInversion(ctx context.Context, trialID, modelID int64) (*model.InversionResult, error) {
	if _, err := s.ensureTrialState(ctx, trialID, model.TrialPendingInversion, model.TrialConfirmed); err != nil {
		return nil, err
	}
	layers, err := s.repos.Layer.ListLayers(ctx, trialID)
	if err != nil {
		return nil, err
	}
	layerValues := make([]model.Layer, 0, len(layers))
	for _, l := range layers {
		layerValues = append(layerValues, *l)
	}
	net, err := thermal.BuildNetwork(layerValues)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrInvalidInput, err)
	}
	envs, err := s.repos.Env.ListEnvs(ctx, trialID)
	if err != nil {
		return nil, err
	}
	if len(envs) == 0 {
		return nil, fmt.Errorf("%w: no env profile", model.ErrInvalidInput)
	}
	env := envs[len(envs)-1]
	envInit := env.Samples[0].Temperature
	envFinal := env.Samples[len(env.Samples)-1].Temperature

	// 选择内壁参考序列：优先 valid 状态，否则 pending。
	ref, refSamples, err := s.referenceSeries(ctx, trialID)
	if err != nil {
		return nil, err
	}
	fit := inverse.FitRC(inverse.FitInput{
		InitTemp: envInit,
		EnvTemp:  envFinal,
		Samples:  refSamples,
		CapPrior: net.CTotal,
	})
	result := &model.InversionResult{
		TrialID:     trialID,
		ModelID:     modelID,
		SeriesID:    ref.ID,
		REff:        fit.REff,
		CEff:        fit.CEff,
		Tau:         fit.Tau,
		RmsResidual: fit.Rms,
		Converged:   fit.Converged,
		Iterations:  fit.Iterations,
	}

	// 校验通过的时序统一标记为有效，再进行接触异常识别。
	if err := s.markPendingSeriesValid(ctx, trialID); err != nil {
		return nil, err
	}
	if err := s.detectContact(ctx, trialID, envInit, envFinal, fit.Tau); err != nil {
		return nil, err
	}

	created, err := s.repos.Inversion.CreateInversion(ctx, result)
	if err != nil {
		return nil, err
	}
	_ = s.repos.Audit.Record(ctx, trialID, "inversion.run",
		fmt.Sprintf("model=%d R=%.4f C=%.4f tau=%.4f converged=%v", modelID, fit.REff, fit.CEff, fit.Tau, fit.Converged))
	return created, nil
}

// referenceSeries 返回内壁参考传感器的最新可用时序及其样本。
func (s *Service) referenceSeries(ctx context.Context, trialID int64) (*model.TempSeries, []model.TempSample, error) {
	sensors, err := s.repos.Sensor.ListSensors(ctx, trialID)
	if err != nil {
		return nil, nil, err
	}
	for _, sn := range sensors {
		if sn.Position != "inner" {
			continue
		}
		series, err := s.repos.Series.ListSeriesBySensor(ctx, trialID, sn.ID)
		if err != nil {
			return nil, nil, err
		}
		for i := len(series) - 1; i >= 0; i-- {
			srs := series[i]
			if srs.State == model.SeriesGap {
				continue
			}
			samples, err := s.repos.Series.LoadSamples(ctx, srs.ID)
			if err != nil {
				return nil, nil, err
			}
			return srs, samples, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: no inner reference series", model.ErrInvalidInput)
}

// markPendingSeriesValid 把待校验时序批量标记为有效。
func (s *Service) markPendingSeriesValid(ctx context.Context, trialID int64) error {
	series, err := s.repos.Series.ListSeries(ctx, trialID)
	if err != nil {
		return err
	}
	for _, srs := range series {
		if srs.State == model.SeriesPending {
			if err := s.repos.Series.UpdateSeriesState(ctx, srs.ID, model.SeriesPending, model.SeriesValid); err != nil {
				return err
			}
		}
	}
	return nil
}

// detectContact 对每只传感器计算残差 RMS，标记接触异常。
func (s *Service) detectContact(ctx context.Context, trialID int64, initTemp, envTemp, tau float64) error {
	sensors, err := s.repos.Sensor.ListSensors(ctx, trialID)
	if err != nil {
		return err
	}
	rmsBySensor := map[int64]float64{}
	seriesBySensor := map[int64]int64{} // sensorID -> seriesID
	for _, sn := range sensors {
		series, err := s.repos.Series.ListSeriesBySensor(ctx, trialID, sn.ID)
		if err != nil {
			return err
		}
		if len(series) == 0 {
			continue
		}
		// 取最新非 gap 序列。
		for i := len(series) - 1; i >= 0; i-- {
			srs := series[i]
			if srs.State == model.SeriesGap {
				continue
			}
			samples, err := s.repos.Series.LoadSamples(ctx, srs.ID)
			if err != nil {
				return err
			}
			res := inverse.Residuals(inverse.FitInput{InitTemp: initTemp, EnvTemp: envTemp, Samples: samples}, tau)
			rmsBySensor[sn.ID] = inverse.RMS(res)
			seriesBySensor[sn.ID] = srs.ID
			break
		}
	}
	if len(rmsBySensor) < 2 {
		return nil // 传感器不足，无法可靠离群判定。
	}
	for _, anomalyID := range inverse.DetectAnomalies(rmsBySensor, 3.0) {
		if sid, ok := seriesBySensor[anomalyID]; ok {
			_ = s.repos.Series.UpdateSeriesState(ctx, sid, model.SeriesValid, model.SeriesContact)
		}
	}
	return nil
}

// ListInversions 列出反演结果。
func (s *Service) ListInversions(ctx context.Context, trialID int64) ([]*model.InversionResult, error) {
	return s.repos.Inversion.ListInversions(ctx, trialID)
}

// GetInversion 读取单条反演结果。
func (s *Service) GetInversion(ctx context.Context, id int64) (*model.InversionResult, error) {
	return s.repos.Inversion.GetInversion(ctx, id)
}

// ---- 快照 ----

// PublishSnapshot 发布版本化验证快照，替代旧发布版本。
func (s *Service) PublishSnapshot(ctx context.Context, trialID int64) (*model.Snapshot, error) {
	if _, err := s.ensureTrialState(ctx, trialID, model.TrialPendingInversion, model.TrialConfirmed); err != nil {
		return nil, err
	}
	invs, err := s.repos.Inversion.ListInversions(ctx, trialID)
	if err != nil {
		return nil, err
	}
	if len(invs) == 0 {
		return nil, fmt.Errorf("%w: no inversion result to snapshot", model.ErrInvalidInput)
	}
	last := invs[len(invs)-1]
	version, err := s.repos.Snapshot.NextVersion(ctx, trialID)
	if err != nil {
		return nil, err
	}
	sn := &model.Snapshot{
		TrialID: trialID,
		Version: version,
		State:   model.SnapshotComputing,
		Summary: fmt.Sprintf("R=%.4f K/W C=%.4f J/K tau=%.4f s", last.REff, last.CEff, last.Tau),
	}
	created, err := s.repos.Snapshot.CreateSnapshot(ctx, sn)
	if err != nil {
		return nil, err
	}
	// 发布前替代旧发布版本。
	_, _ = s.repos.Snapshot.SupersedePublished(ctx, trialID)
	_ = s.repos.Snapshot.UpdateSnapshotState(ctx, created.ID, model.SnapshotComputing, model.SnapshotPublished)
	_ = s.repos.Audit.Record(ctx, trialID, "snapshot.publish", fmt.Sprintf("version=%d", version))
	return s.repos.Snapshot.GetSnapshot(ctx, created.ID)
}

// ListSnapshots 列出快照。
func (s *Service) ListSnapshots(ctx context.Context, trialID int64) ([]*model.Snapshot, error) {
	return s.repos.Snapshot.ListSnapshots(ctx, trialID)
}

// GetSnapshot 读取单条快照。
func (s *Service) GetSnapshot(ctx context.Context, id int64) (*model.Snapshot, error) {
	return s.repos.Snapshot.GetSnapshot(ctx, id)
}

// ---- 自检与统计 ----

// SelfCheck 返回完整性自检结果。
func (s *Service) SelfCheck(ctx context.Context) (map[string]int, error) {
	out := map[string]int{"integrity_check": 0}
	trials, err := s.repos.Trial.ListTrials(ctx)
	if err != nil {
		return out, err
	}
	for _, t := range trials {
		if t.Code == "" {
			return out, errors.New("selfcheck: empty trial code")
		}
	}
	out["integrity_check"] = 1
	out["trials"] = len(trials)
	return out, nil
}

// Stats 返回统计信息。
func (s *Service) Stats(ctx context.Context) (map[string]int, error) {
	out := map[string]int{}
	trials, err := s.repos.Trial.ListTrials(ctx)
	if err != nil {
		return out, err
	}
	out["trials"] = len(trials)
	out["snapshots"] = 0
	out["inversions"] = 0
	out["series"] = 0
	for _, t := range trials {
		snaps, _ := s.repos.Snapshot.ListSnapshots(ctx, t.ID)
		invs, _ := s.repos.Inversion.ListInversions(ctx, t.ID)
		series, _ := s.repos.Series.ListSeries(ctx, t.ID)
		out["snapshots"] += len(snaps)
		out["inversions"] += len(invs)
		out["series"] += len(series)
	}
	return out, nil
}

// ListAudit 列出试验审计事件。
func (s *Service) ListAudit(ctx context.Context, trialID int64) ([]*model.AuditEvent, error) {
	return s.repos.Audit.List(ctx, trialID)
}

// ListAuditAll 列出全部审计事件。
func (s *Service) ListAuditAll(ctx context.Context, limit int) ([]*model.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repos.Audit.ListAll(ctx, limit)
}
