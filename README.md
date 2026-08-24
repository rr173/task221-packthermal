# task221-packthermal 冷链包装热阻参数反演服务

面向冷链验证工程师的纯后端服务：登记包装箱体的层序、环境阶跃曲线与传感器布置，接收多点温度时序，反演等效热阻与热容参数，分析拟合残差结构，识别接触不良传感器，并发布版本化的不可变验证快照。

## 业务闭环

1. 创建验证试验，登记箱体层序（材料、厚度、导热系数、密度、比热、面积）。
2. 登记环境温度阶跃曲线与传感器位置（内壁 / 界面 / 外壁）。
3. 批量接收各传感器的温度时序（温标校验、时间单调性、缺口检测、内容指纹幂等）。
4. 冻结采集后，用内壁参考序列做一阶 RC 时间常数反演，结合层序热容先验分离热阻 / 热容。
5. 计算各传感器残差均方根，用 MAD 离群检测识别接触不良传感器。
6. 发布版本化验证快照（旧发布版本自动替代），确认并封存试验。

## 标准命令

```bash
# 启动 HTTP 服务
go run ./cmd/packthermal --addr :8080 --db packthermal.db

# 确定性端到端自检（Docker 判据，结束后退出）
go run ./cmd/packthermal --smoke-test

# 构建 / 静态检查 / 测试
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
```

## API 入口（统一 /api 前缀）

- 试验：`POST/GET /api/trials`、`GET /api/trials/{id}`、`POST /api/trials/{id}/freeze|confirm|archive`
- 层序：`POST/GET /api/trials/{id}/layers`
- 环境曲线：`POST/GET /api/trials/{id}/env`
- 传感器：`POST/GET /api/trials/{id}/sensors`
- 温度时序：`POST/GET /api/trials/{id}/tempseries`、`PUT /api/series/{id}/contact|valid`
- 热模型：`POST/GET /api/trials/{id}/models`、`POST /api/models/{id}/confirm`
- 反演：`POST/GET /api/trials/{id}/inversions`、`GET /api/inversions/{id}`
- 快照：`POST/GET /api/trials/{id}/snapshots`、`GET /api/snapshots/{id}`
- 元信息：`GET /healthz`、`GET /api/selfcheck`、`GET /api/stats`、`GET /api/audit/events`、`GET /api/audit/trials/{id}`

## 持久化

SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 无关）。试验、层序、环境曲线、传感器、温度时序、热模型、反演结果、快照与审计台账全部落盘；服务重启后从数据库完整恢复。时序按内容指纹幂等，封存试验只读，快照版本不可变。

## 状态机

- 验证试验：`planned → collecting → pending_inversion → confirmed → archived`
- 温度序列：`pending → valid / contact_anomaly / gap`
- 热模型：`draft → runnable / boundary_conflict → confirmed`
- 验证快照：`computing → under_review → published → superseded`
