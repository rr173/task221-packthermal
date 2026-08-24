基于 Go 实现的冷链包装热阻参数反演服务，一款纯后端工程分析服务，处理温度时序、热阻热容反演与验证快照发布。

# task221-packthermal 评测说明

## 启动与自检

```bash
# 构建（本地）
CGO_ENABLED=0 GOTOOLCHAIN=local go build -o packthermal ./cmd/packthermal

# 端到端自检（退出码 0 为通过）
./packthermal --smoke-test
```

`--smoke-test` 从空库开始执行确定性端到端演示：建试验 → 登记三层箱体层序与环境阶跃 → 布置三只传感器 → 上传温度时序（含幂等重复）→ 冻结 → 建模型并确认 → 反演热阻/热容 → 识别接触异常传感器 → 发布版本 1 快照 → 确认试验 → 关闭并重开同一数据库验证持久化恢复，全部断言通过后以 0 退出。

## HTTP 服务

```bash
./packthermal --addr :8080 --db packthermal.db
```

- 健康检查：`GET /healthz`
- 自检：`GET /api/selfcheck`
- 统计：`GET /api/stats`

## Docker 双架构

镜像同时支持 `linux/amd64` 与 `linux/arm64`：

```bash
docker build --platform linux/amd64 -t packthermal:amd64 .
docker run packthermal:amd64    # 默认 CMD 为 --smoke-test
```

## 组件版本

| 组件 | 版本 |
| --- | --- |
| Go | 1.26.3 |
| SQLite | 3.46.1 |
