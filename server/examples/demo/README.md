# BenxinAdminPro Demo

> 最小可运行后台：装配 T-001~T-005 五大块，验证全链路。

## 快速开始

### 1. 起依赖
```bash
docker compose -f deploy/docker-compose.dev.yml up -d
```

### 2. 配置
```bash
cd server/examples/demo
cp config.example.yaml config.local.yaml
# 编辑 config.local.yaml，填入密钥（openssl rand -base64 32）
```

### 3. 启动
```bash
cd server
go run ./examples/demo
```

迁移执行器自动建表 → 种子数据自动创建 → HTTP 服务启动于 :8080。

## 种子账号

| 用户名 | 密码 | 角色 | 数据范围 |
|--------|------|------|----------|
| admin | (配置的 seed.admin_password) | super_admin | 全部 |
| editor | editor123 | editor | 仅本人 |
| dept_mgr | manager123 | dept_mgr | 本部门 |
| biz_user | bizuser123 | (无角色) | — |

## 端到端验证 (curl)

### 1. 获取验证码
```bash
curl -s http://localhost:8080/auth/captcha -X POST | jq .
```

### 2. 登录
```bash
TOKEN=$(curl -s http://localhost:8080/auth/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"YOUR_ADMIN_PWD"}' | jq -r '.data.access_token')
echo $TOKEN
```

### 3. 带令牌调受保护接口
```bash
# 用户列表（超管看全部）
curl -s http://localhost:8080/sys/users \
  -H "Authorization: Bearer $TOKEN" | jq .

# 无令牌 → 401
curl -s http://localhost:8080/sys/users | jq .
```

### 4. 数据权限演示
```bash
# editor 登录（data_scope=Self，只看自己）
EDITOR_TOKEN=$(curl -s http://localhost:8080/auth/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"editor","password":"editor123"}' | jq -r '.data.access_token')

curl -s http://localhost:8080/sys/users \
  -H "Authorization: Bearer $EDITOR_TOKEN" | jq '.data.total'
# 应返回 1（仅本人）
```

### 5. 权限码验证
```bash
# editor 无 sys:user:create 权限 → 403
curl -s http://localhost:8080/sys/users -X POST \
  -H "Authorization: Bearer $EDITOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}' | jq .
```

### 6. 刷新 + 登出
```bash
REFRESH=$(curl -s http://localhost:8080/auth/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"YOUR_ADMIN_PWD"}' | jq -r '.data.refresh_token')

# 刷新
curl -s http://localhost:8080/auth/refresh -X POST \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH\"}" | jq .

# 登出
curl -s http://localhost:8080/auth/logout -X POST \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 7. 字典 / 参数 / 日志
```bash
# 字典
curl -s http://localhost:8080/sys/dict/types \
  -H "Authorization: Bearer $TOKEN" | jq .

# 操作日志
curl -s http://localhost:8080/sys/logs/oper \
  -H "Authorization: Bearer $TOKEN" | jq .

# 登录日志
curl -s http://localhost:8080/sys/logs/login \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## daxing 真人验收清单（T-001~T-005 汇总，照做清账）

- [ ] **T-002**: captcha→login→带 access 调 /sys/users 200→refresh→logout 后旧 access 被拒
- [ ] **T-002 锁定**: 连续错密码触发验证码→再触发锁定
- [ ] **T-003a**: 用户/部门/岗位 CRUD + 部门树；GormUserProvider 登录
- [ ] **T-003b**: 建角色→配菜单→用户配角色→有权通过/无权 403；超管全放行；/sys/auth/menus+perms；对外 ID hashid
- [ ] **T-003c**: 不同 data_scope 角色数据范围；无角色/无 dept 收紧
- [ ] **T-004a**: 字典/参数 CRUD；操作/登录日志无敏感信息
- [ ] **T-004b**: 文件上传/下载/删除；超限被拒；穿越被拒；根目录未裸暴露
- [ ] **T-004c**: 错误响应格式统一
- [ ] **T-005**: 迁移一键建表；加密参数 DB 密文 + 列表脱敏；热加载；CBC 信封正常
- [ ] 全程：头注释五项、日志无敏感信息、无真实密钥入库
