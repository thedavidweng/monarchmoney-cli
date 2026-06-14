# monarchmoney-cli 领域术语表

## 核心守则

**monarchmoney-cli 是 Monarch Money 网页版的替代品**，额外提供自动化便利和 agent 友好。

## 核心概念

**monarchmoney-cli** — Monarch Money 的本地 CLI 工具，用于查询、管理和自动化个人财务数据。安装后的二进制文件是 `monarch`。

## 用户

**个人理财用户** — 需要从终端管理 Monarch Money 账户的人。

**Agent** — 自动化代理。需要确定性行为。

## 命令设计决策

**安全门控** — `--read-only`、`--dry-run`、`--confirm` 三级安全机制。

**审计日志** — 每个远程写操作都记录到 JSONL 文件。

**缓存** — 本地 SQLite 缓存，用于加速查询。
