# BenxinAdminPro（本心通用管理后台）

[![CI](https://github.com/BenxinAdminPro/BenxinAdminPro/actions/workflows/ci.yml/badge.svg)](https://github.com/BenxinAdminPro/BenxinAdminPro/actions/workflows/ci.yml)

开箱即用的完整后台系统，同时是纯 Go 的可复用底座（脚手架 + 模块库）。

- `server/` — Go / Gin 后端底座（认证、RBAC、系统管理、配置中心、加密中间件、统一响应、限流、监控、代码生成、消息中心、驱动接口）。
- `admin/` — Vue3 + Element Plus 管理前端。

## CI

每次向 `main` 推送或提 PR 时，GitHub Actions 自动执行：`go build` → `go vet`（默认 + integration tag）→ `go test`（默认闸门）→ `go test -tags=integration`（连 MySQL/Redis service container）。
集成测试连接参数经 `BENXIN_TEST_MYSQL_DSN` / `BENXIN_TEST_REDIS_ADDR` 注入。
