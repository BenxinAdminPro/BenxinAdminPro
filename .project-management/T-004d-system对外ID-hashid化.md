# 任务书：T-004d system 对外 ID hashid 化（全套）+ 前置抽 idcodec 中立包

**任务编号**：T-004d
**切片名称**：system 实体对外 ID 全链路 hashid 化（dict/config/file/日志）+ 前置抽 `idcodec` 中立包保 system↔rbac 解耦
**目标后端**：Gin（纯 Go 底座）
**契约版本**：openapi v0.9.0（当前）→ 本片升 v0.10.0
**前置依赖**：T-003b（hashid 出参闭环 + ResponseEncoder 模式）、T-003e（rbac 入参收口 + decode helper 范式）、T-004a/b（system 实体与 handler）、T-003d（RegisterRoutes 注入 PermGuard 解耦）

---

### 1. 目标（一句话）
把 system 包（dict/config/file/操作日志/登录日志）所有对外 ID 从裸 `uint64` 全链路收口为 hashid（出参 + path `:id` + openapi），并**先把 hasher 抽到中立 `idcodec` 包**供 rbac/system 共用，**不引入 system→rbac 方向性依赖**——实现"全链路 grep 零裸 uint64 主键"这条明线。

### 2. 范围
- ✅ 包含：
  - **【前置·重构】抽 `idcodec` 中立包**：把 hasher（现 `rbac/hashid.go` 的 `Hasher`/`NewHasher`）搬到中立位置（建议 `server/idcodec`），rbac 改为引用 `idcodec`。对 rbac **纯搬迁、对外零行为变化**（按 T-004c 重构纪律：不改码值/方法签名/出入参行为）。
  - **system 出参 hashid 化（全套）**：SysDictType / SysDictData / SysConfig / SysFile / SysOperLog / SysLoginLog 的 `id`（及任何对外 ID 字段）经 encoder 编码输出，不再裸吐 `uint64`。沿用 T-003b 的 ResponseEncoder 模式，system 由装配注入 `idcodec` hasher。
  - **path `:id` 收口**：三处 `strconv.ParseUint`（`handler.go:197` pid() 服务 dict/types、dict/data、configs 的 PUT/DELETE；`handler_file.go:80`/`:116` files download/delete）改为 hashid 解码；非法/伪造/越界 → **400（复用既有 ErrInvalidID，无新增段）**，与 rbac/T-003e 同一套语义与防探测策略。
  - **openapi v0.10.0 对齐**：修两处已知不一致（`/sys/files/{id}` path 已标 string 但实现 ParseUint；dict/config 的 `:id` PUT/DELETE path 参数 spec 缺失补全）；system 实体出参 schema `id` integer→string(hashid)；info.description 升版 + PHP parity 注明 system 对外 ID 亦为 hashid 字符串。redocly 0 error。
  - **【顺手】修 demo 装配 typed-nil 自检盲区**：本片 system 新引 hasher 进装配，正是改自检写法的时机——`map[string]any{...}==nil` 抓不住 typed-nil（如 `(*Hasher)(nil)`）；改为能真正侦测 typed-nil 的写法（如反射 / 逐项类型断言 + nil 判定），覆盖 hasher 及该 map 全部项。**真正兜底仍是构造器契约（NewHasher 永不返回 (nil,nil)）+ err fail-fast**，自检为第二道。
- ⛔ 不包含：
  - 前端任何改动（sys 页在 T-007d/e/f）。
  - system body 入参收口——**据现状清单确认 system body 无对外 ID 入参**（dict/config 实体靠 string code/key 互引：DictType/ConfigKey 等均 string，非实体主键），故入参侧**只处理 path `:id`**，不动 body。任务书在此明确，执行端勿误以为有 body ID 待收。
  - rbac 业务/码值/签名改动（idcodec 抽取仅搬迁，非重构 rbac 逻辑）。
  - T-003d-fix 那条陈旧红测（独立切片，不混入本片）。

### 3. 数据模型
- **无 DDL 改动**（持久层 id 仍 `uint64`；hashid 是出入边界的对外表示，DB 不变，沿 T-003b 既定模式）。
- **现状基线（执行端已据实交付，作为收口对象，逐条核对消除）**：
  - 出参裸 uint64：SysDictType(model.go:24)、SysDictData(model.go:36)、SysConfig(model.go:54)、SysFile(model_file.go:19)、SysOperLog(model.go:78)、SysLoginLog(model.go:97)，全经 `response.OK` 直吐 model/list。
  - path ParseUint：`handler.go:197` pid()、`handler_file.go:80`/`:116`。
  - system 包当前**零 Hasher/ResponseEncoder 引用**（与 rbac 解耦，靠 PermGuard 接口）。

### 4. 接口契约
- system 全部含 `:id` 的 path 参数与出参实体 schema 改 hashid 字符串；逐一列在完成报告"接口实现情况"。
- 统一响应包络不变；非法 hashid 走既有 ErrInvalidID（400）。
- **解耦硬约束（最重要的架构红线）**：
  - 抽 `idcodec` 后，**system 依赖 `idcodec`、rbac 依赖 `idcodec`，system 与 rbac 之间无直接 import**。
  - `PermGuard` 接口解耦（T-003d）原样保住，不得因引 hasher 反向耦合。
  - 完成报告须附 **`go list -deps` / import grep 证据**：证明 `system` 包的 import 列表中**无 `rbac`**、`rbac` 与 `system` 均只新增对 `idcodec` 的依赖、无环。

### 5. 安全要求（对照架构文档安全章节 §8 / 防枚举）
- 非法/伪造/空/越界 hashid → 400，不回落、不暴露内部 id、错误信息中立（沿 T-003e 防探测：不区分"不存在 vs 格式错"）。
- **`sys_file` 的 `:id` download/delete 是本片安全收益核心**：hashid 化叠加既有 RequirePerm（T-004b），构成纵深防御——即便鉴权某处疏漏，攻击者也无法靠枚举 id 遍历他人文件。
- dict/config 的 id 防枚举收益较弱（真句柄是 code/key），但纳入全套是为"全链路零裸 uint64"明线一致性，避免 system 模板把裸 uint64 传染进消费方业务模块。
- hasher（idcodec）盐/配置经装配注入，禁包级可变全局（沿 T-003a/b 铁律）。

### 6. 规范约束
- 代码头注释五项到秒；搬迁/改既有文件追加 `@updated YYYY-MM-DD HH:mm:ss`；idcodec 新文件用底座项目名头注释。
- 参数化：idcodec hasher 配置注入，零硬编码盐/前缀。
- 重构纪律（idcodec 搬迁段，对标 T-004c）：rbac 侧不改码值、方法签名、出入参行为；搬迁后 rbac 全部既有单测/集成零回归为硬证。
- Casbin/perm code 不动（本片不碰鉴权语义）。
- 装配/RegisterRoutes 若需新增 system 的 hasher 注入参数，属对外契约变更，完成报告显式标注（demo 装配须跟进；BenxinKP 等消费方接入 system 时须注入 idcodec hasher）。

### 7. 验收标准
**执行端自测项（自动化）：**
- [ ] **idcodec 搬迁零回归**：rbac 全部既有单测 + 集成（`-tags=integration`）全绿，hashid 入出参行为与搬迁前一致（rbac 出参 hashid 用例、T-003e 入参用例均守门）
- [ ] **解耦证据**：`go list -deps`/import grep 证 system 不 import rbac、二者均只新增 idcodec 依赖、无环（附输出）
- [ ] **system 出参 hashid**：6 个实体 list/detail 出参 `id` 均为 hashid 字符串（单测断言 + 集成真 MySQL 验）
- [ ] **path 收口**：dict/config PUT/DELETE 与 file download/delete 的 `:id` 接 hashid 通、裸数字/乱码 → 400（单测 + 集成）
- [ ] **grep 全链路零裸 uint64 主键**：system 包对外（出参 + path）无残留裸 uint64 主键（grep 佐证；持久层 model id 字段除外，经 encoder 编码出参）
- [ ] **typed-nil 自检修正**：demo 自检对 typed-nil（含构造一个 `(*Hasher)(nil)` 反例）能 fail-fast（单测坐实新写法抓得住，旧写法抓不住的对比）
- [ ] **demo e2e 仍 ALL PASSED**：装配跑通，含 system path 用 hashid 的调用（至少补一条 file download by hashid :id 进 e2e，或集成等价覆盖）
- [ ] openapi v0.10.0，redocly 0 error；两处已知不一致已修；system 出参 schema 全 hashid
- [ ] `go build && go vet && go test`（含 integration）全绿（先前已红的 TestNewEnforcerMySQL_RoleInheritance 属 T-003d-fix，不在本片责任，但须确认本片未新增红）

**daxing 真人验收项（API 层）：**
- [ ] 用 demo/curl：`GET /sys/files` 列表返回的 `id` 是 hashid → 用该 hashid `GET /sys/files/{id}/download` 成功取到文件；裸数字 `:id` / 乱码 `:id` → 400
- [ ] 抽验 dict 或 config：列表 `id` 为 hashid → 用该 hashid `PUT`/`DELETE` 通；裸数字 → 400

### 8. 测试与提交要求
- 测试：单测 + 契约 + integration + demo e2e 由执行端负责；API 层 curl 复核由 daxing。
- **测试铁律（#3/#4/#5 教训）**：idcodec 抽取是跨包搬迁、system 新引 hasher 进装配——典型"零件绿≠装配跑"风险区。**必须 demo e2e 真跑兜底**（不能只验 system 单模块单测），且解耦证据要用 `go list -deps` 实证而非"我看着没 import"。
- git 提交：执行端双仓（Gitee 主 + GitHub 镜像），提交前查 .gitignore（确认不夹带 `*.local.yaml`/`.env`/数据产物），**push 前贴 `git status`+`git diff --stat` 待 PM 核 → 经放行才双推、两 remote 各贴回执**。
- 本机依赖：MySQL/Redis 走 docker-compose，应用原生跑。
- **流程铁律**：完成判定权在 PM；执行端回交完成报告 → PM 评审 + daxing API 验收 → PM 放行 → 双推 → 翻 ✅。不得自标完成、不得自行双推、不得擅改 PROJECT_STATUS。

---

> 完成后请按 `TASK_TEMPLATE.md` 文末「完成报告」格式回交可复制 markdown。
> 重点附：① idcodec 搬迁 rbac 零回归证据 ② **解耦证据（go list -deps，system 无 rbac import、无环）** ③ system 出参/path hashid 化清单（逐实体逐接口）④ typed-nil 自检新旧写法对比单测 ⑤ openapi diff 摘要（两处已知不一致已修）。
