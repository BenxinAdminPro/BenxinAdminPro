# PROJECT_STATUS — BenxinAdminPro 底座（进度账本 · 跨对话接续权威来源）

> 每个新对话开始先读本文件；每完成一个切片后更新本文件。本文件优先于"记忆"。

## 项目元信息
- 项目：BenxinAdminPro（本心通用管理后台）。前后端同仓：`server/`（Go）+ `admin/`（Element Plus）
- 定位：开箱即用的完整后台系统 + 可复用底座（脚手架 + 模块库）
- 架构文档：BenxinAdminPro-架构与规划文档 v1.1
- 仓库：Gitee 主仓 https://gitee.com/benxin-admin-pro/benxin-admin-pro.git ；GitHub 镜像 https://github.com/BenxinAdminPro/BenxinAdminPro.git（独立于 BenxinKP）
- 本地：/Users/daxing/projects/BenxinAdminPro（与 BenxinKP 平级，多根工作区聚合）
- 最后更新：2026-06-15
- 当前 HEAD = T-009a 后台禁搜索引擎收录 · 报告归档提交（= 最终 HEAD）。提交链 a4b3fc8（T-009a feature）→ 账本（本提交）→ 报告归档（最终 HEAD）。账本提交时后两节 hash 未生成，**HEAD 语义 = 报告归档提交（非 feature）**，双推后回报 PM 三处实际 hash 对账。
  - 上一收官：HEAD d89e3e2（T-005b-3 配置加密参数写链路）。提交链 1d21cf4（feature）→ 2b96a09（账本）→ d89e3e2（报告归档）。三方 HEAD 一致（ls-remote 实测）。
  - 再上一收官：HEAD f88d3c7（T-008c 角色授权树）。提交链 1d4b9aa→fe6ea09→6988ee2→f88d3c7。

## 核心原则（铁律）
- 业务中立：不进任何业务概念。
- 参数化：禁硬编码 kp_/okp:/BenxinKP；表前缀、Redis 前缀、错误码段、@project 全由配置注入。
- 表前缀随实例走，禁任何包级/全局可变前缀（gorm NamingStrategy{TablePrefix} 绑 *gorm.DB）。
- 纯 Go + 语言中立 spec（供 PHP 等实现 parity）。
- SemVer 库复用，绝不为单应用改底座源码。

## 协作约定
- Cursor / Claude Code：编码 + API/单测/契约测试 + git 提交（双仓库）。
- daxing：用 examples/demo 真人验收 + 关键决策 + git remote/凭证 + 双推确认。
- 节奏：纯 Go 垂直切片；每个能力须在 demo 跑通。
- 本机：docker-compose 仅起依赖，应用原生跑；生产不用 Docker。
- PM 评审硬规矩：真后端正确性由 `-tags=integration` 集成测试兜底；单测用内存假实现/SQLite 保 CI 快。

## 代码头注释（五项，到秒）
项目名中英文 / 本文件功能 / 作者 / 邮箱 / 生成日期时间。改既有文件追加 @updated。模板见 .cursor/rules/benxinadminpro.mdc §6。

## 环境就绪状态
- ✅ 本地目录骨架、go mod init、admin Vite+Vue3+TS、根文件 + .gitignore（含 *.local.yaml / config.yaml / .claude/）、多根工作区、deploy/docker-compose.dev.yml（MySQL 8.0 + Valkey 8.x）
- ✅ git init + 双仓推送链路（Gitee origin/main + GitHub github/main）；注：近期推送 SSL_ERROR_SYSCALL 抖动转频，必要时给 remote 走代理或换 SSH
- ⏳ 待办：放置 .cursor/rules/benxinadminpro.mdc（如尚未提交）

## 里程碑
- 🏁 RBAC 整块完成（T-001 安全地基 → T-002 认证 → T-003 组织+RBAC a/b/c）。认证→功能权限→数据权限闭环。HEAD 当时 e167098。
- 🏁 **Go 后端底座五大块全部完成（T-001 安全 / T-002 认证 / T-003 RBAC / T-004 系统管理 / T-005 配置中心）+ 渲染债清（T-004c）。**
- 🏁 **T-006 demo 真·跑通（e2e 冒烟 8 步 ALL PASSED）+ T-003d 鉴权接线修复。** 注：demo 装配过程逼出并修掉三个潜伏缺陷（#4 casbin 版本冲突 / #5 迁移器静默不建表 / #3 功能权限 enforce 未接线），均在远程仍干净时拦下。
- 🏁🏁 **阶段一收官（HEAD 78342cf）：后端五大块 + demo e2e + admin 全套（地基 T-007a / 验证码 T-002b / 权限+CRUD T-007b）。admin 已是"按权限工作的可用后台"——超管全功能、普通角色菜单/按钮/数据范围收窄、刷新不白屏、登录链路完整。底座"成品+底座"双重身份均立住。**
- 🚀 **前端迭代批次开局（"把底座 admin 做厚"，daxing 全按建议）**：决策定档——锚点=消费方驱动（x-table 厚度 + 后端入参收口为必还债先行，页面铺量按需）；x-table 本批长到只读模式+工具栏/行操作插槽+列筛选排序（树形归 x-table 内建 tree、前端全量、无 lazy-load）；导出 defer；vitest 本批引入单列不阻塞。切片执行序：**T-003e 后端入参收口（✅ 621f7a8）→ T-004d system 对外 ID hashid 化（✅ 85067fb，含抽 idcodec 中立包保解耦）→ T-007c x-table 增强（✅ 2b0505e）→ T-004e 重名→友好错误码（✅ 7b2aca4）→ T-007d 字典+参数页（✅ ac9b3b0）→ T-007e 操作+登录日志页 → T-007f 文件管理页 → T-007g 菜单树形页（建树形能力）→ T-007h dept/post 选择器+回填（gate 在 T-003e，已解锁）→ T-007i vitest 基建（可选）**。注：T-007g 菜单树形提到 T-007h 之前，dept 选择器复用菜单页产出的 buildTree。
- 🏁 **T-007d 字典/参数管理页双推归档（HEAD ac9b3b0）**：T-007c（x-table 增强）+ T-004e（唯一键友好码）首个端到端回证消费页。字典类型↔数据双表联动 + 参数 sys_config CRUD + 加密参数安全降级，两条 409 友好码（dict_type 11060 / config_key 11062）前端闭环回证非 500。读源码兜住任务书两处假设（加密 留空/重填 语义后端不支持、列表无 search/sort），衍生 4 项后端缺口已上报挂切片（见待办 T-005b）。
- 🏁 **T-007e 操作/登录日志页双推归档（HEAD 9e5ebb4）**：x-table 只读模式首个真消费页 + 偿还 T-007c §8 B 部分 operlog/loginlog 种子缺口。纯前端消费 + demo seed 增量，未改后端 Go 业务、未升 openapi（仍 v0.10.0）、未新增错误码。PM blocker 补证坐实 enforce 三段链路（F 行入表 → role_menu 授权 → ReloadAll 重建 policy）：dept_mgr 正向 enforce 四端点全 200 ↔ editor 全 403，policy 删除-再生对照（casbin_rule 72→手删 8→64→重启再生 72）。浏览器验收三证齐全（含产品内操作日志列表可视复现 enforce 对照）。源码核实纠正典型 admin 假设：无 /:id 详情端点（行内弹窗）、清理固定删 3 个月前（无入参/无多选）、operator 落内部 ID（§8-1 上报 T-005b）。
- 🏁 **T-007f 文件管理页双推归档（HEAD 5c998ba）**：x-table 消费 + 本批首个真实写动作页（上传/下载/删除）+ 偿还 T-007c §8 B 部分 sys:file:* 种子缺口。纯前端消费 + 请求层 blob 透传（鉴权下载必要前置）+ demo seed 增量，未改后端 Go 业务、未升 openapi（仍 v0.10.0）、未新增错误码。enforce 正向证据**执行端前置自带**（吃 T-007e blocker 教训）：dept_mgr list/upload/download/delete 四端点含写动作全 200 ↔ editor 全 403，policy 删除-再生 80→72→80。源码核实纠正典型假设：下载无 tokenized URL（逼出 axios blob 带 JWT 取流，禁裸 a href）、字段是 mime 非 content_type、上传单文件字段名 file、删除单条软删无多选。浏览器验收齐：上传正常入列 + 负例 >10MB 中文拦截 / 下载 blob 落盘文件名正确 / 删除干净文件行消失共 2→1 条（refresh 真生效，早前"删存量死链行成功但行未消失"定性为 §8-5 边缘表现非 bug）/ storage_key 不裸露 / 上传人列空印证 §8-1 后端缺陷。挖出 T-004b 两潜伏缺陷（uploader 类型断言恒失败连 ID 都没存上 / file 文案缺 6 键返裸 key），并入 T-005b（uploader 标优先项）。
- 🏁 **T-007g 菜单树形管理页双推归档（HEAD 33679a3）**：前端迭代批次第五片 + **建 buildTree 通用树工具（T-007h dept/post 选择器前置产出）**。纯前端消费（3 新增文件 +576：utils/tree.ts + api/menu.ts + views/sys/menu/index.vue），seed/后端 Go/openapi/错误码全零改动。源码核实#2 坐实**后端 GET /sys/menus/tree 已返嵌套树**→ 前端表格直接消费零再组装，buildTree 真消费点=父级选择器 flatten→subtreeIds 排除自己及子孙→重建（且为 T-007h 备好通用扁平→树）。buildTree 抽独立模块、键名参数化、防环/防孤儿降级+dev 告警，21 项脏数据自测 ALL PASSED（孤儿/自引用/互引环/三元环/环状嵌套）。menu_type 三类型动态表单镜像后端 validateMenuType（F 必填 perm_code、M/C 强制空串、全量覆写、父子类型后端无约束故前端不做伪安全）。enforce 执行端前置自带：dept_mgr tree/create/update/delete 含写动作全 200 ↔ editor 全 403/11009，policy 80→80→80。浏览器验收齐。**挖出 T-003b 缺陷：菜单 CUD 不触发 Casbin policy 重载（改/删 F 的 perm_code 后 p 规则滞留→删 F 权限收不回的安全后果），提级 T-005b 优先·安全相关项（非中性观察）。**
- 🏁🏁 **T-007h dept/post 选择器 + 编辑回填双推归档（HEAD 2460013）：前端迭代批次第六片，B 部分种子最后一块清零、前端铺厚批次基本收尾。** buildTree **首个复用者**（验证 T-007g 中立工具产出：dept 树后端已嵌套故零调用，但参数化够用性逐字段验证通过、tree.ts 零触碰、将来部门管理页可零参数复用）+ dept 树/post 多选选择器嵌入用户表单 + **编辑回填（T-003e 入参 hashid 收口的兑现验收，API 五步+DB 实查 junction 行坐实）** + 岗位管理页（补 sys:post:* 种子）。XTable 三项中立扩展（slot 字段/api.get 回填/delConfirm）属必要前置非擅扩、缺省零回归。policy 80→88（sys:post:* 4 码×2 角色）。浏览器验收：岗位页 CRUD + 用户编辑部门/岗位回显正确。**连带修复 T-007b 静默资料损坏缺陷：编辑表单原无 dept_id 字段→每次界面编辑静默清空用户部门，本片回填机制闭合 + api.get 失败不开弹窗防呆（"装配/真界面操作才暴露"潜伏缺陷模式又一例）。**
- 🏁🏁🏁 **T-007i admin 测试基建双推归档（HEAD c2a3d90）= 前端铺厚批次 T-007a~i 全部收官。** 引入 vitest（独立 vitest.config.ts，复用 vite.config 的 @→./src 别名、environment node、最小依赖未引 jsdom，与自建脚手架 Vite8/TS6 工具链对齐）+ 收编 T-007g 一次性 node 自测脚本为正式入仓单测（src/utils/tree.spec.ts，buildTree/flattenTree/subtreeIds 共 17 条脏数据/防环断言，覆盖附录 B 8 类 + 补全 subtreeIds 祖先/旁系排除等隐含点）。**测试基建自身防假绿三证**：① 故意改坏断言→真 FAIL→还原→PASS（证明非空跑/全 skip）② 首条断言校验 import.meta.env.DEV===true（否则 dev 告警断言会集体假绿）③ 孤儿/环用例 vi.spyOn console.warn 断言告警被调用。tree.ts 零改动（中立工具铁律）；spec 让 vue-tsc 类型干净通过（纳入类型网而非排除，符合 Vue 脚手架惯例）；scope 未外扩（不给 x-table/请求层/各页铺测试）、依赖全 devDependencies。daxing 真人 `pnpm test` 17 PASS + `pnpm build` exit 0 验收通过。**底座补上前端测试基建短板——"成品+底座"双重身份的底座侧再加一块。**
- 🏁🏁 **T-005b-1+2 双推归档（HEAD f42dc80）= 前端批次收官后转后端债批次首切，T-005b 篮子两个 ★ 优先项一次清零。** T-005b-1（★安全）菜单写路径 create/update/delete 接 Casbin policy 全量重载联动，修 T-003b 既有安全缺陷（菜单 CUD 不触发 policy 重载→删 F 节点权限收不回，"以为关了门其实没关"）。验证=policy 生命周期对照：建/改/删 F 各步 casbin_rule **真跟着变**（对照 T-007g 观察到的 80→80→80 纹丝不动那正是缺陷态）+ **enforce 即时翻转**（dept_mgr 改 F perm_code 后对应端点 200→403、删 F 后 403、改回恢复 200 = 安全洞真堵的证据）。全量 ReloadAll 决策（从 role_menu JOIN menu 真相重建、不误删其他 policy、与 role/user 同范式）；SetPolicySync 可选 setter 不改构造签名零回归；事务口径=先提交后重载、失败返错幂等恢复、与既有 AssignMenus 同级（真原子需改 PolicySync 读 tx 超 scope，记待办）。T-005b-2（★装配静默失败）auth.Claims 加 GetSubject() 方法，修 T-007f 暴露的 uploader 恒空（鸭子断言 `interface{GetSubject()string}` 对只有 Subject 字段的 Claims 静默失败返零值，同 T-006 类病）→ sys_file.uploader 真落值、过滤可用。**首个后端 Go 业务改动切片（前九片纯前端消费）——验收暴露教训：改后端 Go 必须重启 demo 才生效（前端有热更新无需重启），daxing 初次验收因 demo 跑旧二进制看到 uploader 仍空，重启后落值。** pre-existing red（T-003d-fix）git stash 干净 HEAD 复跑实证非本片引入。两支集成测试入仓（menu_policy / uploader）。
- 🏁🏁 **T-005b-4 双推归档（HEAD 4e16f91）= T-005b 后端债批次第二片：system/日志列表查询能力补齐 + 操作人可读化。** 给 dict/config/oper_log/login_log 列表补 search/filter/sort + 时间范围、dict_data 改服务端真分页，并把 oper_log.operator / sys_file.uploader 在出参解析为用户名（已删→「已注销」、空→「匿名」）。**操作人可读化选型 B（出参 JOIN）**：核实 operator/uploader 存 claims.Subject=内部自增 ID 与 sys_user.id 干净对应、无非用户主体混入（B 无阻断）→ 非持久化 OperatorName/UploaderName（gorm:"-"）批量 IN 解析（非 N+1：收集本页非空 ID→去重→一次 WHERE id IN 建 map→回填），过滤入参改「按用户名模糊」（用户名→ID 集→列 IN），不新增内部 ID 出参字段（守 T-004d）。**「已注销」机制=「未命中即已注销」且显式 deleted_at IS NULL**（关键：resolveUserNames 走原生 .Table()，GORM 软删 scope 在 .Table 下不自动生效，故显式写排除——又一个「.Table 下软删 scope 静默失效」潜伏点，前置兜住）。**排序强白名单**（用户输入永不进 ORDER BY，sort 经白名单映射列名、未命中回退默认不报错；负例 `id; DROP TABLE x;--` 被忽略实证）+ LIKE 转义 + 时间范围非法/start>end→400 不静默吞 + 分页封顶 100 + JOIN 仅 SELECT id,username 不带 password_hash。dict_data 从全量数组→分页包络（与 enc.Page 逐字同构）。**最小前端配套（必要前置非擅扩，同 T-007f blob 先例）**：dict 适配层一行切真分页（去本地切全量）+ operator/uploader 显示列改绑 _name + uploader 过滤随显示改按用户名 + loginlog/config 纯注释诚实修正（旧降级注释现已就绪）。openapi v0.10.0→v0.11.0（新查询参数 + operator_name/uploader_name 出参 + dict_data 响应形态）。**daxing 验收五证**：①操作日志 operator 列显 admin 用户名非 ID ②删用户→已注销认集成测试证据 ③dict_data 右表 25 条切三页（第 2 页项-11~20 非重复第 1 页/非全量堆出，服务端真分页）④文件页 uploader 列新上传显 admin、历史空显匿名 ⑤时间范围 curl 五条全对（全量146→限今天后46→限2020前0→两负例400，且返回行 op=admin 证可读化在时间范围路径同生效）。**enforce 前置自带**：dept_mgr 6 带新参数端点全 200 ↔ editor 全 403。后端 14 改 + 3 新（query.go/2 集成测试）+ admin 8 文件，+1076/-114。零新增错误码段、零 DDL。commit 4e16f91。
- 🏁 **T-008a 双推归档（HEAD c2e620a）= 用户/角色管理补缺片首片：用户改密码 + status 假能力修复。** 纯前端、零后端 Go 改动、不升 openapi（消费既有端点）。**起因**：daxing 在 T-005b-4 验收时撞上前端缺口——新建用户无法在界面分配角色/授权，登录后只有工作台。摸底坐实一批「后端有、前端缺」的关系/管理装配界面缺口，PM 决定开「用户/角色管理补缺片」插队到 T-005b 后端债之前。本片是其中最轻、零后端依赖的一片先做回血。**重置密码**：用户页行操作加「重置密码」（v-permission sys:user:password）→ 弹窗输入+二次确认 → PUT /sys/users/:id/password（入参 {password} 仅 binding required 无强度校验，前端预校验只做非空+二次确认不伪造规则）。**status 假能力修复（修 T-007h §8-3）**：根因=updateUserReq 无 Status 字段（handler_user.go:312）→ 编辑弹窗改 status 被静默吞。修法=status 标 createOnly 使编辑弹窗物理不渲染（XTable.vue:132 过滤）+ 行操作「停用/启用」按钮调 PUT /sys/users/:id/status（status 语义 0=正常/非0=停用，model.go:45 + 登录拒判 service_auth.go:136）→ **状态变更唯一路径=独立端点**（后端本就不收 + 前端不渲染双保险，根除非遮住）。**偏差①（PM 接受）**：§2 原写「列内 el-switch」改为「行操作按钮」——列内 switch 需给 x-table 加单元格插槽=碰核心（§2 禁），覆写 #row-actions 又丢内置编辑/删除。功能等价零核心改动。口径：将来 x-table 若要支持列内自定义单元格控件，作有意识能力增强独立评估（像 T-007h field-slot 那样全可选缺省零回归），不为单页临时塞。**daxing 验收三过**：改密后旧密码失效/新密码登录成功、停用拒登/启用放行、编辑弹窗已无 status 字段。enforce：editor PUT password/status 403↔dept_mgr 200。admin 2 文件 +148/-2。commit c2e620a。
- 🏁 **T-008b 双推归档（HEAD 1d4b9aa）= 用户/角色管理补缺片第二片：用户分角色 + 列表角色列。** 后端搭车补回填查询 + 前端主体 + openapi v0.12.0。后端 4 改 + 2 新集成测试，前端 3 改，共 9 文件 +578/-4。**写端点已有不动**：PUT /sys/users/:id/roles（sys:user:assign，入参 role_ids hashid 数组，service 纯覆写 = 先删该 user 全部 user_role 再按 role_ids 重建 + 联动 Casbin SyncUserRoles g 规则）。**回填选型 A（GET /:id 预载 roles，PM 默认，核实无阻断）**：UserService.Get 仅 1 调用点（grep 实证）+ enc.User 手工构建出参（password_hash 从不进表）→ 加 roles 数组零污染其他出参、List 不预载（omitempty）；与 T-008c「角色详情预载已授菜单」同口径统一；无新端点、无新 perm（复用 sys:user:get）。否决 B（单独端点）：Get 单调用点使 A 无副作用。**列表角色列（daxing 验收追加诉求，归本片闭环非另开片）**：UserService.List 一处 fillUserRoles 批量回填——查询③ Model(&SysUserRole{}).Where(user_id IN pageIDs) → userID→[]roleID map + distinct roleIDs；查询④ Where(id IN roleIDs).Find(&roles)（**model 查询让软删 scope 剔除软删角色，不用 .Table()——反用第 7 例潜伏点：这次要 scope 生效故用 model 查询**）→ 内存一对多分组回填。**批量非 N+1 硬证**：GORM After("gorm:query") 回调计数，2 行页=4 次查询==6 行页=4 次（Count+用户+junction+角色固定不随 N 增长，N+1 会是 N 次）。前端角色列走 formatter 顿号文本（无角色「—」），**不用 el-tag**（PM 裁定：el-tag 需 x-table 单元格插槽=碰核心，§2 禁，同 T-008a status 落点；x-table 列内自定义单元格控件诉求已第三次，记可专门排基建片）。**daxing 验收四过**：①分配弹窗回显当前角色 ②覆写不误删（temp05 编辑员+加部门经理→两个都在，原有未冲掉；集成测试 2→3→1 DB 行数精确）③**分配角色后登录看到菜单**（temp05 挂 dept_mgr 后登录侧边栏完整菜单全出 ↔ 挂空菜单 editor 时只有工作台，坐实链路正常）④列表角色列正确显示 + 分配后即时反映。**editor 空菜单核实结论 A（非 bug）**：editor 角色 seed 只授 1 个 sys:user:list（menu_type=F 不进侧边栏），授的 M/C 可见菜单数=0 → 空菜单是数据如此；GetUserMenuTree 链路正确（menu_type IN(M,C) 过滤后空集是正确结果）；交叉对照 dept_mgr 同链路满菜单 ↔ editor 空，唯一差异=授权数据 → 坐实链路无 bug。**真正的「给角色配可见菜单」能力缺口 = T-008c 角色授权树**（editor 留 T-008c 授菜单后补验）。集成测试坐实 Casbin g 联动真生效（分配含 sys:user:list 的角色→probe 登录 GET 200，清空→403）+ password_hash 不泄漏（A 预载后 GET /:id 仍无 password_hash，守 T-003a）。commit 1d4b9aa。

- 🏁🏁🏁 **T-008c 角色授权树双推归档（HEAD f88d3c7）= 用户/角色管理补缺片压轴收官，权限体系两段界面闭环成型。** 角色页加 el-tree「分配菜单」授权树弹窗（**check-strictly 独立勾选** + 回填全量已授 menu_ids + 全量覆写提交 PUT /sys/roles/:id/menus）+ 后端搭车给 GET /sys/roles/:id 补 menu_ids 回填来源。**写路径 + Casbin 联动零改动**：AssignMenus + SyncRolePerms 自 T-003b 已完整（role_service.go:151-190，成功增量 SyncRolePerms 只重写本角色 p 规则 + 失败 ReloadAll 兜底），**区别于 T-005b-1 的菜单 CUD 当时缺联动需补**——摸底坐实角色授菜单链路本就完整，本片后端只剩回填读端点这一小补。**选型 check-strictly（PM 裁定·安全优先）**：摸底坐实 role_menu M/C/F 三层扁平原样存（AssignMenus 不展开不裁剪）+ GetUserMenuTree 取全量后 menu_type IN(M,C) 过滤渲染、buildMenuTree 对父目录缺失的 C 走孤儿提根（不丢权限/不丢页面、只丢目录分组）→ **GetUserMenuTree 不要求父 M 进 role_menu 才正确** → check-strictly 成立（setCheckedKeys(stored)≡getCheckedKeys() 恒等往返，从根上消灭回填/提交不一致面，规避级联两经典坑：回填污染越权扩权 + 半选丢父）。**回填 A 方案**（GET /:id 预载 menu_ids，同 T-008b 口径）：RoleService.Get 全仓唯一调用点（handler_role.go:64）、List 走独立 enc.RoleList → 加 menu_ids 预载零污染列表/其他出参；SysRole 加 MenuIDs []uint64 gorm:"-"（镜像 SysUser.Roles 范式）。openapi v0.12.0→v0.13.0（SysRole schema 加 menu_ids hashid 数组、详情 only、omitempty），零新增错误码段、零 DDL、零新 perm（复用 sys:role:list/assign）。**editor 补验闭环兑现（T-008b 留尾）**：给 editor 授可见 M/C 菜单 → editor/temp05 无痕登录侧边栏从空（原只工作台、seed 只授 1 个 F 码无 M/C）变出菜单，坐实「给角色配可见菜单」能力缺口已补。**daxing 验收全过（含命门项大集合改一处不误删）**：① 回填全量（dept_mgr 全树勾满）② check-strictly 父子不联动（editor 用户管理父勾、其下仅用户列表勾）③ **大集合 round-trip 不漏勾**（dept_mgr 满授权取消「菜单删除」单项 → 重开确认仅那一项变空、其余文件/字典/日志数十项纹丝不动，比集成测试 DB 行数精确多一层真界面坐实）④ editor 补验侧边栏变满 ⑤ check-strictly UX 代价可控（整片勾父目录后侧边栏正确嵌套、不再平铺）。集成测试五项全绿（回填全量/覆写往返不丢无残留/enforce 正向 dept_mgr 200↔editor 403/policy 生命周期：授含 F 集 p=1+探针登录 200、去该 F p=0+探针 403/enc.Role 无 password_hash）。**已知 UX 非 bug 活体印证**：check-strictly 下勾 C 不勾父 M 时该页侧边栏平铺顶层（GetUserMenuTree 孤儿提根），按目录整片勾即正常嵌套——「授页面隐含授目录」若要做属后端+spec 独立决策，本片前端不偷做（守语言中立 spec parity）。**连带证 T-008b 列表角色列零回归**（temp05 显「编辑员、部门经理」顿号拼接，本片动 enc.Role/RoleService.Get 未碰 User 侧）。后端 4 改 + 1 新集成测试，前端 2 改，共 7 文件 +160/-6。commit f88d3c7。**「用户挂角色（T-008b）+ 角色挂菜单（T-008c）」权限体系两段界面均可装配——补缺片核心使命达成。**

- 🏁🏁 **T-005b-3 配置加密参数写链路双推归档（HEAD d89e3e2）= 后端债批次第三片：加密参数「建/编/读/脱敏/seed」闭环成型。** 让加密参数能经管理 API 正确**新建**（落密文）与**编辑**（不破坏密文、留空即保持），并解除 T-007d 前端加密行编辑禁用——全程对外不回显明文。10 文件 +683/-65（后端 7 改/新：dict_service.go/handler.go/main.go/seed.go/openapi.yaml/dup_integration_test.go + 3 新测试；前端 1 改）。**地基（摸底头号发现）**：ConfigService{db,errs} 是 handler CRUD 实际用的服务、**不持有 GCMKey/不经 ConfigCenter**，而 ConfigCenter.EncryptValue 持密钥但**全仓零调用者**（死代码）→ 写路径根本够不到加密能力 → 本片先经 **SetGCMKey 可选 setter**（不改构造签名零回归，同 SetPolicySync 范式）给 ConfigService 注入 gcmKey。**新建**：CreateConfigInput 补 is_encrypted，=1 走 EncryptGCM 落密文 + IsEncrypted=1（原 Create 明文直存且从不设 IsEncrypted 恒 0）。**编辑（命门·防潜伏数据损坏）**：拆独立 **UpdateConfigInput（ConfigValue *string 指针三态）**——因对外恒 mask（****），前端编辑表单永远拿不到明文，**不能用「值是否为空」隐式判断是否改密文**（原样回填会把 ****** 当新值毁密文）→ 指针语义：**先 First 取该行现有 is_encrypted 作加密判据（不信入参防类型偷改）**；ConfigValue==nil（payload 省略字段）→ Updates map **不含 config_value 键**（GORM 只更新存在的键）→ 密文/明文原样保留；非 nil 且现有 is_encrypted==1 → EncryptGCM 重新加密覆盖；非 nil 且 ==0 → 明文覆写。**编辑态键/类型锁定**：UpdateConfigInput 不含 config_key（禁改，对齐前端 disabled）、不含 is_encrypted（明↔密切换走删除重建，不在编辑态做）。**前端解禁**：去 T-007d 的 disabled+tooltip；新建弹窗加 is_encrypted 选择（select 默认明文）；加密行编辑值框**打开即空**（不回填明文/******）+placeholder「留空＝保持原密文，填写＝重新加密替换」；submitEdit 按加密标志分流（加密行空值省略 config_value=保持、明文行始终提交=零回归）。**错误码复用 ErrConfigDecryptFailed**（既注册）兜运行时加密不可用/失败（is_encrypted=1 但 gcmKey 空、EncryptGCM 真错），不开新段；**ConfigService 缺密钥不 panic**（底座库对明文参数不强依赖密钥，运行时降级返错）+ demo 装配自检加「加密能力就绪(gcmKey 非空)」第二道 fail-fast（执行端补丁，PM 接受——补 setter「忘调→运行时才暴露」弱点）。**seed 搭车**：补明文(site.title)+真 EncryptGCM 加密样例(demo.secret_token)，免手工 SQL、连跑两遍幂等。**openapi v0.13.0→v0.14.0**（create 加 is_encrypted+remark；update 重写键/类型锁定+指针语义；搭车补回 T-008c 漏掉的 v0.13.0 changelog 行），零新增错误码段、零 DDL、零新 perm（复用 sys:config:create/update）。**头号正确性证据=解密往返三态**（因对外恒 mask，密文是否破坏唯有「ConfigService 写 → ConfigCenter.GetConfig 解密读」往返可证）：① 新建加密 GetConfig 解出==原明文、落库合法 base64 信封非明文字面 ② Update 带新值解出==新明文密文换新 ③ **Update 不带值(nil) 密文原样保留、解出==原明文、name 已更新**（防损坏命门坐实）；明文零回归 + 无密钥降级返 ErrConfigDecryptFailed 不 panic 均断言（config_crypto_test.go SQLite 进默认 go test 闸门）。**全栈 e2e + 写路径 enforce**（config_write_integration_test.go 真 HTTP+MySQL）：dept_mgr POST/PUT 全 200 ↔ editor 全 403；HTTP 建 is_encrypted=1→DB 真密文(DecryptGCM 还原)+列表恒 ******；编辑三态留空保持/填值替换全栈坐实。seed 幂等（seed_config_integration_test.go）：连跑两遍 sys_config 2 行稳定、加密样例真密文不被覆盖。**设计性移除 TestDupConfigUpdateRename_MySQL**（PM 批准）：config_key 编辑锁定后「改名撞键」按设计不可达，非凑绿删、留注释，Create 侧友好码仍由 TestDupConfigSimpleCreate 覆盖。**daxing 浏览器验收六项全过**：新建加密显 ****** / 留空保持 / 填值替换 / 编辑值框打开即空+placeholder / 明文编辑零回归 / editor 403（集成测试兜底）。commit 1d21cf4。**加密参数写链路补齐——T-007d 安全降级禁用的入口至此正式放开，加密参数全生命周期闭环。**

- 🏁 **T-009a 后台禁搜索引擎收录双推归档（HEAD = 报告归档提交，feature a4b3fc8）= 新需求池首条落地，noindex 三件套 + 参数化权威控制点。** 让管理后台默认禁止被搜索引擎收录（登录页/接口路径被爬是攻击面），且禁与否由部署配置控制（守底座中立，私有化可关）。9 文件 +285/-6（后端新建 httpmw 包 + 中间件 + 单测 + 集成测试 + demo 装配/config + config 库参考结构体；前端 robots.txt 新增 + index.html meta）。**三件套**：① **后端 X-Robots-Tag 全局中间件（唯一碰 Go、参数化权威控制点）**——新建 `httpmw` 通用 HTTP 中间件包 + `XRobotsTag(RobotsConfig)`，挂 engine 级 `r.Use`（Recovery 后、所有路由前）对**所有响应无差别注入**（含鉴权失败 401/未匹配 404，全栈测试坐实）；开关 + 内容走静态配置，`DefaultXRobotsTag="noindex, nofollow"` 常量单一定义、逻辑不硬编码字面。② **admin `public/robots.txt`**（`User-agent: * / Disallow: /`，Vite 拷站点根）。③ **admin `index.html` meta** `<meta name="robots" content="noindex,nofollow">` 构建期兜底。**默认安全（命门）**：loadConfig `v.SetDefault("security.x_robots_tag.enabled", true)` + content 默认 → 配置整段省略也走禁收录（viper GetBool 缺省返 false=不安全，SetDefault 兜底）+ 中间件内容空回退 DefaultXRobotsTag（开了不注入空头）。**前端 A1 裁决（daxing 定）**：robots.txt/meta 构建期写死，**不做运行时/构建期 env 联动**（静态产物兜底件不值得引构建变量；权威可配置点是后端中间件）；私有化要放开 = 关后端开关 + 手删 robots.txt/meta（部署文档说明）。**底座中立**：开关/内容全配置注入；新建 httpmw 包不绑应用、为将来其他安全响应头（CSP/X-Frame-Options 等）预留，本片只放 X-Robots-Tag。库 canonical `config.AppConfig` 加 SecuritySection/XRobotsTagSection 保参考结构完整（附「消费方须 SetDefault 保默认安全」责任注释）。**测试**：中间件四态单测（开默认/关/自定义/开但内容空回退）+ 常量值锁定（httpmw/robots_test.go，进默认闸门）；全栈 e2e（robots_integration_test.go 真 buildApp）坐实 401/404 响应均带头 + 内容回退 + 自定义 + toggle 关无头。**daxing 验收**：curl 响应头含 X-Robots-Tag（含 404 全局注入）、/robots.txt 返 Disallow: /、页面 head 含 meta——功能全过。**退回修复（编码缺陷）**：验收发现 robots.txt 套用源码五项中文代码头注释致乱码 → 选项 B 去中文头改纯 ASCII（一行英文注释 + 协议两行），`file` 报 ASCII text、无 BOM、dist 产物正确；index.html 核查 UTF-8 无 BOM、meta 行纯 ASCII（未动）。**规范澄清（PM 记账本）：robots.txt 是机器读的标准协议文件、非源码，不纳入「五项代码头注释」规范。** 无 DDL/错误码/权限码/openapi 端点。**新需求池禁收录条目至此清零；媒体管理条目仍待先摸底再拆。**

## 切片进度

### 已完成
| 任务编号 | 切片 | 备注 |
|---|---|---|
| T-001 | 安全地基（crypto + JWT + Casbin + 跨语言向量） | ✅ |
| T-002 | 认证授权（登录/刷新/登出 + 验证码 + 失败锁定 + Argon2id） | ✅ 至 8b8e7da |
| T-003a | 组织架构（用户 + 部门 + 岗位 + GormUserProvider） | ✅ 至 3faf158 |
| T-003b | RBAC 核心（角色/权限/菜单 + Casbin 联动 + Hashid 收口） | ✅ 至 bd990a7 |
| T-003c | 数据权限（B 档三档 + 通用 DataScope 解析器） | ✅ 至 e167098（RBAC 收官） |
| T-004a | 系统管理（response Registry 接管 + 字典 + 参数 + 操作日志 + 登录日志） | ✅ 至 f1594d0。详见下方记录。 |
| T-004b | 文件管理 + 存储驱动（StorageDriver + LocalDriver + 鉴权下载 + 上传安全） | ✅ 至 448671f。T-004 整块完成。详见下方记录。 |
| T-004c | 渲染收敛（handler 统一 response.Render + errcode 降级纯常量） | ✅ 至 589f9e1。纯重构，对外零行为变化。 |
| T-005 | 配置中心（动态参数缓存 + 热加载 Pub/Sub + GCM 加密 + 迁移执行器） | ✅ 至 303bf1e。详见下方记录。 |
| T-003d | RBAC 鉴权接线修复（消除假 RequirePerm + RegisterRoutes 注入真 enforce） | ✅ 至 813dce2。修 T-003b 接线缺陷。详见下方记录。 |
| T-006 | examples/demo 装配 + 全链路 e2e 跑通（含 #4/#5 缺陷修复） | ✅ 至 813dce2。demo 真·跑通。详见下方记录。 |
| T-007a | admin 前端地基（请求层 + 登录 + 路由/布局 + Pinia + 主题/i18n） | ✅ 至 23ac0a8。浏览器验收通过。详见下方记录。 |
| T-002b | 验证码修复（开源字体可读 + 触发对齐 + 填错刷新） | ✅ 至 d073903。浏览器验收通过。 |
| T-007b | 前端权限与 CRUD（动态路由 + v-permission + x-table + 用户/角色页 + 登出） | ✅ 双推完成（至 78342cf）。浏览器验收通过。阶段一收官。详见下方记录。 |
| T-003e | 入参对外 ID 收口（rbac 全部对外 ID 入参 string/hashid 解码 + 系统性扫尾） | ✅ 至 621f7a8。补 T-003b 入参侧遗漏。详见下方记录。 |
| T-004d | system 对外 ID hashid 化（全套）+ 抽 idcodec 中立包 | ✅ 至 85067fb。补 system 裸 uint64（dict/config/file/日志），rbac↔system 经 idcodec 解耦。详见下方记录。 |
| T-007c | x-table 增强（只读 + 工具栏/行操作插槽 + 列筛选排序）+ 修角色页操作列种子 | ✅ 至 2b0505e。前端迭代批次首片。浏览器验收通过。详见下方记录。 |
| T-004e | DB 唯一键冲突转友好业务错误码 backstop（全 create/update 系统性收口） | ✅ 至 7b2aca4。修 T-007c 暴露的重名报 500 既有缺陷。详见下方记录。 |
| T-007d | 字典/参数管理页（双表联动 + 加密参数安全降级 + 409 友好码端到端回证） | ✅ 至 ac9b3b0。浏览器验收通过。详见下方记录。 |
| T-007e | 操作/登录日志页（x-table 只读真消费 + 详情弹窗 + 清理确认）+ seed 补日志菜单/F 码 | ✅ 至 9e5ebb4。浏览器验收通过 + dept_mgr 正向 enforce 补证。详见下方记录。 |
| T-007f | 文件管理页（x-table 消费 + 鉴权上传/下载 blob 取流 + 软删确认）+ seed 补 sys:file:* 菜单/F 码 | ✅ 至 5c998ba（前置 chore c2a1026）。浏览器验收通过 + enforce 正向证据执行端前置自带。详见下方记录。 |
| T-007g | 菜单树形管理页（el-table 树表 + M/C/F 动态表单 + buildTree 通用树工具防环） | ✅ 至 33679a3。浏览器验收通过 + enforce 前置自带。建 buildTree（T-007h 前置产出）。详见下方记录。 |
| T-007h | dept/post 选择器 + 编辑回填（复用验证 buildTree）+ 岗位管理页 + seed 补 sys:post:* | ✅ 至 2460013。浏览器验收通过（回填回显正确）。B 部分种子最后一块清零。连带修 T-007b 静默清空缺陷。详见下方记录。 |
| T-007i | admin vitest 测试基建 + tree.ts 单测（17 项脏数据/防环断言，收编 T-007g 一次性脚本） | ✅ 至 c2a3d90。pnpm test 17 PASS + pnpm build 通过。前端铺厚批次收官。详见下方记录。 |
| T-005b-1+2 | 菜单 CUD 联动 Casbin policy 重载（★安全·删 F 权限真收回）+ Claims.GetSubject 修 uploader 恒空（★装配静默失败） | ✅ 至 f42dc80。**后端债批次首切**。policy 生命周期对照 + enforce 即时翻转坐实。详见下方记录。 |
| T-005b-4 | system/日志列表查询能力（search/filter/sort + 时间范围 + dict_data 真分页）+ 操作人可读化（operator/uploader→用户名，B 出参 JOIN） | ✅ 至 4e16f91。**后端债批次第二片**。排序强白名单防注入 + 可读化批量 IN + dict_data 真分页。openapi v0.11.0。详见下方记录。 |
| T-008a | 用户改密码 + status 假能力修复（编辑弹窗静默吞状态→行操作按钮接独立端点） | ✅ 至 c2e620a。**用户/角色管理补缺片首片**（纯前端）。修 T-007h §8-3 假能力。详见下方记录。 |
| T-008b | 用户分角色（多选弹窗回填+全量覆写）+ 列表角色列（批量回填非 N+1） | ✅ 至 1d4b9aa。**补缺片第二片**（后端搭车补回填 A 方案）。回填/覆写不误删/Casbin g 联动坐实；editor 空菜单核实为 seed 未授可见菜单非 bug。openapi v0.12.0。详见下方记录。 |
| T-008c | 角色授权树（el-tree check-strictly 勾选授菜单 + 全量覆写不丢权限）+ GET /:id 搭车补 menu_ids 回填 | ✅ 至 f88d3c7。**补缺片压轴·权限体系两段闭环**。check-strictly 选型（恒等往返）+ 写路径/policy 零改（T-003b 已联动）+ editor 补验兑现。大集合改一处不误删真界面坐实。openapi v0.13.0。详见下方记录。 |
| T-005b-3 | 配置加密参数写链路（新建可加密 + 编辑指针三态不破坏密文 + 解禁前端编辑）+ seed 加密样例 | ✅ 至 d89e3e2。**后端债批次第三片**。ConfigService 经 SetGCMKey 接 GCM；UpdateConfigInput ConfigValue *string 三态（取 DB 现有 is_encrypted 判据，nil=保持）。解密往返三态 + 全栈 enforce + seed 幂等坐实。openapi v0.14.0。T-007d 加密行编辑禁用至此放开。详见下方记录。 |
| T-009a | 后台禁搜索引擎收录（noindex 三件套：X-Robots-Tag 中间件参数化 + robots.txt + index.html meta） | ✅ feature a4b3fc8（HEAD=报告归档提交）。**新需求池首条**。新建 httpmw 包全局中间件无差别注入（含 401/404）；默认安全 SetDefault 兜底；前端兜底件构建期固定（A1 裁决）。中间件四态单测 + 全栈 e2e 坐实。退回修复 robots.txt 编码（去中文头改纯 ASCII）。无 openapi/DDL/错误码。详见下方记录。 |

**T-001~T-003 收尾记录**（详见此前版本，要点）
- T-001 crypto/auth/rbac 三核心 + crypto-vectors KAT + DI；集成测试真后端。
- T-002 Argon2id+DummyVerify 防枚举；验证码纯标准库自绘零字体；失败锁定固定窗口；编排防枚举；令牌复用 T-001。
- T-003a 组织三表 + GormUserProvider；去包级前缀改 NamingStrategy（两前缀不串表）；部门树防环；password_hash 不泄漏断言。
- T-003b 角色/菜单(menu_type)/Casbin 联动；model.conf 改 perm code 精确匹配（破坏性，同步 spec）；超管短路服务端可信；授权变更事务内回滚保一致；Hashid 入出参闭环（装配须注入 hasher）；RequirePerm 28 路由全覆盖。
- T-003c data_scope 三档 + 中立 DataScope 解析器 + ApplyScope(字段名调用方传入)；失败一律收紧 5 处 WHERE 1=0 有断言；范围来源服务端；叠加 enforce 不绕过。

**T-004a 收尾记录（PM 评审定档）**
- response 模块：Registry（Register/Lookup/冲突 fail-fast）+ Success/Fail/FailMsg/Page 渲染 + i18n 默认表。
- **错误码迁移现状=渐进中间态：两套渲染并存**（T-004c 已收敛，见下）。回归硬断言：migration_test.go 32 个既有码经 AllSpecs→Register→Lookup 逐项断言 32/32 匹配。既有码零变更，PHP parity 安全。
- 字典（sys_dict_type/data）+ 参数（sys_config）CRUD，纯 DB，缓存/热加载留 T-005；sys_config 敏感参数加密存储留 T-005。
- 操作日志 sys_oper_log + 中间件自动采集（异步、写失败不阻塞、排除列表配置注入）+ 脱敏（password/captcha_code/token/secret→***）。
- 登录日志 sys_login_log：LoginLogger 接口定义在 auth 包（auth 不依赖 DB），GormLoginLogger 实现在 system 侧由装配注入。
- 日志列表/清理挂 RequirePerm，清理独立权限码。errcode offset +60~+63。openapi v0.6.0。commit f63c1ae+f1594d0。

**T-004b 收尾记录（PM 评审定档，T-004 整块完成）**
- 存储驱动 server/drivers/storage：StorageDriver 接口 + LocalDriver；云驱动为扩展点，底座不引任何云 SDK；storage_type 字段留位。
- 路径穿越防护（头号）：safePath = filepath.Clean + 根目录前缀校验 + 拒 ../绝对路径/控制字符；单测 5 恶意用例 + 集成测试真盘 os.Stat 双层闭合。
- 上传安全四件套（配置注入）：大小上限/扩展名白名单/文件名消毒/Content-Type 一致校验，各有断言。
- key 策略：yyyy/MM/dd/uuid.ext（日期分目录 + uuid 防覆盖/防枚举），原名存 sys_file.original_name。
- 鉴权下载：经 RequirePerm 流式返回，根目录不挂静态路由；URL() 返回下载接口引用不泄漏文件系统路径。
- 删除：软删元信息 + 物理异步清理。sys_file 元信息表 DDL。错误码 +70~75，回归快照 38/38。openapi v0.7.0。commit 8c9bd0a+448671f。

**T-004c 收尾记录（PM 评审定档，渲染债清）**
- 纯重构、对外零行为变化：消除 T-004a 遗留两套渲染并存。
- errcode.Error 剥离自渲染/HTTP 语义：删 HTTP/Message 字段，只留 Code + GetCode + Error()；grep 无 .HTTP/.Message 引用。
- response.Registry 成为 code→HTTP→i18n 唯一权威 + 唯一渲染路径：全 handler 走 response.OK/ErrResp/BadReq，全中间件走 response.AbortErr。
- 零行为变化双证：migration 快照 38/38 + T-001~T-004b 全部旧测试全绿。openapi 未升版。commit 至 589f9e1。

**T-005 收尾记录（PM 评审定档，后端底座收官）**
- 边界：聚焦动态参数缓存+热加载+加密；静态启动配置仍归 viper。
- GCM 加密：crypto 新增 EncryptGCM/DecryptGCM，与 T-001 CBC 信封并存互不影响；主密钥配置注入 fail-fast；每次随机 12B nonce + 篡改检测。存储格式 base64(nonce12||ct||tag)，自描述、PHP 可解，入 spec。
- sys_config 增列 is_encrypted：写加密落库密文、读自动解密；加密闭环三道：存密文 + 授权路径解密 + 列表/详情 maskEncrypted() 返回 ******。
- ConfigCenter + 缓存层：ConfigCache 接口（RedisConfigCache + 内存假实现）；命中→回源 DB→回填；写后失效。
- 热加载 Redis Pub/Sub：跨实例集成测试真 Valkey 跑通；单实例无 Redis 退化本地刷新。
- SQL 迁移执行器 migrator.go：读 spec/migrations、字典序排序、{{TABLE_PREFIX}} 替换、按序执行、sys_migration 记录版本+校验和、幂等跳过、失败中止。
- errcode +80~81，回归快照 40/40。openapi v0.8.0。commit 1c339e2+303bf1e。

**T-006 + T-003d 收尾记录（PM 评审定档，阶段一后端可信收官）**
- 经过：上个会话提交 T-006 装配（d601de6）并自行把 PROJECT_STATUS 改写"完成"——两个必补项未落地未推送未经 PM 批准。本会话接续核出抢跑要求补齐；补的过程 demo e2e 逐层逼出三个潜伏缺陷。**教训：账本完成判定权在 PM，执行端不得自标完成/自行双推。**
- demo = 装配样例 + 回归环境：迁移执行器一键建表 → 装配五大块 → 装配自检（关键依赖非 nil fail-fast）→ 种子数据（超管/3角色/3部门/4用户/菜单权限+Casbin ReloadAll，幂等）。
- 种子密码去硬编码：seedPassword() 零明文默认值、缺失 fail-fast、全来自 cfg.SeedPasswords；env replacer 修复。
- #4 casbin 版本冲突修复：gorm-adapter/v3 钉 v3.38.0，casbin v2 路径零影响。
- #5 迁移器静默不建表修复（T-005 恶性 bug）：splitStatements 把"注释头+CREATE"切成纯注释跳过。修：stripLineComments 逐行剥 -- 注释再 Split(;)。回归断言真库建表数=16、15 表 HasTable、sys_migration 记录数==18。
- #3 / T-003d 鉴权接线修复（修 T-003b 缺陷）：删除式消除假 RequirePerm；RegisterRoutes 注入鉴权（rbac 用 *AuthzEnforcer、system 用 PermGuard 接口零耦合无环）；路由级闭包捕获 code 真 enforce；超管短路保留。**46 条受保护路由全真 enforce**（rbac 26 + system 20）。RegisterRoutes 签名变更属对外契约变更。
- 验证三重证据：grep 无假版 + 单测 + demo e2e 8 步 ALL PASSED。commit 99716da(T-006)+813dce2(T-003d)。

**T-007a 收尾记录（PM 评审定档，首个可运行界面，浏览器验收通过）**
- admin/（Vite+Vue3+TS+Element Plus），自建脚手架不引第三方 admin 模板；仅开源素材。
- 请求层 src/request：axios 拦截器注入 JWT、解统一包络、错误码按 HTTP 分流（401 刷新一次/403/423/其余 toast）；关键陷阱：401 刷新只对非 /auth/* 端点触发；hashid 字符串 ID 原样透传不解码。
- 登录页 + 路由守卫 + 响应式布局 + Pinia user/app store；令牌存 localStorage（key 前缀 bxap_）；暗色+主题色骨架；i18n 框架。浏览器验收通过。commit 23ac0a8。

**T-002b 收尾记录（PM 评审定档，登录链路完整可用，浏览器验收通过）**
- 验证码可读：引 Go Mono Bold 开源字体（BSD-3-Clause，记 CREDITS）；字符集排除易混。抗自动化：旋转+波浪扭曲+干扰线+噪点。
- 触发对齐：新增 POST /auth/precheck（{username}→{captcha_required}）；后端独立判定强制校验防绕过。
- 填错刷新：验证码一次性消费（GetDel），登录失败自动换新。openapi v0.8.1。commit d073903。
- 待办（后期评估）：图形验证码可被 AI/OCR 识别是根本局限，真实抗自动化靠失败锁定；将来正路是行为验证码，涉第三方与底座中立性，单独评估。

**T-007b 收尾记录（PM 评审定档，阶段一收官，浏览器验收通过）**
- 动态路由：登录拉 menus → 过滤 M/C 生成路由+侧边栏；组件路径经 import.meta.glob 映射 .vue，缺失降级占位。
- v-permission 指令：拉 perms 存 store，无权码隐藏元素，支持单码/数组，超管全显。
- x-table 配置化 CRUD（最小版）：列+分页+搜索+增/编辑弹窗+删除确认；对接统一包络/错误码；hashid 透传；行操作挂 v-permission。
- 样例页：用户管理 + 角色管理。登出：调 /auth/logout + 清令牌 + 跳登录。
- 浏览器验收暴露并修两 bug：① 刷新 /sys/* 404——dev 代理 bypass（Accept:text/html 返回 SPA 只代理 API XHR）。② editor 空菜单卡登录页——null 兜 []、守卫 catch 仅令牌失效才登出否则放行 dashboard。commit 78342cf。

**T-003e 收尾记录（PM 评审定档，入参侧 hashid 收口，API 验收通过）**
- 补 T-003b 遗漏：入参裸 uint64 → 选择器无法回填。本片 rbac 全部"引用对外 ID 的入参"统一在 handler/DTO 边界 string/hashid 解码，service 层签名保持 uint64。
- 收口 11 项（系统性 grep 全扫，扫出漏列的 UserListQuery.dept_id）。decode.go 单一解码出入口承载 空→nil/根=0/移动三态/列表语义。
- 非法/伪造 hashid 一律 400（复用 ErrInvalidID）；防探测；service Input 对外 ID 字段标签降级 "-" 杜绝误绑定。
- openapi v0.8.1→v0.9.0：入参 schema integer→string，并对齐出参 schema hashid（修 T-003b 文档债）。装配/RegisterRoutes 签名零变更。commit 621f7a8。

**T-004d 收尾记录（PM 评审定档，system 全链路 hashid + idcodec 解耦，API 验收通过）**
- 全套粒度：dict/config/file/oper_log/login_log 六实体对外 ID 全收口，"全链路 grep 零裸 uint64 主键"明线。
- 前置·抽 idcodec 中立包：hasher 从 rbac 搬至 server/idcodec，rbac/system 共用互不 import。go list -deps 六条实证无环、零反向耦合。
- system 出参 hashid：6 实体 id 经 encoder（反射+json 重映射）；config 脱敏不被破坏。path :id 收口 6 处 decodePathID。
- openapi v0.9.0→v0.10.0：补全 dict/config item 路径；file :id hashid 一致。对外契约变更：system.NewHandler/NewFileHandler 新增 hasher 注入。commit 85067fb。

**T-007c 收尾记录（PM 评审定档，x-table 加厚为承重件，浏览器验收通过，前端迭代批次首片）**
- x-table 增强（纯前端）：只读模式 + 工具栏插槽(#toolbar) + 行操作(XAction+#row-actions，可挂 v-permission) + 列筛选(表头 popover) + 服务端列排序(sortable='custom'，后端未就绪降级)。配置 schema 全可选、缺省即 T-007b 现状。
- v-permission 回炉（PM 要求）：空绑定值从"显示"翻转为"隐藏 + dev 告警"，让"忘记传码"与"故意公开"语法上长得不一样。
- 验收五道弯：① 新增用户报 500 → 重名撞唯一键（记 T-004e）。② 角色页操作列空 → 种子缺 sys:role:update/delete（补）。③ 系统性扫种子扫出 27 缺口，PM 裁定补 A 部分 15 码、否决 B 部分（挂各切片）。commit 2b0505e。

**T-004e 收尾记录（PM 评审定档，重名 500 修复，真库验收通过）**
- 重大发现：5 个"已存在"友好码早已注册，真缺口是 1062 转换链路从未接到这些码——service 只在应用层预检路径返友好码，预检漏判场景（软删占位/并发/update 改名）下 1062 直接冒泡。本片补转换 backstop，不新增码、openapi 不升版。
- 单一出入口 server/dberr 中立包（同 idcodec 模式）：IsDuplicate(err) = gorm.ErrDuplicatedKey || MySQLError{1062}。各 service 写边界返裸 errs.ErrXxxExists 不被 %w 包裹。收口 5 实体 10 写站点。
- 真跑两次拦下两个测试侧 bug：① config 旧"子串黑名单"误判 "1062"（合法码 errcode:11062 含子串）→ 改精确等值。② role 1054 测试 setup 漏载 data_scope ALTER。commit 7b2aca4。

**T-007d 收尾记录（PM 评审定档，前端迭代批次第二片，浏览器验收通过）**
- 定位：T-007c（x-table 增强）+ T-004e（唯一键友好码）首个端到端回证消费页。纯前端，未改后端一行；5 代码文件（2 新 api + 2 新 view + XTable 加固）。
- **字典管理页（双表联动）**：类型(左)↔数据(右)，行操作「字典数据」选中类型→ :key 重建右表；数据列表数组响应经适配层包装分页包络本地切页（数据全量未伪造）；dict_type 由适配层注入不进表单；未选类型空态不发请求。类型/数据增改删浏览器逐点闭环（吃 T-007c「不止看按钮、要真点」教训）。
- **参数管理页（sys_config）**：明文行 CRUD 闭环；加密行(is_encrypted)列表恒显 ******（后端 maskEncrypted 落地）；**加密行编辑按钮禁用 + tooltip 诚实归因「加密参数暂不支持界面编辑（后端缺更新加密链路，已上报）」**——§8-1 安全降级落点，堵住会损数据的写路径入口（留空清密文/重填存明文致解密必败）；明文行编辑 config_key 禁改。
- **两条 409 友好码端到端回证（本片核心使命）**：重复 dict_type → 11060「字典类型已存在」、重复 config_key → 11062「参数键已存在」，均干净 409 非 500，弹窗保留供修正。T-004e 在前端闭环坐实。
- **XTable 加固（承重件，+30/-11）**：create/update/remove/list 失败路径 try-catch 兜底（请求层已 toast，失败 return 保留弹窗供修正、不冒未处理 rejection——重复键 409 场景 Console 干净）；删除取消/失败各自 return。成功路径零变化，用户/角色页回归（重名用户失败弹窗保留 + Console 干净）实证零回归。@updated 头注释已补。
- 路由/菜单前端零改动：seed 已有 C 菜单 /sys/dict、/sys/config，动态路由 glob 自动映射新 view。F 码 seed 齐（dict/config 各 list/create/update/delete 4 码），未复现 T-007c 空操作列坑。
- **读源码兜住任务书两处 PM 假设**（以源码为准）：① 加密「留空保持/重填替换」语义后端根本不支持（CreateConfigInput 无 is_encrypted、Update 全量明文覆写不再加密）→ 安全降级禁用而非硬铺会损数据的假 UI；② dict/config 列表仅 page/page_size 无 search/filter/sort（dict_data 连分页都无、返全量数组）→ 列筛选/排序开关已挂、参数并入 query 透传，后端补参后前端零改动生效。
- 字段名以源码为准：dict 用 name/label/value/sort（无 dict_ 前缀）、config 用 config_key/config_value/name/is_encrypted（无 config_type）。
- 安全自查：加密参数前端无明文展示/打印/回填（编辑禁用使回填路径不可达）；行操作 v-permission 传真实码无空值；hashid 透传不解码；editor 调 dict 接口 403 实证后端 enforce 边界，前端不做伪安全。
- 双推 Gitee+GitHub 一致（ls-remote 实测三处 HEAD=ac9b3b0 无单推）。commit ac9b3b0（6 files，+527/-12）。

**T-007e 收尾记录（PM 评审定档，前端迭代批次第三片，浏览器验收通过）**
- 定位：x-table 只读模式（T-007c 产出）首个真消费页 + 偿还 T-007c §8 B 部分 operlog/loginlog 种子缺口。纯前端消费 + demo seed 增量，**未改任何后端 Go 业务逻辑、未升 openapi（仍 v0.10.0）、未新增错误码**。6 files +290（2 新 api + 2 新 view + seed.go/menuIcon.ts 增量）；x-table 本片零改动。
- **前置门禁（吃 T-007d 教训）落地**：8 项后端源码核实全部带出处，反"典型 admin"假设到位——无 `GET /:id` 详情端点（改行内数据弹窗，零额外请求）、无 DTO 直接 c.Query、清理无入参（固定删 3 个月前，handler 硬编码 `AddDate(0,-3,0)`）、无多选语义（故未触发"x-table 多选回 PM"条款，未擅扩 x-table）。
- **操作日志页**（sys_oper_log）：x-table 只读列表 + 分页 + 详情弹窗（行内数据：req_summary/UA/耗时/结果码）+ 工具栏清理。字段以源码为准：operator/method/path/perm_code/ip/user_agent/req_summary/result_code/latency_ms/created_at（无 oper_ 前缀、无软删/updated_at）。
- **登录日志页**（sys_login_log）：x-table 只读列表 + 分页 + 工具栏清理（无行操作列）。字段：username/ip/user_agent/success/reason/created_at。
- **查询能力诚实降级**（同 T-007d 做法）：列表仅 page/page_size + 单字段精确过滤（oper=operator / login=username）；path 筛选、created_at 排序、时间范围均后端未就绪，filterable/sortable 开关已挂、参数并入 query 透传，后端补参后零改动生效。不伪造前端假过滤。
- **清理破坏性操作**：工具栏按钮 + 二次确认明示"删 3 个月前/不可恢复" + 独立清理权限码（非复用 list）+ 成功显示删除条数 + 刷新；取消/失败各自 return 兜底（T-007d XTable 加固模式），Console 无未处理 rejection。
- **seed 增量（幂等）**：+2 C 菜单（/sys/operlog sort7、/sys/loginlog sort8，挂 sysDir 下）+4 F 码（sys:operlog:list/clean、sys:loginlog:list/clean）。超管/dept_mgr 经 allMenuIDs 全量授权自动获授；editor 仅 sys:user:list 天然 403。真库连跑两遍幂等实证：37 菜单/75 授权 → 43/87 → 43/87 不变。
- **enforce 三段链路坐实（本片头号正确性点，PM blocker 补证）**：① F 行入 menu 表（+4）② role_menu 授超管/dept_mgr（+12）③ **ReloadAll 从 role_menu(perm_code≠'') 重建 p 规则、调用点在授权之后（seed.go:181，授权 159-176）**。补证用 **policy 删除-再生对照**（casbin_rule 72 → 手删 8 条新码 p 规则 → 64 → 重启 ReloadAll → 72 全再生），比快照存在更强。**dept_mgr 正向 enforce**（非超管、不短路、走真 Casbin）四端点全 200 ↔ editor 全 403，正反对照完整——纠正了原报告"超管 200/editor 403"均为"天生该绿"的假绿（超管走短路不过 policy、editor 天生无码）。
- **浏览器验收三证**：① 操作日志页 47 条真数据，列表内中间件采集的记录可视复现 enforce 对照（操作人 3=dept_mgr DELETE 成功 / 操作人 2=editor DELETE 403 / 操作人 1=admin 成功）——产品内被动记录的第三个独立佐证。② 详情弹窗字段完整（DELETE 无 body 显"(空)"为正确渲染）。③ 登录日志 41 条，dept_mgr 真人登录记录在列首（真人侧正向对照达成），username 列显真实名印证 §8-1。
- 安全自查：hashid 透传不解码（清理无 id 入参）；v-permission clean×2 挂真实码无空值；脱敏字段后端 Sanitize 产物前端忠实渲染、不还原不打印；enforce 边界 editor 403 实证后端真把关、前端隐藏仅 UX。
- 双推 Gitee+GitHub 一致（ls-remote 实测三处 HEAD=9e5ebb4 无单推）。commit 9e5ebb4（6 files，+290）。

**T-007f 收尾记录（PM 评审定档，前端迭代批次第四片，浏览器验收通过）**
- 定位：x-table 消费 + **前端迭代批次首个真实写动作页（上传/下载/删除）** + 偿还 T-007c §8 B 部分 sys:file:* 种子缺口。纯前端消费 + 请求层 blob 透传 + demo seed 增量，**未改任何后端 Go 业务逻辑、未升 openapi（仍 v0.10.0）、未新增错误码**。4 files +228/-1（1 新 api + 1 新 view + request/index.ts blob 透传 + seed.go）；x-table 本片零改动；前置 chore c2a1026（.gitignore +1 行）。
- **前置门禁（吃 T-007d/T-007e 教训）落地**：9 项后端源码核实全部带出处，反"典型 admin 文件页"假设到位——字段是 `mime` 非 content_type（无 file_ 前缀）；下载**无 tokenized URL 机制**（GET /sys/files/:id/download 挂 RequirePerm + 须 Authorization 头，流式返回）→ 逼出 **axios blob 带 JWT 取流 + 客户端落盘，禁裸 `<a href>`**（裸 href 带不上 token）；上传单文件 multipart 字段名 `file`；删除单条软删无多选 → 均未触发"x-table 多选/扩展回 PM"条款，x-table 零扩展。
- **文件管理页**（sys_file）：x-table 只读基底 + 分页 + 工具栏 el-upload 上传 + 行操作下载/删除。列：original_name/类型 ext/mime/size/uploader/created_at；**storage_key 出参有但 UI 不展示**（§5 PM 立场：相对 key yyyy/MM/dd/uuid.ext 非绝对路径，最小暴露，后端 mask 归 T-004b 加固不阻塞）。
- **上传**（POST /sys/files，sys:file:upload）：工具栏 el-upload（v-permission 挂整体）；before-upload 客户端预校验（≤10MB + 9 类扩展名 jpg/jpeg/png/gif/pdf/docx/xlsx/zip/txt，**仅 UX 提前拦，与后端默认值对齐**）；http-request 走统一请求层；成功 toast+刷新。**上传安全边界在后端**：负例实证 11MB→413/11070、.sh→415/11071（四件套真把关）。
- **下载**（GET /sys/files/:id/download，sys:file:download）：**axios responseType blob 带 Authorization 取流 → objectURL + a.download=original_name 落盘 → revoke**；无裸 href；401 刷新链路同样生效。请求层 index.ts 改动（+17/-1）：blob 响应透传不走包络解包 + blob 错误体解析友好 message——鉴权下载必要前置（原拦截器把一切当 JSON 包络解包、blob 会被解成 undefined），最小化实现、普通 JSON 路径零行为变化（超"1api+1view"清单但属第 0 节#5 预期的必要改动，非擅扩 scope，PM 接受）。
- **删除**（DELETE /sys/files/:id，sys:file:delete）：XAction 二次确认（明示"软删 + 物理异步清理不可恢复"）+ refresh:true；取消/失败各自 return 兜底，Console 无未处理 rejection。
- **查询能力诚实降级**（同 T-007d/T-007e 做法）：列表仅 page/page_size + uploader 精确过滤；original_name 筛选、created_at 排序后端未就绪，filterable/sortable 开关已挂、参数并入 query 透传，后端补参后零改动生效。
- **seed 增量（幂等）**：+1 C 菜单（/sys/file sort9 挂 sysDir、图标 folder menuIcon.ts 已有映射无需改）+4 F 码（sys:file:list/upload/download/delete）。超管/dept_mgr 经 allMenuIDs 全量授权；editor 天然 403。真库连跑两遍幂等实证：基线 43 菜单/87 授权/72 policy → 48/97/80 → 48/97/80 不变。
- **enforce 正向证据（执行端前置自带，吃 T-007e blocker 教训，本片不留到放行前补）**：① 4 条 F 行入 menu 表（48 实查）② role_menu 授超管/dept_mgr（87→97）③ ReloadAll 调用点 seed.go:189 在授权循环 166-185 之后。**policy 删除-再生对照**（casbin_rule 80 → 手删 8 条 sys:file: p 规则 → 72 → 重启 ReloadAll → 80 全再生）。**dept_mgr 正向**（非超管/不短路/走真 Casbin）list/upload/download/delete **四端点含写动作全 200**（download 内容 diff 逐字节一致）↔ **editor 同四端点全 403**。超管 200 不计为证据（T-007e 口径）。
- **浏览器验收**（逐项真点 + 写动作三连）：① 上传正常 pdf（6.55MB）入列 + 负例 >10MB 前端中文 toast「文件大小不能超过 10MB」拦截。② 下载 blob 带 JWT 落盘、中文文件名 original_name 正确。③ **删除用干净文件复验**（早前"删存量死链行 1wR9wYV8 toast 成功但列表行未消失"经排查定性为 §8-5 物理文件已丢的边缘表现、非 T-007f bug）：删干净 pdf 行 → 行消失、共 2 条→1 条，refresh 真生效——补齐"成功提示 ≠ 状态变更"的最后一幕。④ storage_key 不裸露。⑤ 上传人列空（印证 §8-1 后端缺陷非前端 bug）。
- 安全自查：hashid 透传不解码（伪造 id 后端 decodePathID 统一 400）；鉴权下载经统一 axios 带 JWT、无裸 href、objectURL 用后即 revoke；上传安全边界在后端（前端预校验仅 UX、负例实证后端独立把关）；v-permission 全真实码无空值；删除二次确认明示破坏性。
- **挖出 T-004b 两潜伏缺陷（如实暴露未擅改，并入 T-005b）**：①【优先】uploader 恒空——handler 类型断言 `interface{ GetSubject() string }`，但 auth.Claims 只有 Subject 字段无 GetSubject() 方法（装配期类型断言静默失败、返零值，同 T-006 "编译过单测绿装配才暴露"类病），后果上传人列恒空 + uploader 过滤实际不可用（比 operlog 存内部 ID 更甚，连 ID 都没存上），修法极小（Claims 加 GetSubject() 零破坏）；②file 错误文案缺 6 键 → 下载/删除后端错误透传时返裸 i18n key（验收下载死链时肉眼撞见 `sys.storage_failed`），+6 行文案即点亮。
- 双推 Gitee+GitHub 一致（ls-remote 实测三处 HEAD=5c998ba 无单推）。commit 5c998ba（feature，4 files +228/-1）+ c2a1026（chore .gitignore）。

**T-007g 收尾记录（PM 评审定档，前端迭代批次第五片，浏览器验收通过）**
- 定位：**前端迭代批次第五片 + 建 buildTree 通用树工具（T-007h dept/post 选择器前置依赖产出）**。纯前端消费，3 新增文件 +576（utils/tree.ts + api/menu.ts + views/sys/menu/index.vue）；**seed/后端 Go/openapi（仍 v0.10.0）/错误码全零改动**（核实#7：菜单管理 C 页 + 4 F 码 seed 早已存在，是其他页能显示的前提）。
- **前置门禁（吃 T-007d/e/f 教训，本片新增"树形数据形态"维度）落地**：9 项源码核实全部带出处，三项关键结论明确——
  - **#2 树形形态**：后端 `GET /sys/menus/tree` 服务端 buildMenuTree **已返嵌套树**（无任何查询参数）→ 前端表格直接消费、**零再组装、不引顺序分歧**。buildTree 没因此变废：真消费点 = 父级选择器"flatten→subtreeIds 排除自己及子孙→重建"管线，且为 T-007h 备好通用扁平→树能力。
  - **#4 menu_type 约束**：validateMenuType——F perm_code 必填（11041）+ 全局唯一（11046）；M/C perm_code 必空（11042）；**父子类型后端无任何约束**（不检查 parent.MenuType）；类型可互转；Update 全量覆写（隐藏字段保留原值，唯 perm_code 在 M/C 强制空串）。前端表单动态显隐镜像后端、不做伪安全。
  - **#6 排序**：sort 仅经 create/update 单节点改，**无批量重排接口** → 本片**不做拖拽**（按任务书不擅造接口），sort 经表单单节点改。
- **buildTree 通用树工具（关键产出，落 utils/tree.ts 独立模块）**：buildTree（扁平→嵌套）+ flattenTree（嵌套→扁平，环状输入 visited 防护）+ subtreeIds（自身+子孙 id 集）。**纯函数、键名参数化（idKey/parentKey/childrenKey 可配，不绑 menu 专属字段）→ T-007h dept 选择器直接复用**。防环/防孤儿：parent 指向不存在/自引用/互引环/三元环全降级为根 + dev 告警，不死循环不白屏。**21 项脏数据自测 ALL PASSED**（esbuild 转译→node 真跑；脚本一次性未入仓，T-007i 收编 vitest）。
- **菜单管理页**（views/sys/menu/index.vue）：el-table 内建 tree（row-key/children，直接消费后端嵌套树）+ 工具栏新增/展开折叠 + 行操作（新增子级/编辑/删除）+ M/C/F 三类型动态表单（F 显 perm_code 必填、隐 path/component/icon/visible；M 隐组件；C 全显）+ 父级树选择器编辑时经 subtreeIds 排除自己及子孙（buildTree 真消费点）+ 删除二次确认（措辞按后端拒删行为）。
- **后端约束行为真跑实证**（dept_mgr，全友好码非 500）：删有子节点→409/11043；建 F 缺 perm_code→400/11041；建 M 带 perm_code→400/11042；建 F 撞已存在码→409/11046；伪造 parent hashid→400/11045；移动成环/自挂自→400/11035（文案"父部门"系复用 ErrInvalidParentDept，§8-2）。
- **enforce 正向证据（执行端前置自带）**：dept_mgr 经 allMenuIDs 全量授 sys:menu:*（seed.go:182-185，非超管/不短路/走真 Casbin）：tree/create/update/delete **含写动作全 200** ↔ editor（仅 sys:user:list）同四端点全 **403/11009**。**policy 80→80→80**（建临时 F→删，policy 不变）——此处"不变"既证本片零 policy 增量诉求，**也正面暴露 §8-1 缺陷：菜单 CUD 不触发 policy 重载**。超管 200 不计为证据（T-007e 口径）。测试产物清理、demo_sys_menu 活跃行复原 48。
- **浏览器验收**：① 树形层级正确（系统管理 M > 9 个 C > 各 F，缩进/展开/折叠正常，48 节点无错位丢失重复）。② 真点新增三类型动态表单字段显隐正确。③ 真点编辑移动节点选择器排除自身子孙（subtreeIds 真消费）。④ 删有子节点目录友好提示「菜单下有子节点」非 500。⑤ editor 菜单不可见。验收用自建临时节点试增删改（避开现役菜单，§8-1 风险）。
- 安全自查：hashid 透传不解码（伪造 id 后端 400/11045）；buildTree 防环兜底（前端选择器排除仅 UX 预拦，后端 ancestors 校验仍是移动防环权威）；v-permission 全真实码（sys:menu:create/update/delete 与 handler 逐字一致）；menu_type 约束不做伪安全（前端镜像后端、editor 403 实证后端真把关）；删除二次确认明示破坏性。
- **挖出 T-003b 缺陷（提级，非中性观察）**：§8-1 菜单 CUD 不触发 Casbin policy 重载——MenuService 无 policySync，改/删 menu_type=F 节点的 perm_code 后 casbin p 规则滞留至下次授权变更/重启。**安全后果**：删 F 节点（意图收回权限）但旧 perm_code 的 p 规则仍在→已授角色仍 enforce 通过、权限收不回；或改 perm_code 致新码失联。属 T-003b 既有行为非本片引入，前端已在验收项警示"用临时节点"。**提级 T-005b 优先·安全相关项**（修法：菜单写路径接 policy 重载联动或文档化 + 手动重载入口）。
- 双推 Gitee+GitHub 一致（ls-remote 实测三处 HEAD=33679a3 无单推）。commit 33679a3（feature，3 files +576）。

**T-007h 收尾记录（PM 评审定档，前端迭代批次第六片，浏览器验收通过）**
- 定位：**buildTree 首个复用者（验证 T-007g 中立工具产出）+ dept/post 选择器 + 编辑回填（T-003e 入参 hashid 收口的兑现验收）+ 岗位管理页（偿还 B 部分种子最后一块 sys:post:*）**。纯前端消费 + demo seed 增量，后端 Go 业务零改动、openapi 不升版（仍 v0.10.0）、错误码零新增、**tree.ts 零触碰**。11 files +372/-8（4+1 新增：api/dept.ts + api/post.ts + selectors/DeptTreeSelect.vue + selectors/PostSelect.vue + views/sys/post/index.vue；6 修改：x-table types.ts/XTable.vue + api/user.ts + views/sys/user/index.vue + menuIcon.ts + seed.go）。
- **前置门禁（8 项源码核实带出处）落地**，三项关键结论：
  - **#2 buildTree 复用可行性（中立工具兑现）**：后端 `GET /sys/depts/tree` 已返嵌套树（同菜单口径）→ 本片选择器直接消费、**buildTree/flattenTree/subtreeIds 实际零调用**。**这不是 T-007g 参数化缺口而是本片场景不需要**（强行 flatten→rebuild 是伪消费；subtreeIds 防自挂的真消费点=部门管理页"移动部门"场景，不在本片 scope）。**参数化够用性已逐字段验证**：dept 出参 idKey/parentKey/childrenKey/根判定（parent_id=null）与 buildTree 默认全对得上 → 将来部门管理页可零参数复用菜单页同构管线。**tree.ts 零改动、签名零触碰**（中立性铁律遵守）。
  - **#5 回填语义（T-003e 兑现·头号正确性点）**：详情 GET /sys/users/:id（sys:user:get）出参 dept_id hashid|null、posts 完整岗位对象数组（id hashid，仅非空出现/omitempty）；入参（T-003e 已收口）dept_id 空串=清空、post_ids 缺省=不变/[]=清空/数组=覆写。回填=出参塞 v-model（null→''、posts→map id），提交=hashid 原样回传不解码。
  - **#8 用户↔岗位多对多**：junction sys_user_post（复合 PK，无软删）；岗位 Delete 不检查用户关联（软删放行、junction 残留但不再生效）→ 删除确认文案据此如实措辞。
- **dept 树选择器**（DeptTreeSelect.vue）：el-tree-select 直接消费后端嵌套树，check-strictly + clearable，v-model ''/hashid 对齐 decodeOptionalID；停用部门如实标注不伪造约束。
- **post 多选器**（PostSelect.vue）：el-select multiple 消费 listAllPosts（循环翻页取齐 total，不静默截断；单页 100 为后端硬限）；v-model 恒数组配合全量覆写语义自洽。
- **岗位管理页**（views/sys/post/index.vue）：x-table CRUD（code/name/sort/status），permPrefix sys:post；列表无 search/sort 诚实降级；delConfirm 按真实删除行为措辞；重名 code 走 T-004e 兜底 409/11034 端到端再回证。
- **XTable 三项中立扩展（必要前置非擅扩，同 T-007f blob 透传先例）**：① XField.type 增 'slot'（业务页经 #field-<prop> 作用域插槽提供自定义控件）② XApi 增可选 get（编辑回填详情来源）③ XTableConfig 增可选 delConfirm（删除文案按资源真实行为覆写）。三者全可选、缺省=T-007c 现状、其他 XTable 页（字典/参数/日志/文件/角色）零回归面（daxing 已回归扫验正常）。openEdit 异步化：有 api.get 先拉详情填表，**拉失败不开弹窗**（防残缺回填被全量覆写静默清空——回填防误清设计）。
- **连带修复 T-007b 静默资料损坏缺陷（§8-2，"装配/真界面操作才暴露"模式又一例）**：T-007b 起用户编辑表单**无 dept_id 字段** → 后端 update 空串=清空语义下，**每次界面编辑任何用户都把其部门静默清成 NULL**（不报错、toast 成功、资料已损）。demo 未暴露因从未用界面编辑过有部门用户。本片回填机制顺势闭合（编辑必先 api.get 回填再全量提交，dept_id 带原值/新值不再为空）+ api.get 失败不开弹窗防呆。
- **seed 增量（幂等）**：+1 C 菜单（/sys/post sort10，组件 sys/post/index，图标 postcard menuIcon.ts 补映射）+4 F 码 sys:post:list/create/update/delete（与 handler_post.go RequirePerm 逐字一致）。真库连跑两遍：基线 48 菜单/97 授权/80 policy/0 岗位 → 53/107/88 → 53/107/88 不变。**B 部分种子最后一块清零，T-007c 扫出的 B 部分种子缺口全部偿还。**
- **回填闭环自测（头号正确性点，API 全链路 + DB 实查）**：dept_mgr 编辑 editor——① GET 详情回填基线（dept_id=技术部 hashid、posts omitempty 缺失实证）② 全量 payload 改部门→总公司+挂岗位 200 ③ 重开回显新值精确一致 ④ DB 实查 sys_user.dept_id + sys_user_post junction 行（user_id=2,post_id=1）真实存在 ⑤ post_ids=[] 清空语义回显复原 ALL PASSED。
- **enforce 正向证据（执行端前置自带）**：dept_mgr（非超管/不短路/真 Casbin，allMenuIDs 全量授权 seed.go:191-193）GET depts/tree + GET posts + POST/PUT posts 含写全 200 ↔ editor（仅 sys:user:list）五端点全 403/11009。**policy 80→88 增量**（+8 = sys:post:* 4 码×2 角色），与种子精确对应、二遍幂等不变。超管 200 不计为证据（T-007e 口径）。
- 浏览器验收：① 岗位页 CRUD（新增 bx_003 工程师/bx_004 自媒体 + 删除 bx_004 行消失共 2→1）② 用户编辑 temp02 部门回显「总公司」+ 岗位回显「工程师」（带 × tag + 下拉勾选态，回填可视坐实）③ dept 树选择 + post 多选 ④ XTable 其他页回归无碍。
- 安全自查：hashid 全程透传不解码（回填=出参塞 v-model、提交原样回传，后端 decode）；回填防误清（api.get 失败不开弹窗）；v-permission 真实码（sys:post:create/update/delete 逐字）；岗位删除二次确认按后端真实行为措辞（软删不可恢复、已分配用户即刻不再持有，不谎称拒删）；停用部门/岗位如实标注不伪造约束。
- 双推 Gitee+GitHub 一致（ls-remote 实测三处 HEAD=2460013 无单推）。commit 2460013（feature，11 files +372/-8）。

**T-007i 收尾记录（PM 评审定档，前端铺厚批次收官片，daxing 命令验收通过）**
- 定位：**前端铺厚批次（T-007a~i）的轻量收尾 + 给底座立前端测试基建**。纯前端基建，零后端改动、零业务代码改动、tree.ts 零触碰、不碰 openapi/接口/错误码。4 文件（2 新增：vitest.config.ts + src/utils/tree.spec.ts；2 修改：package.json scripts+devDeps、pnpm-lock.yaml）。
- **现状核实（先读再动手，反"典型 vitest 模板"假设）**：admin 自建脚手架 Vite ^8.0.12 / Vue ^3.5.34 / TS ~6.0.2 / pnpm 9.15，**无任何既有测试依赖**；TS 分文件结构（tsconfig.json references + tsconfig.app.json paths `@/*`→`./src/*`）；vite.config alias `@`→fileURLToPath('./src')。两处关键决策：① vitest 走独立 vitest.config.ts（引 vitest/config），alias 与 vite.config 单口径对齐；② environment=node（tree.ts 纯函数无需 DOM，最小依赖不引 jsdom/@vue/test-utils）。
- **tree.ts 单测（收编 T-007g 一次性脚本为正式入仓）**：17 条 vitest 断言覆盖附录 B 8 类（正常树/孤儿降根/自引用环/互引环/三元环入链/键名参数化/环状嵌套 visited 防护/空输入）+ 补全 subtreeIds 自身+后代/祖先旁系排除/根全集/目标不存在、buildTree 纯函数不改写入参等隐含点。用例数 17≠原报告口径"21"——原脚本未入仓，按 8 类汇总 + tree.ts 实际逻辑重建并补全（口径以入仓 spec 为准，§3 覆盖映射在报告）。
- **测试基建自身防假绿（本片头号正确性点——立测试更要防测试是假的）**：① **故意改坏一个断言**（同层顺序期望改错）→ `pnpm test` 真 FAIL（1 failed | 16 passed）→ 还原 → 17 passed，证明非空跑/全 skip 的假绿；② 首条断言显式校验 `import.meta.env.DEV === true`（vitest mode=test 下为真），否则后续"dev 告警被触发"的断言会集体假绿；③ 孤儿/环用例 `vi.spyOn(console,'warn')` 断言告警含「孤儿」/「成环」/「环状」被真调用。
- **build 不被破坏 + spec 纳入类型网**：`pnpm build`（vue-tsc -b && vite build）通过、exit 0、spec 不进生产产物（dist grep 无 tree.spec 痕迹）。**过程坐实并解决一处真问题**：vue-tsc -b 经 tsconfig.app（include src/**/*.ts）会类型检查 spec，buildTree<T> 返回 T[]（T 不含 children）→ 直接 .children 访问报 TS2339。**处置正路**：没把 spec 排除出类型检查（那会让 spec 脱离类型网、tree.ts 签名改了 spec 不跟也不报错），而是把 spec 写成类型干净（带可选 children 的 type 别名；用 type 非 interface——interface 缺隐式索引签名不满足 `T extends Record<string,unknown>`）。符合 Vue 官方脚手架"root references 让 build 一并检查测试"惯例。
- 残留 build 告警（`#__PURE__` 注释位置 / chunk >500KB）来自 vite8/rolldown 对 @vueuse/core 处理，与本片无关、build exit 0。
- 安全/卫生：测试纯内存构造脏数据零密钥；tree.ts 纯函数测试离线无网络/FS 副作用；vitest 全在 devDependencies（^4.1.8，caret 风格与周边一致），运行时 dependencies 零新增；.gitignore 已覆盖 node_modules/dist，本片未跑 coverage。
- **scope 未外扩**：仅立基建 + tree.ts 单测，未给 x-table/请求层/store/各 sys 页写测试、未引 jsdom/E2E、未改业务代码、未升既有 Vite/Vue/TS 版本。将来测组件再按需加 jsdom+@vue/test-utils（已留扩展路径注释）。
- 双推 Gitee+GitHub 一致（ls-remote 实测三处 HEAD=c2a3d90 无单推；GitHub 首次 SSL_ERROR_SYSCALL 重试即成、Gitee 走 Clash 代理出口直推）。commit c2a3d90（feature，4 files）。

**T-005b-1+2 收尾记录（PM 评审定档，后端债批次首切，daxing 验收通过）**
- 定位：**前端铺厚批次收官后转 T-005b 后端债批次的第一片——首个后端 Go 业务改动切片（前九片 T-007c~i 均纯前端消费/测试基建、后端零改动）。** 一次清掉 T-005b 篮子里两个 ★ 优先项（菜单·8-1 安全 + 文件·8-1 装配静默失败）。5 files +469/-6（3 修改：menu_service.go/auth/claims.go/demo/main.go；2 新增集成测试：menu_policy_integration_test.go/uploader_integration_test.go）。openapi 不升版（仍 v0.10.0，行为修复无新端点）、错误码零新增。
- **源码核实（7 项带出处，#4 全量决策 / #5 事务一致性明确）**：现有 ReloadAll（policy_sync.go:84）=全量重建（ClearPolicy → role_menu JOIN menu WHERE perm_code≠'' 重建 p → user_role 重建 g → SavePolicy），读 s.db 非 tx；调用点 grep（role AssignMenus:184 / user AssignRoles:307 / seed:197）；MenuService 原 {db,errs} 无 policySync（坐实 T-007g 观察）。
- **T-005b-1 菜单 policy 联动（★安全·修 T-003b）**：MenuService 加 policySync 字段 + SetPolicySync（同 UserService 范式）+ syncPolicy 助手；Create/Update/Delete 在变更**提交后**调 syncPolicy（全量 ReloadAll）。
  - **全量 vs 增量决策=全量**（依据：菜单 CUD 后台低频；从 role_menu JOIN menu 真相重建天然正确、**不误删其他角色/perm_code 的 policy**；与 role/user 共用同一 PolicySync 范式可维护性一致。增量须手动算受影响角色集易引新 bug，非必要不做，优化留注释）。
  - **事务一致性=先提交菜单、后重载**（ReloadAll 读 s.db 不可见未提交 tx，tx 内重载会读旧态重建错误）。残留口径同既有 AssignMenus：重载失败则菜单已提交、policy 滞留、返 error，靠下次写/重启 ReloadAll 幂等恢复。**真原子需改 PolicySync 接口让其读 tx（牵连 role/user 三处写路径），超本片 scope 未做 → 记待办。**
  - **构造签名=SetPolicySync 可选 setter 不改 NewMenuService 签名**（同 UserService 范式）→ 5 处既有菜单单测 + AuthInfoHandler 复用零回归、policySync nil 退化原行为。非对外契约变更，仅 demo 装配新增注入一行。
- **T-005b-2 uploader 修复（★装配静默失败·修 T-004b/T-007f 暴露）**：auth.Claims 加 `func (c *Claims) GetSubject() string`（指针接收者）。根因：system/handler_file.go getClaimsFromContext 鸭子断言 `interface{ GetSubject() string }`，但 auth.Claims 只有 Subject 字段、无该方法 → 断言静默失败返 ""（同 T-006 #3/#4/#5 编译过单测绿、装配/真跑才暴露类病）。全仓仅此一处依赖该断言（grep 实证），零破坏。修后 sys_file.uploader 真落值（上传者 Subject/内部 ID）、过滤可用。
- **policy 生命周期对照 + enforce 即时生效（本片头号正确性点，对照 T-007g 80→80→80 缺陷态）**：集成测试 TestMenuPolicyReloadE2E（dept_mgr 非超管 enforce 主体）——**建 F**：未授角色 total p 不变（no-op-until-assign，诚实）；**改 F**（perm_code 改名）：旧码 p 1→0、新码 0→1，dept_mgr GET 对应端点 **200→403**（旧码即时失联）；**改回**：恢复 200（双向可恢复）；**删 F**：该 perm_code p →0，dept_mgr **403**（删 F 权限真收回=核心安全证据）。
- **uploader 落值真跑**：集成测试 TestUploaderFillE2E——上传后 sys_file.uploader="3"（非空、=上传者 Subject）+ GET ?uploader=3 total=1（过滤可用）。
- **首个后端切片的验收教训（值得沉淀）**：改后端 Go 必须**重启 demo 重新编译**才生效——前九片纯前端有 Vite 热更新无需重启，daxing 惯性未重启 demo，初次验收看到 uploader 仍空（旧二进制），重启后落值。**启示：T-005b 后续后端切片（b-3/b-4）验收前必须先确认 demo 用新代码重启过。**
- pre-existing red（T-003d-fix，账本既有待办）：TestNewEnforcerMySQL_RoleInheritance git stash 在干净 HEAD 复跑实证同样 FAIL → 坐实非本片引入，归该待办切片（处理顺序铁律：先确认角色继承经 perm code 有覆盖再重写/删旧 URL keyMatch 断言，不许删了凑绿）。
- 安全自查：policy 全量重建不误删其他 policy（DB 实查 + enforce 翻转双证）；超管短路不受影响（验证用 dept_mgr 非超管）；uploader 不扩权（GetSubject 仅返 Subject）；scope 未外扩（未做手动重载入口/uploader 可读化/前端/菜单父子类型校验）。
- 双推 Gitee+GitHub 一致（ls-remote 实测三处 HEAD=f42dc80 无单推；Gitee 直推成功未触发 fake-IP 劫持）。commit f42dc80（feature，5 files +469/-6；T-005b-1+2 同 commit）。注：本片含 /clear 插曲——执行端会话中途被 /clear，工作区改动完好（/clear 不动文件），重新接上下文 + git status 确认 5 文件改动在 → 正常 commit 双推。

**T-005b-4 收尾记录（PM 评审定档，后端债批次第二片，daxing 验收通过）**
- 定位：T-005b 后端债第二片。给 dict/config/oper_log/login_log 列表补 search/filter/sort + 时间范围、dict_data 改服务端真分页，oper_log.operator/sys_file.uploader 出参解析用户名。后端 14 改 + 3 新（query.go/query_integration_test.go/query_enforce_integration_test.go）+ admin 8 文件，+1076/-114。openapi v0.10.0→v0.11.0、零新增错误码段、零 DDL。
- **第 0 节核实（带出处）**：0-1 dict_type 原仅 page/page_size 固定 id ASC（dict_service.go:57）；0-2 dict_data 原 ListDataByType 返全量数组无分页（dict_service.go:106/173/188）；0-3 config 原仅 page/page_size 固定 id ASC + maskEncrypted 脱敏；0-4 ListOperLogs 已有 startTime/endTime *time.Time 形参、handler 传 nil 未暴露（operlog.go:170/handler.go:183），operator 原精确等值、排序固定 id DESC；0-5 login_log 直接存 username（loginlog.go:34）无需可读化；0-6 关键：operator/uploader 存内部自增 ID 字符串(=claims.Subject=user.ID)与 sys_user.id 干净对应、无非用户主体混入→B 无阻断；0-7 软删=DeletedAt（rbac/model.go:49 + T003a sql:23），「已注销」靠 deleted_at 非 status；0-8 标准分页 gin.H{list,total,page,page_size}（response.go:56），dict_data 复用 enc.Page 逐字同构；0-9 排序白名单工具本片新建（rbac Query 结构范式沿用，applySort + 各 *Sortable map 新建）。
- **操作人可读化（选型 B 出参 JOIN，本片头号正确性点）**：非持久化 OperatorName/UploaderName（gorm:"-" 仅随 json）。**批量 IN 非 N+1**：fillOperatorNames（operlog.go:227）收集本页非空 ID→去重→resolveUserNames（query.go:69）一次 WHERE id IN 建 map→回填。过滤入参改「按用户名模糊」（userIDsByName→列 IN）。**「已注销」=「未命中即已注销」且显式 deleted_at IS NULL**（query.go:103 displayUserName：空→「匿名」/命中→username/未命中→「已注销」）——**关键潜伏点兜住**：resolveUserNames 走原生 .Table()，GORM 软删 scope 在 .Table 下不自动生效，故显式 Where("deleted_at IS NULL") 排除软删用户（否则软删用户被 JOIN 出来、「已注销」永不触发，又一例 .Table 下 scope 静默失效）。「已注销」「匿名」走集中常量（query.go:24-27 const，单一出口引用）非进 response.Registry——PM 采纳论证：Registry 是错误码 i18n 表，这两个是数据展示兜底值非错误消息，语义不属错误码段，进表会污染命名空间；集中常量已满足宪法 §6「不硬编码散落」真意（将来若做数据层多语言再统一收，低优待办）。id 仅作 map 键内部用不 marshal，出参只暴露 _name（守 T-004d 不暴露内部 ID）。
- **安全（排序注入头号）**：applySort（query.go:47）sort 经白名单（值=代码字面量列名）映射，未命中→回退默认 Order 不报错不注入；col 来自白名单 value 非用户输入。**负例 `id; DROP TABLE x;--` 被忽略回退默认、不进 ORDER BY、不报错实证**。LIKE 通配符转义 + 时间范围非法/start>end→400 不静默吞 + 分页封顶 100 + JOIN 仅 SELECT id,username 不带 password_hash + enforce 不旁路、可读化不扩权。
- **dict_data 真分页（后端 + 最小前端配套）**：服务端分页返 enc.Page 标准包络；前端 dict/index.vue:89 适配层一行透传 page/page_size + dict_type（去本地切全量 all.slice 旧写法）。
- **最小前端配套清单（8 文件，必要前置非擅扩，同 T-007f blob 先例）**：api/dict.ts（listDictData 收 params 返分页）+ views/sys/dict/index.vue（适配层切真分页）+ api/operlog.ts（OperLogRow 加 operator_name）+ views/sys/operlog/index.vue（列改绑 _name）+ api/file.ts（SysFileRow 加 uploader_name）+ views/sys/file/index.vue（列改绑 _name）+ views/sys/loginlog/index.vue（纯注释诚实修正：旧降级注释现已就绪）+ views/sys/config/index.vue（同前纯注释）。= PM 预期三项（dict 切真分页 + operator/uploader 改绑 _name + uploader 过滤随显示改按名）+ 2 个纯注释诚实修正，无逻辑擅扩。
- **openapi v0.11.0 diff**：各列表新增查询参数（dict_type keyword/name/status、config keyword/config_key/name/is_encrypted、oper operator_name/path/start_time/end_time + 400、login username/ip/success/start_time/end_time + 400、file uploader_name/original_name，各带 sort 白名单 + order/page/page_size）；oper_log 加 operator_name、sys_file 加 uploader_name 出参；dict_data 全量数组→分页包络（description 标版本起点供 PHP parity）。
- **daxing 验收五证**：①操作日志「操作人」列显 admin 用户名（截图坐实非内部 ID）②删用户→「已注销」认集成测试证据（软删 user 后 operator_name 返「已注销」-tags=integration 真跑）③字典页右表 dict_data 25 条切三页（第 2 页显项-11~20，非重复第 1 页/非全量堆出=服务端真分页，截图坐实）④文件页 uploader 列新上传显 admin、历史空值显「匿名」⑤时间范围 curl 五条全对（全量 total146→start_time=今天 total46→end_time=2020 total0→start_time 非法 400→start>end 400，且返回行 op=admin 证可读化在时间范围查询路径同生效）。
- **enforce 正向证据（执行端前置自带）**：dept_mgr（非超管/不短路/真 Casbin）6 个带新查询参数端点全 200 ↔ editor 全 403；operator 可读化 e2e（dept_mgr 写动作日志 operator_name=="dept_mgr"）。超管 200 不计为证据（T-007e 口径）。
- 偏差：①file uploader 过滤随显示列改按用户名模糊（任务书 §4 仅明确 oper）——属正确性必要前置（列已显用户名否则过滤框失效），PM 接受。②时间范围仅后端能力 + curl 验收，前端日期 UI 另开「前端补缺片」（date-range 非 x-table 现有 affordance、属页级 toolbar 控件，归补缺片低成本做），PM 裁定。
- pre-existing red（T-003d-fix）git stash 干净 HEAD 复跑同 FAIL 坐实非本片引入、未删断言凑绿。前端 pnpm build exit 0 + pnpm test 17 passed。
- **验收插曲（curl 000）**：执行端代跑 curl 一度全 000——shell 的 HTTP_PROXY/ALL_PROXY=127.0.0.1:7897（Clash）把 loopback:8080 也代理走，unset 代理后正常（demo 一直健康、daxing 浏览器验收未受影响）。已记 git/网络经验。
- 双推 Gitee+GitHub 一致（ls-remote 三处 HEAD=4e16f91，本轮 Gitee 未触发 fake-IP 劫持、未动隧道绕法）。commit 4e16f91（feature，25 files +1076/-114）。

**T-008a 收尾记录（PM 评审定档，用户/角色管理补缺片首片，daxing 验收通过）**
- 定位：用户/角色管理补缺片首片，纯前端、零后端 Go 改动、不升 openapi（消费既有端点）。**起因**：daxing 在 T-005b-4 验收时撞前端缺口——新建用户无法在界面分配角色/授权，登录后只有工作台、产生不了操作日志（卡住「删用户→已注销」浏览器路径）。摸底坐实一批「后端有、前端缺」的关系/管理装配界面缺口，PM 开「用户/角色管理补缺片」插队到 T-005b 后端债之前，本片是其中最轻、零后端依赖一片先做回血。admin 2 文件（api/user.ts + views/sys/user/index.vue）+148/-2。
- **第 0 节核实（带出处）**：0-1 PUT /sys/users/:id/password 入参 {password} 仅 binding required 无强度校验（handler_user.go:277）→ 前端预校验只做非空+二次确认不伪造规则；0-2 PUT /sys/users/:id/status {status:int8}，0=正常/非0=停用（model.go:45 + service_auth.go:136 停用拒登）；0-3 假能力根因坐实 updateUserReq 无 Status（handler_user.go:312）→ 编辑改 status 被静默吞；0-4/0-5 status 现在编辑表单内无 createOnly=假 UI，密码控件复用登录页 el-input password show-password。
- **重置密码**：用户页页级弹窗（v-permission sys:user:password）→ 输新密码 + 二次确认 → PUT :id/password。前端预校验只做非空+二次确认（与后端 binding required 对齐，不伪造强度规则）。
- **status 假能力修复（修 T-007h §8-3）**：status 标 createOnly 使编辑弹窗物理不渲染（XTable.vue:132 过滤）+ 行操作「停用/启用」按钮（v-permission sys:user:status）调 PUT :id/status。**状态变更唯一路径=独立端点**（后端 updateUserReq 本就不收 Status + 前端编辑弹窗不渲染=双保险，根除非遮住）。
- **偏差①（PM 接受）**：§2 原写「列内 el-switch」改为「行操作按钮」——列内 switch 需给 x-table 加单元格插槽=碰核心（§2 禁），覆写 #row-actions 又丢内置编辑/删除（插槽只暴露 row）。功能等价零核心改动、UX 为按钮非开关。PM 口径：将来 x-table 若要支持列内自定义单元格控件，作有意识能力增强独立评估（像 T-007h field-slot 全可选缺省零回归），不为单页临时塞。
- **观察（pre-existing 后端 quirk）**：SetStatus 把 status 设为相同值时 GORM RowsAffected==0→误返 404（实为「无变更」），属 T-003 既有；本片 toggleStatus 永远翻转值→必真变更→不触发。记低优待办（归 T-005b 后端债篮）。
- **daxing 验收三过**（纯前端无需重启 demo）：①重置密码后旧密码登录失效、新密码登录成功 ②停用后该用户登录被拒、启用后放行 ③编辑弹窗已无 status 字段（假能力真除、不再误导）。enforce：editor PUT password/status 403/403 ↔ dept_mgr 200/200（真值变更）。pnpm build exit 0 + pnpm test 17 passed。
- 双推 Gitee+GitHub 一致（ls-remote 三处 HEAD=c2e620a，本轮 Gitee 一次成功未触发 fake-IP 劫持）。commit c2e620a（feature，2 files +148/-2）。

**T-008b 收尾记录（PM 评审定档，用户/角色管理补缺片第二片，daxing 验收通过）**
- 定位：补缺片第二片，**本批第一个碰后端 Go 的补缺片**（T-008a 纯前端）。后端搭车补回填查询 + 前端分配角色弹窗 + 列表角色列。9 文件 +578/-4（后端 4 改：model.go/response.go/user_service.go/openapi.yaml + 2 新集成测试：assign_roles_integration_test.go/list_roles_integration_test.go；前端 3 改：api/role.ts/api/user.ts/views/sys/user/index.vue）。openapi v0.11.0→v0.12.0（SysUser 出参加 roles），零新增错误码、零 DDL。
- **第 0 节核实（带出处）**：0-1 PUT /sys/users/:id/roles perm sys:user:assign（handler_user.go:67），入参 {role_ids:[]string} hashid（:315 + decodeIDSlice :321），service 纯覆写 = tx.Where(user_id).Delete(SysUserRole) → 按 role_ids Create + 联动 SyncUserRoles（user_service.go:281）→ 回填是覆写正确性前提；0-2 UserService.Get（:147）原仅预载 Posts 不返 roles（SysUser model 无 Roles 字段 :53）→ 回填无现成来源确认；0-3 选型 A（详见下）；0-4 角色选项 GET /sys/roles 分页（handler_role.go:45）→ 前端循环翻页 listAllRoles 取齐（复用 T-007h listAllPosts 范式）；0-5 SysUserRole{UserID,RoleID} 复合 PK 无软删（model_role.go）→ 覆写=全删重建干净；0-6 用户页行操作现有编辑/删除/重置密码/停用启用（T-008a），分配角色加为 config.actions 第一项。
- **回填选型 A（GET /sys/users/:id 预载 roles，PM 默认，核实无阻断）**：UserService.Get 仅 1 调用点（handler_user.go:225，grep 实证）→ 预载 roles 只影响详情端点零污染其他出参；enc.User 手工构建出参（response.go，password_hash 从不进表）→ 加 roles 干净；List 不预载（显式 Select 无 role 列）+ roles,omitempty → 列表出参零污染、无 N+1（注：此为 T-008b 初版口径，列表角色列增量后 List 改为批量回填，见下）；与 T-008c「角色详情预载已授菜单」同口径统一；无新端点、无新 perm（复用 sys:user:get）。否决 B（单独端点）：Get 单调用点 + 手工编码器使 A 无副作用。
- **前端分配角色弹窗**：用户页行操作「分配角色」（v-permission sys:user:assign）→ el-select multiple（listAllRoles 选项 + getUser(id).roles 回填当前已授）→ 改选 → assignUserRoles（PUT :id/roles 全量覆写 role_ids hashid 数组）→ 成功 toast。**回填防误清**：openAssignRoles 先 Promise.all([getUser, listAllRoles])，任一失败不开弹窗（避残缺回填被全量覆写静默清空，同 T-007h api.get 防误清）。
- **列表角色列（daxing 验收追加诉求，PM 裁定归本片闭环非另开片）**：daxing 验收时提「用户列表看不到谁是什么角色」。PM 裁定归入本片（与分配角色同一关系场景，同片闭环优于拆开）。**先核实后实现（同 T-005b-4 节奏）**：UserService.List（:177-211）Find(&users) 后 fillUserRoles 批量回填——查询③ Model(&SysUserRole{}).Where("user_id IN ?", pageIDs).Find(&urs) → userID→[]roleID map + distinct roleIDs；查询④ Where("id IN ?", roleIDs).Find(&roles)（**model 查询让 GORM 软删 scope 自动剔除软删角色，故意不用 .Table()——反用账本第 7 例潜伏点：SysRole 有 DeletedAt，这次要 scope 生效故用 model 查询，.Table 才会绕过 scope 把软删角色查出来**）→ 内存一对多分组回填 users[i].Roles。复用 SysUser.Roles gorm:"-" + enc.User 输出（enc/model/handler 零改，只动 List 一处）。
- **批量非 N+1 硬证（本片头号正确性点）**：集成测试 TestListUserRolesBackfillAndQueryCount 用 GORM After("gorm:query") 回调真计数：**2 行页 = 4 次查询 == 6 行页 = 4 次**（Count+用户+junction+角色，固定不随本页行数 N 增长；N+1 反例会是 N 次）。比 T-005b-4 可读化（id→username 一对一）复杂一层=user→roles 一对多分组，但查询次数仍固定。
- **前端角色列**：column formatter 顿号文本（多角色「编辑员、部门经理」，无角色「—」），**不用 el-tag**（PM 裁定：el-tag 需 x-table 单元格插槽=碰核心 §2 禁，同 T-008a status 落点。**x-table 列内自定义单元格控件诉求已累积第三次（T-008a switch / 本片 tag），记可专门排「x-table 单元格插槽」基建片统一做，全可选缺省零回归**）。
- **openapi v0.12.0**：SysUser schema 加 roles 数组出参（id hashid + code/name，无敏感字段）；description 标「详情与列表均返（列表批量回填非 N+1）、omitempty」。列表列增量并入同一 v0.12.0（仅描述微调，无新字段/端点）。
- **daxing 验收四过**：①分配弹窗回显当前角色（temp05 回显 editor 勾选）②**覆写不误删**（temp05 编辑员基础上加选部门经理 → 提交后列表显「编辑员、部门经理」两个都在，原有未冲掉；集成测试 2→3→1 每步 GET roles 数 + DB 实查 sys_user_role 行数双重精确）③**分配角色后登录看到菜单**（temp05 挂 dept_mgr 角色后无痕登录 → 侧边栏完整菜单全出 ↔ 之前挂空菜单 editor 时只有工作台，同一 temp05 对照坐实链路正常）④列表角色列正确显示 + 分配后即时反映。
- **editor 空菜单核实结论 A（非 bug，daxing 验收 ③ 时撞见 → 只读核实排除下发 bug）**：editor 登录只有工作台,**对照 seed 自带 editor 账号同样空菜单**（排除「界面分配问题」）。只读核实坐实 A：editor 角色 seed 只授 1 个 sys:user:list（seed.go:183-188），而它 menu_type=F（按钮/perm 码不进侧边栏），授的 M/C 可见菜单数=0 → 空菜单是数据如此；GetUserMenuTree（menu_service.go:201-234）链路正确（user→role→role_menu→menu，menu_type IN(M,C) 过滤进侧边栏，editor 过滤后空集是正确结果）；**交叉对照 dept_mgr（seed 授 allMenuIDs 全量 M/C）同一链路满菜单 ↔ editor 空，唯一差异=授权数据 → 坐实链路无 bug（若链路坏 dept_mgr 也该空）**。真正缺口=「给角色配可见菜单」无界面=**T-008c 角色授权树**（账本既有「角色管理页缺分配菜单权限界面」待办指此），editor 留 T-008c 授菜单后补验。
- **Casbin g 联动 + password_hash 不泄漏（集成测试坐实）**：TestAssignRolesE2E——给 probe 分配含 sys:user:list 的 editor 角色 → probe 登录 GET 200；清空角色 → 403（g 规则真收回非只改 junction）。A 方案预载 roles 后 GET /:id 出参仍无 password_hash（断言，守 T-003a）。enforce 正向：editor PUT roles 403 + GET /:id 403 ↔ dept_mgr 200（超管不计为证据，T-007e 口径）。
- **本片有后端 Go 改动，daxing 验收前 demo 已重启**（吃 T-005b-1 教训，执行端重启 + 冒烟确认 GET /:id 与列表均带 roles）。pre-existing red（T-003d-fix）git stash 复证非本片引入（本片零触碰 enforcer）。前端 pnpm build exit 0 + pnpm test 17 passed。
- 无偏差（角色列文本非 el-tag = PM 裁定）。双推 Gitee+GitHub 一致（ls-remote 三处 HEAD=1d4b9aa，本轮未触发 fake-IP 劫持）。commit 1d4b9aa（feature，9 files +578/-4）。

**T-008c 收尾记录（PM 评审定档，用户/角色管理补缺片压轴，权限体系两段闭环，浏览器验收通过）**
- 定位：角色页加 el-tree「分配菜单」授权树（消费 GET /sys/menus/tree + 回填全量已授 menu_ids + check-strictly + 全量覆写 PUT /sys/roles/:id/menus）+ 后端搭车补 GET /sys/roles/:id 的 menu_ids 回填来源。后端 4 改 + 1 新集成测试，前端 2 改，共 7 files +160/-6。
- **只读摸底前置（沿用 T-005b-4 节奏：先核实后实现）**：出实现任务书前先让执行端只读摸底三口径（带 file:行号），坐实三事——① **写路径 + Casbin 联动已完整存在**（AssignMenus PUT /sys/roles/:id/menus，sys:role:assign，menu_ids hashid 数组经 decodeIDSlice，service 全量覆写先删后建 role_service.go:167-177 + SyncRolePerms 联动 :181 从 F 节点 perm_code Pluck，失败 ReloadAll 兜底），**自 T-003b 就接好、本片零改动**，区别于 T-005b-1 菜单 CUD 当时缺联动需补；② **回填 A 方案零阻断**（RoleService.Get 全仓唯一调用点 handler_role.go:64、List 走独立 enc.RoleList → 加预载零污染）；③ **口径①树半选落点坐实**（见下）。摸底结论经 PM 评审拍板选型再出正式任务书。
- **选型 check-strictly=true（PM 裁定·安全优先，采纳摸底推荐）**：摸底坐实 **role_menu M/C/F 三层扁平原样存**（AssignMenus 不做层级展开/裁剪，存什么完全由提交决定，seed.go:179-180/191-192 allMenuIDs 含三层）+ **GetUserMenuTree（menu_service.go:201-210）取该用户全部 role_menu menu_id 后 Where menu_type IN(M,C) 过滤建树、buildMenuTree（:280-296）对父 M 缺失的 C 走孤儿分支当根节点**（:291-293）→ **勾 C 不勾父 M：页面不丢、权限不丢，只是从目录掉出平铺到侧边栏顶层（纯 UX 降级非 correctness/安全）** → GetUserMenuTree **不要求**父 M 进 role_menu 才正确 → PM 上轮预设分水岭「除非摸底坐实需父 M 才正确才走级联」未触发 → check-strictly 默认成立。**三条加固理由**：(1) role_menu 扁平 + 全量覆写 → 回填与提交任何不一致即静默丢权限/越权；check-strictly 下 setCheckedKeys(stored)≡getCheckedKeys()（回填集≡勾选集≡提交集恒等往返），从根上消灭不一致面；(2) 级联两经典坑正中本项目「潜伏缺陷」红线——回填污染（勾父自动勾未授子 → 重提交越权扩权 + 资料损坏）+ 半选丢父（只交 getCheckedKeys 漏半选父 → 页面掉出目录），叠加且与防误清优先级冲突；(3) F 按钮天然要独立勾（授页面部分按钮是常态，级联反别扭）。**否决 ancestor 自动补全**（守 §2 scope + 语言中立 spec parity：「授页面隐含授目录」属后端 AssignMenus + spec 独立设计决策，前端单方补全会与后端/PHP parity 口径分叉）。
- **后端搭车（回填来源 A 方案，同 T-008b 用户侧同构）**：SysRole 加 `MenuIDs []uint64 gorm:"-" json:"menu_ids,omitempty"`（镜像 SysUser.Roles，model_role.go）；RoleService.Get 在 First(&role) 后 Pluck("menu_id") from SysRoleMenu where role_id=id（**全量 M/C/F 三层不过滤类型** —— 全量覆写下回填必须拿全集否则覆写丢权限）；enc.Role 在 MenuIDs 非空时输出 menu_ids hashid 数组（经 encoder，守 T-004d）；List（enc.RoleList）不预载、omitempty 不出。openapi v0.12.0→v0.13.0（SysRole schema 加 menu_ids、详情 only、含 M/C/F 说明），零新增错误码、零 DDL、零新 perm。
- **前端授权树（主体）**：角色页行操作「分配菜单」（v-permission sys:role:assign）→ el-tree（node-key="id"、:props label='name'（**摸底修正：标题字段是 name 非 title**）children='children'、show-checkbox、check-strictly、展开到 C 层）数据 = GET /sys/menus/tree（已嵌套，复用 T-007g getMenuTree 零再组装）。**防误清（吃 T-007h api.get 失败教训）**：打开弹窗先 Promise.all([getRole(id) 取 menu_ids, getMenuTree()])，任一失败不开弹窗、不提交 → 杜绝空树/残缺回填被全量覆写清光。回填 setCheckedKeys(menu_ids)、提交 getCheckedKeys()（check-strictly 下半选恒空即全集）。底部诚实提示「独立勾选（父子不联动）；提交后以当前勾选全量覆写该角色授权（含 Casbin 权限联动）」。**前端全程不解码 hashid**（菜单树 id/回填 menu_ids/提交 body 全 hashid 字符串闭环）。
- **集成测试五项全绿（-tags=integration 真 MySQL+Valkey，TestRoleAssignMenusE2E PASS）**：① 回填全量 M/C/F（GET /:id menu_ids=实存全集均 hashid）② **全量覆写往返不丢/无残留**（assign A→Get=A→A'(换一节点)→Get=A'→A→Get=A，每步 GET 出参与 DB role_menu 行数精确）③ enforce 正向（editor PUT menus/GET 详情 403 ↔ dept_mgr 非超管真 Casbin 200）④ **policy 生命周期 + 全链路 enforce**（授含 F(sys:user:list) 集 → casbin_rule p=1、绑该角色探针登录 GET /sys/users 200；覆写去该 F → p=0、探针重登同端点 403 = SyncRolePerms 真驱动可收回）⑤ enc.Role 加 menu_ids 后无 password_hash。同跑 TestAssignRolesE2E/TestMenuPolicyReloadE2E 仍 PASS（零回归）。pnpm build exit 0 + pnpm test 17 PASS（tree.ts 零改动）。
- **demo fresh 二进制冒烟（吃 T-005b-1 教训，验收前重启重编译）**：执行端重启 demo + 冒烟 GET /sys/roles/{super_admin} 返 menu_ids 54 条全 hashid 无 password_hash + List 行无 menu_ids（omitempty 零污染坐实）后再交验收。
- **浏览器验收（命门项 + 闭环）**：① 回填全量（dept_mgr 全树勾满）② check-strictly 父子不联动（editor 用户管理父勾、其下仅用户列表勾，其余子节点空）③ **大集合 round-trip 不漏勾（命门）**——dept_mgr 满授权取消「菜单删除」单项 → 重开「分配菜单」确认仅那一项变空、其余文件/字典/日志数十项纹丝不动（吃 T-007f「成功提示≠状态变更」教训，重开坐实而非只看 toast，比集成测试 DB 行数精确多一层真界面证据）④ **editor 补验闭环兑现**（给 editor 授可见 M/C 菜单 → editor/temp05 无痕登录侧边栏从空变出菜单）⑤ check-strictly UX 代价可控（整片勾父目录后 temp05 侧边栏系统管理整套正确嵌套、不再平铺，对照上一轮只授 C 时平铺顶层）。连带证 T-008b 列表角色列零回归（temp05 显「编辑员、部门经理」顿号拼接）。
- **已知 UX 非 bug（PM 选型已定，记入偏差）**：check-strictly 下勾页面 C 不自动勾父目录 M；只勾 C 不勾 M 时该页在侧边栏平铺顶层（GetUserMenuTree 仅渲染 role_menu 里 M/C，孤儿 C 落根 buildMenuTree）。不丢权限/仅丢目录分组；按目录整片勾即正常嵌套。已在页头注释 + openapi 说明。
- 安全自查：全量回填防静默丢权限（恒等往返）；防误清（Promise.all 任一失败不开）；hashid 透传不解码、menu_ids 出参经 encoder（守 T-004d）；enforce 边界 editor 403 实证后端真把关前端隐藏仅 UX；enc.Role 无敏感字段（守 T-003a）。
- 双推 Gitee+GitHub 一致（待执行端双推后 ls-remote 三处 HEAD=f88d3c7 核验）。commit f88d3c7（feature，7 files +160/-6）。

**待办（前端迭代，不阻塞）**
- 菜单树形管理页、日志/文件等其余 sys 页（复用 x-table）、x-table 高级能力（导出）、dept 树选择器。
- ~~admin 无 vitest 单测基建（本批拟由 T-007i 引入；在此之前沿用构建+类型+真人浏览器兜底）~~（✅ T-007i 已立 vitest 基建 + tree.ts 17 项单测，c2a3d90）。

**待办（衍生后端小切片，T-007d §8 上报，处置由 PM 裁定，不阻塞前端批次）**
暂记工作名 **T-005b「配置中心 / system API 增强」**（正式编号待定），打包 5 项：
- §8-1【缺陷】加密参数 create/update 写链路缺失：CreateConfigInput 无 is_encrypted、Update 全量明文覆写不再加密 → 经 API 无法建加密参数、对加密行任何提交都破坏密文。需补「加密参数更新语义」（碰 GCM 信封 + 大概率配套授权查看明文流程）。落地后前端放开加密行编辑。
- §8-2【缺口】dict/config 列表无 search/filter/sort，仅 page/page_size；dict_data 无分页返全量数组。补查询参数后前端解除列筛选/排序降级、dict_data 适配层一行切真分页。
- §8-3【缺口·demo】seed.go 不建任何 sys_config 行、无加密样例。补一条走 ConfigCenter.EncryptValue 的真密文样例，省手工 SQL。
- §8-4【观察】UpdateType 允许改 dict_type 且无级联 → 改名孤儿化 dict_data。PM 倾向直接禁改（与 config_key/前端立场一致，最简）。
- §8-5【观察】悬空 F 码 sys:secret:view 后端无端点消费，八成为「明文查看」预留 → 接到 §8-1 的明文查看端点。
（以下三项 T-007e §8 上报，并入 T-005b，同属"列表查询能力 / 可读性"补齐）
- 【日志·8-1·缺陷/可读性】sys_oper_log.operator 存内部自增 ID（demo subjectFn 返 claims.Subject）非用户名 → 列表可读性差、按用户名过滤不可用、与"对外不暴露内部 ID"（T-004d 精神）相悖。改法在后端/装配（存 username 或出参 hashid 化），前端已挂诚实注释、后端改后零改动生效。
- 【日志·8-2·缺口】日志列表无 search/sort：仅 page/page_size + 单字段精确过滤（oper=operator / login=username），无模糊/时间范围/排序（固定 id DESC）。前端 filterable/sortable 已挂参数透传，后端补参后零改动生效。同 §8-2 同类。
- 【日志·8-3·观察】时间范围过滤后端半成品：ListOperLogs service 签名已有 startTime/endTime，handler 传 nil,nil 未暴露（handler.go:183）→ 补两个 query 参数即点亮，属日志·8-2 最低成本子项。
（以下两项 T-007f §8 上报，并入 T-005b；均为 T-004b 潜伏后端缺陷，文件页消费时暴露）
- ~~【文件·8-1·缺陷·★优先】sys_file.uploader 恒空~~（✅ **T-005b-2 已修，f42dc80**：auth.Claims 加 GetSubject() 方法，uploader 真落值 + 过滤可用）：handler_file.go 类型断言 `interface{ GetSubject() string }`，但 auth.Claims 只有 Subject 字段、无 GetSubject() 方法（auth/claims.go）→ **装配期类型断言静默失败返零值**（同 T-006 #3/#4/#5 "编译过单测绿、装配才暴露"同类病），后果上传人恒空 + uploader 过滤不可用（比 operlog 存内部 ID 更甚，连 ID 都没存上）。修法极小：auth.Claims 加 GetSubject() 方法，零破坏。**标优先项**——这是装配期静默失败类缺陷，区别于其他"列表查询能力补齐"。前端列+过滤已挂、注释诚实归因，后端修后零改动生效。
- 【文件·8-2·缺口】response defaultMessages 缺全部 6 个 file 键（render.go 实查仅到 config 段）→ file 错误 message 返裸 i18n key（如 `sys.storage_failed`/`sys.file_too_large`）。浏览器路径前端预校验先中文拦截、不受阻；下载/删除后端错误透传 + curl 直连才见裸 key（T-007f 验收下载死链时肉眼撞见 `sys.storage_failed` 实证）。+6 行文案即点亮。
（以下两项 T-007g §8 上报，并入 T-005b；菜单·8-1 为 T-003b 潜伏安全缺陷，菜单页消费时暴露）
- ~~【菜单·8-1·缺陷·★优先·安全相关】菜单 CUD 不触发 Casbin policy 重载~~（✅ **T-005b-1 已修，f42dc80**：菜单写路径接全量 ReloadAll 联动，建/改/删 F 后 casbin_rule 真跟着变 + dept_mgr enforce 即时 200→403 权限真收回）：MenuService 无 policySync 字段，改/删 menu_type=F 节点的 perm_code 后 casbin p 规则滞留（ReloadAll 仅在 role AssignMenus 兜底/user 角色分配/seed 调用）。**安全后果**：① 删 F 节点（意图收回该权限）但旧 perm_code 的 p 规则仍在 → 已授角色用户仍 enforce 通过、权限实际收不回（"以为关了门其实没关"）；② 改 perm_code 致新码未进 policy、旧码变幽灵。policy 80→80→80 实测正是该缺陷证据（CUD 改了菜单 policy 纹丝不动）。属 T-003b 既有行为非 T-007g 引入。**标优先·安全相关**（区别于"列表查询能力补齐"批）：修法=菜单写路径 create/update/delete 接 policy 重载联动，或至少文档化 + 提供手动重载入口。
- 【菜单·8-2·文案】菜单父节点错误复用 ErrInvalidParentDept：移动成环/父不存在报「无效的**父部门**」（11035），语境是菜单（menu_service.go:67,118,123 复用部门错误）。+1 菜单专属码或改通用文案即愈，低优。
排期：不阻塞前端批次；若阶段二 BenxinKP 成主驱动可并入。

**待办（PROJECT_STATUS 账本卫生）**
- 工作区 PROJECT_STATUS.md 此前有 T-004e 收官账本未提交存量（+106/-19，内容为 T-007a~T-004e 收尾与切片表）——执行端识别正确未混入 T-007d 本片，按铁律账本提交独立于 feature。本轮整合进可重传副本。
- 历史任务书/报告 untracked 一批（T-004d/T-004e/T-007c 等）归档时单独走 docs: 提交，不和 feature 混。

**待 daxing 真人验收（用到时补，不阻塞）**
- 各片历史验收项见对应记录。demo 已用 e2e 自动覆盖大部分链路；真人可照 server/examples/demo/README.md 清单复核。

### 进行中 / 待收尾
| 任务编号 | 切片 | 状态 | 待收尾项 |
|---|---|---|---|
| — | — | — | 无 |

### 下一步（计划）
**T-008c 双推归档（HEAD f88d3c7）。当前格局：用户/角色管理补缺片三片全收官（T-008a 改密码/status + T-008b 用户分角色/列表角色列 + T-008c 角色授权树），权限体系两段界面（用户挂角色 + 角色挂菜单）闭环成型。下一片 = T-005b-3（配置中心加密参数写链路，体量最大）→ 文案/低优批收尾。补缺片是否还有遗留（部门管理页缺位 T-007h §8-4 / 日志 date-range / x-table 单元格插槽基建片）由 PM 评估排期。**

**🆕 用户/角色管理补缺片（T-008 系列，PM 裁定插队·消费驱动轻→重）：**
> 起因：T-005b-4 验收时 daxing 撞缺口——后端有 AssignRoles/AssignMenus/ResetPassword/SetStatus 接口，但前端缺「关系/管理装配界面」，新建用户无法分角色/授权、登录只有工作台。前端铺厚批次铺的是单表 CRUD，关系装配类界面整批被推后，现集中补。**回填端点随各片搭车补**（PM 选「消费驱动」：补丁与唯一消费者同片闭环，不单开后端前置片）。
- ✅ **T-008a（首片·已完成 c2e620a）**：用户改密码 + status 假能力修复。纯前端零后端。详见上方收尾记录。
- ✅ **T-008b（第二片·已完成 1d4b9aa）**：用户分角色（多选弹窗回填 + 全量覆写）+ 列表角色列（批量回填非 N+1）+ 后端搭车补回填（A 方案 GET /:id 预载 roles）。本批第一个碰后端 Go 的补缺片。详见上方收尾记录。**附：daxing 验收 ③ 撞见 editor 空菜单 → 核实结论 A（editor 角色 seed 只授 1 个 F 码、无 M/C 可见菜单，非下发 bug；用 dept_mgr 角色验证「分配角色后登录看到菜单」链路正常）→ editor 留 T-008c 授菜单后补验。**
- ✅ **T-008c（压轴·已完成 f88d3c7·角色授权树）**：角色页 el-tree 勾选授权（消费 GET /sys/menus/tree + 回填全量已授 menu_ids + **check-strictly 独立勾选** + PUT /sys/roles/:id/menus 全量覆写）+ 后端搭车补 GET /sys/roles/:id 的 menu_ids 回填来源（A 方案）。**写路径 + Casbin 联动自 T-003b 已完整、本片零改动**（区别 T-005b-1 菜单 CUD 需补）。选型 check-strictly（恒等往返消灭回填/提交不一致面，否决级联两经典坑）。editor 补验闭环兑现（授 M/C → temp05 侧边栏从空变满）。命门项「回填全量→改一处→提交不误删」大集合真界面坐实 + 集成测试五项全绿。openapi v0.13.0。详见上方收尾记录。**做完权限体系闭环：用户挂角色（T-008b）+ 角色挂菜单（T-008c）两段界面均可装配。**
- 同篮其余前端补缺（随 T-008 系列或单独排）：日志页 date-range 选择器（T-005b-4 偏差②带出，页级 toolbar 控件）；部门管理页缺位（T-007h §8-4，树表 CRUD + 移动部门，正好让 buildTree/subtreeIds 有真消费者）；角色 CRUD 已有但「分配菜单」即 T-008c。

**T-005b 后端批次裁期（PM 定·分子片 + 优先级，daxing 已认同先做安全洞）：**
T-005b 篮子（注意：以下均为**后端 Go 改动**，区别于前端批次的纯消费——验证须端到端真跑，不能只靠单测绿；**且改后端 Go 须重启 demo 重新编译才生效，验收前先确认重启**）：
- ✅ **T-005b-1+2（首切·★安全+★装配，已完成 f42dc80）**：菜单 CUD 接 Casbin policy 全量重载联动 + Claims.GetSubject() 修 uploader 恒空。详见上方收尾记录。
- ✅ **T-005b-4（第二片·列表查询能力 + 操作人可读化，已完成 4e16f91）**：dict/config/日志列表 search/filter/sort + 时间范围 + dict_data 真分页；operator/uploader 出参 JOIN 解析用户名（B 方案）。详见上方收尾记录。
- ~~**T-005b-3 配置中心加密参数写链路**~~（✅ d89e3e2，feature 1d21cf4）：CreateConfigInput 补 is_encrypted（=1 走 EncryptGCM 落密文）+ UpdateConfigInput ConfigValue *string 指针三态（取 DB 现有 is_encrypted 判据，nil=保持原密文不碰）+ 前端解禁加密行编辑（留空保持/重填替换）+ seed 加密样例。**未做明文查看**（sys:secret:view 悬空码不接端点，PM 三轮论证：覆盖式更新即可，明文回显净增泄漏面收益≈0，详见待办池备忘）。openapi v0.14.0、复用 ErrConfigDecryptFailed 不开新段、无 DDL。
- **文案/低优批（随上面搭车或最后统一收）**：文件·8-2 file 文案缺 6 键 / 菜单·8-2 错误复用 ErrInvalidParentDept / §8-4 dict_type 改名孤儿化（倾向禁改）/ §8-5 悬空 sys:secret:view（接 T-005b-3 明文查看）/ **SetStatus 同值 RowsAffected==0 误返 404（T-008a 观察，归此批）** / **数据展示文案「已注销」「匿名」现集中常量，将来数据层多语言再统一收（T-005b-4 观察）** / **可读化按名过滤边缘：删用户后其旧日志按名过滤不到、仅显已注销（T-005b-4 观察，可接受）**。
- **剩余顺序（PM 现序）**：~~T-008c（角色授权树，压轴）~~（✅ f88d3c7）→ ~~T-005b-3（加密参数，体量大）~~（✅ d89e3e2）→ ~~T-009a（禁收录，新需求池首条）~~（✅ feature a4b3fc8）→ **文案批收尾 / 媒体管理（先摸底再拆）排期**。补缺片三片（T-008a/b/c）+ 后端债 T-005b-1+2+4+3 + 禁收录 T-009a 已全收官。
- **🆕 新需求池（daxing 2026-06-15 提，已记待办段详见边界划分，不立即做）**：① **后台禁搜索引擎收录**（安全/隐私，noindex 三件套 + 参数化，低成本可搭车）② **媒体管理（图片/视频/音频）**（sys_file 能力增强片，分类/批量删/查询，耦合 x-table 多选基建，体量中大需先摸底再拆，可能拆「x-table 多选基建片 + 媒体管理消费片」）。两条均守业务中立——禁收录是底座统一能力、媒体管理只做通用 mime 分类不绑业务关联。排期由 PM 与 daxing 评估，可在 T-005b-3 后或按驱动力插队。
- **方向备选（随时可插队）**：若 BenxinKP 就绪成主驱动力，按"消费方驱动"既定策略可中止后端债直接进阶段二。当前 daxing 选择先补可装配界面 + 还后端债。

### 阶段二（底座可用后）
- BenxinKP 引入 BenxinAdminPro，只写业务；backend-php 照 spec 实现 parity。

## 待决策（F 系列，见底座文档 §14）
| # | 决策项 | 状态 |
|---|---|---|
| F1~F5 | 见底座文档 | ✅ 已决（F2 单独 Project 建议是） |

## T-004 子切片拆分
- T-004a response 接管 + 字典 + 参数 + 日志 ✅
- T-004b 文件管理 + 存储驱动 ✅
- T-004c 渲染收敛（handler 统一走 response.Render、errcode 降级常量）✅
- T-004d system 对外 ID hashid 化（全套）+ 抽 idcodec 中立包 ✅

## 备注
- 宪法级：安全第一、仅开源素材、配置驱动化、参数化复用、统一代码头注释（中英文+到秒）。
- 设计硬约束：表前缀随实例走禁包级（T-003a）；未完成切片接口至少挂 JWT；对外 ID 入出参 hashid 闭环+装配注入 hasher（T-003b）；授权变更事务内回滚保一致（T-003b）；数据权限失败一律收紧绝不放宽（T-003c）；日志脱敏+异步不阻塞、auth 不因日志依赖 DB（T-004a）。
- 错误码：crypto/auth（T-001/T-002）、sys（T-003 +30~+50）、system（T-004a +60~+63）、file（T-004b +70~+75）；response.Registry 为唯一注册/渲染权威（T-004c 已收敛）；errcode 已降级为纯常量；段不破坏、冲突 fail-fast。openapi 当前 v0.10.0。T-004e 唯一键冲突复用既有 ErrXxxExists 友好码（409），零新增段。T-007d/T-007e 纯前端消费（T-007e 含 demo seed 增量），未动错误码/openapi。T-007f 纯前端消费 + seed 增量 + 请求层 blob 透传，未动错误码/openapi。T-007g 纯前端消费（建 buildTree 通用树工具），seed/后端/错误码/openapi 全零改动。T-007h 纯前端消费 + seed 增量（sys:post:*）+ XTable 三项中立扩展，未动错误码/openapi；岗位 code 重名走 T-004e 既有兜底 11034。**T-005b-4 后端改动 openapi v0.10.0→v0.11.0（各列表新查询参数 + operator_name/uploader_name 出参 + dict_data 全量数组→分页包络），零新增错误码段。T-008a 纯前端消费既有端点（password/status），未动错误码/openapi。**T-008b 后端搭车补回填 openapi v0.11.0→v0.12.0（SysUser 出参加 roles 数组，详情+列表均返 omitempty），零新增错误码段、零 DDL。当前 openapi v0.12.0。**T-008c 后端搭车补回填 openapi v0.12.0→v0.13.0（SysRole 出参加 menu_ids hashid 数组，详情返/列表 omitempty），零新增错误码段、零 DDL、零新 perm。当前 openapi v0.13.0。**
- Casbin obj=perm code（非 URL），命名 模块:资源:动作（sys:user:list）；底座只放 sys:* 通用权限点。
- **流程铁律（T-006 教训）：账本完成判定权在 PM，执行端不得自标"完成"、不得自行双推、不得擅改 PROJECT_STATUS；切片必须经 PM 评审 + 双推确认才翻 ✅。**
- admin 前端拆分：T-007a 地基 ✅ / T-002b 验证码 ✅ / T-007b 权限+CRUD ✅（含登出）。阶段一前端完成。前端迭代批次：T-007c x-table 增强 ✅（2b0505e）→ T-004e 重名错误码 ✅（7b2aca4，后端）→ T-007d 字典/参数页 ✅（ac9b3b0）→ T-007e 操作/登录日志页 ✅（9e5ebb4）→ T-007f 文件页 ✅（5c998ba）→ T-007g 菜单树形 ✅（33679a3，建 buildTree）→ T-007h dept/post 选择器+回填 ✅（2460013，B 部分种子全清）→ T-007i vitest 基建 ✅（c2a3d90，tree.ts 17 项单测）。**前端铺厚批次 T-007a~i 全部收官、B 部分种子全清、测试基建已立。** 后端债批次 T-005b-1+2（f42dc80）+ T-005b-4（4e16f91）已清两片。**用户/角色管理补缺片 T-008 系列全收官：T-008a 改密码/status（c2e620a）→ T-008b 用户分角色/列表角色列（1d4b9aa）→ T-008c 角色授权树（f88d3c7，权限体系两段闭环）。下一步转 T-005b-3（加密参数）+ 文案低优批。**
- **测试铁律（#3/#4/#5 教训）：单测/集成"全绿" ≠ 装配能跑——三个潜伏缺陷都是 go build 通过、各模块单测绿，但真装配运行才暴露。凡被多模块组装的能力（demo、将来 BenxinKP 接入、前端联调），必须有真跑的端到端测试兜底。**
- **「装配/真界面操作才暴露」潜伏缺陷模式汇总（持续累积，已 7 例，值得作为底座质量经验沉淀）**：单测绿/编译过 ≠ 真能跑，这一类已反复出现——① T-006 #3 功能权限 enforce 未接线 ② #4 casbin 版本冲突 ③ #5 迁移器静默不建表 ④ 文件 uploader 类型断言恒失败（auth.Claims 缺 GetSubject()，装配期断言静默返零值，连 ID 都没存上，T-005b-2 已修）⑤ 菜单 CUD 不触发 policy 重载（删 F 权限收不回，T-005b-1 已修）⑥ 用户编辑表单无 dept_id 字段→每次界面编辑静默清空部门（T-007b 起，T-007h 回填机制连带闭合）⑦ **GORM 软删 scope 在原生 .Table() 查询下不自动生效（T-005b-4 可读化 resolveUserNames 若靠默认 scope 排软删用户则软删用户反被 JOIN 出来、「已注销」永不触发→执行端显式 Where("deleted_at IS NULL") 前置兜住）**。**共性：类型断言/接线/装配顺序/UI 字段缺失/ORM scope 在特定查询形态下静默失效等，编译与单测都过，唯有真装配运行或真界面操作才暴露，且部分是静默资料损坏（无报错、toast 成功、数据已损）。启示：底座关键链路（鉴权接线、装配依赖非 nil、写路径字段完整性、policy 同步、ORM scope 适用范围）需端到端真跑兜底，不能只靠单测绿。**
- 待办：rbac/system 既有集成测试硬编码 localhost:3306，理想应改 env 可覆盖（参考 e2e/migrator 已支持 3307）；将来顺手收。
- **待办（T-003d-fix 单列）**：TestNewEnforcerMySQL_RoleInheritance 在 78342cf 即红（与 T-003e/T-007d 无关），是 T-003b 改 model.conf 为 perm code 精确匹配时漏更新的陈旧 URL keyMatch 断言，非真鉴权 bug。**处理顺序铁律：先确认角色继承经 perm code 路径有覆盖，再重写/删除该旧断言，不许直接删了凑绿。** 附 CI 卫生缺口：rbac `-tags=integration` 在阶段一收官时这一标签未被真跑过。
- **待办（T-004b 低优·storage_key 暴露）**：SysFile 列表/详情出参含 storage_key（相对 key）。不泄漏绝对磁盘路径、uuidv7 不可枚举、非下载入口，敏感度低。**T-007f 已用到并处置：前端列表默认不展示该列（最小暴露 + UX，规避而非强逼后端改）**；是否后端 mask/省略仍属 T-004b 加固，低优、用到再收。
- **待办（openapi system 出参未强类型化）**：system 走 ApiResponse 泛型包络、data 体未定字段 schema，"id 是 hashid"仅靠 description 文字约束。将来 openapi 完善时强类型化 system 出参 schema。
- **待办（openapi 通用 envelope 未逐端点枚举 409）**：T-004e 友好码走既有 409，但各端点未逐一枚举 409 响应。将来 openapi 强类型化时补。
- **待办（junction 复合 PK 1062 未转码）**：sys_user_post 等关联表复合主键的 1062 未专门转友好码（边缘场景）。用到再评估入参去重。
- **待办（低优·GORM logger 打原始 1062 到 stderr）**：GORM SQL logger 在 service 捕获前把原始 Error 1062 + 索引名打到 stderr 服务端日志。非客户端泄漏（客户端拿干净 errcode:11062）、非攻击面。将来调 GORM logger 级别评估，不阻塞。
- **~~待办（B 部分种子，挂各切片）~~**（✅ 全部偿还）：T-007c 扫出的 B 部分种子缺口（operlog/loginlog/file/post 的 C 页 + F 码）已全部连页带码补齐——~~T-007e 补 sys:operlog:*/sys:loginlog:*~~（9e5ebb4）、~~T-007f 补 sys:file:*~~（5c998ba）、~~T-007h 补 sys:post:*~~（2460013）。**B 部分种子全清。**
- **🆕 待办池（T-005b-3 加密参数讨论沉淀·daxing×PM 三轮论证·2026-06-15）**：T-005b-3 定范围时讨论「加密参数明文查看」牵出一串依赖链，逐条定性归位（**均不进 T-005b-3 本片**）——
  - **① 加密参数明文查看 → 默认不做（备忘，非待办）**：论证结论硬——敏感密钥正确管理方式是「覆盖式更新」非「回看明文」（改即知道值、对就不用看、有疑虑重填覆盖即可）；且「看明文」防的是「能登录后台的人」，**防不住「能直接动数据库的人」**（攻击者能改 DB 就能直接读密文+主密钥解密，根本不走前门）→ 提供明文回看 = 多一条吐明文路径 = 净增泄漏面而收益≈0。**默认不提供明文回显**（很多成熟系统故意如此）。除非将来出现明确的、覆盖式更新解决不了的场景再评估。悬空 F 码 sys:secret:view 暂保留不接端点（接它即等于做明文查看，本片不做）。
  - **② 管理员绑手机号 + 2FA → 独立功能待办（账号安全增强，非看明文）**：手机号真正用途是登录两步验证/找回密码/异地登录提醒/敏感操作通知=保护账号本身，与「看加密参数明文」无关。将来值得做，独立排片。**注：绑定信息防篡改属下面 ④ 的纵深防御范畴。**
  - **③ 短信 / message 能力 → 独立功能待办（底座规划内，业务侧大概率要）**：架构文档 §4 模块清单有 drivers（短信/支付/音视频接口）+ message（站内信/邮件/短信），但**均未落地**（drivers 仅 storage/LocalDriver 已做，T-004b 核实；短信/message 零切片）。是独立中等以上功能（驱动接口 + 腾讯/阿里云实现 + 发送编排 + 配置页 + 发送日志）。**依赖顺序：短信能力要排在加密参数之后**（短信密钥本身应用加密参数存）。不阻塞本片。
  - **④ 主密钥管理 / DB 最小权限 / 敏感表审计 → 架构级安全加固议题（纵深防御，部署+运维维度）**：防「能直接动数据库的人」不是单个功能能解决，是一整套——应用 DB 账号最小权限（不给 DDL/不给改安全表）、主密钥不进 DB/代码走环境变量/KMS、敏感表（绑定/权限/加密参数）变更审计留痕告警。属部署形态 + 运维纪律 + 密钥管理，非某切片。记为底座安全加固议题，将来按部署规划推进。
- **🆕 待办（低优·UX·加密行视觉增强·daxing T-005b-3 验收提）**：加密行值列现固定显 ****** （6 位常量、与真实值长度无关——**这是安全正确行为，刻意不按真实长度显示以免泄漏密钥长度**，类比 GitHub Secrets / 各类云控制台）。daxing 提「空着像没值」的 UX 困扰属实，但解法**不是**按长度加星号（会泄漏长度情报），而是改成「🔒 已加密」标签替代裸星号——明确传达「有值且已加密」、零长度泄漏。属前端纯展示增强、零后端、零安全影响，将来搭车一片做。**注：维护者勿误把固定 6 位星号当 bug 改成真实长度，那是安全倒退。**
- **🆕 经验（已偿还·加密能力装配自检·T-005b-3）**：ConfigService 用 SetGCMKey setter 注入加密能力（非构造签名强制），固有「忘调 setter→运行时才暴露非装配期 fail-fast」弱点。**T-005b-3 已补两道兜底**：① demo 装配自检加「加密能力就绪(gcmKey 非空)」显式 fail-fast（沿用 T-006 关键依赖非 nil 范式）② seed 建加密样例(demo.secret_token 走 EncryptGCM)充当隐性自检（key 没注入 seed 阶段就炸 demo 起不来）。记此经验：底座库经 setter 注入的关键能力，装配层应配显式 fail-fast 自检，勿仅依赖隐性触发。
- **🆕 纪律记账（移除既有回归测试需 PM 复核·T-005b-3 立）**：T-005b-3 移除 TestDupConfigUpdateRename_MySQL（config_key 编辑锁定后「改名撞键」按设计不可达，PM 已逐案批准、非凑绿），做法规范（原处留注释 + Create 侧 11062 仍由 TestDupConfigSimpleCreate_MySQL 覆盖 + Update 内留 dberr.IsDuplicate 兜底）。**立纪律：移除既有回归测试属需 PM 逐案复核项，执行端不得自裁删测**（同 T-003d-fix「不许直接删了凑绿」精神）。
- ~~**🆕 待办（后台禁搜索引擎收录·安全/隐私·daxing 提·2026-06-15）**~~（✅ T-009a，feature a4b3fc8）：noindex 三件套已落地——① admin robots.txt `Disallow: /` ② 后端 httpmw.XRobotsTag 全局中间件注入 `X-Robots-Tag: noindex, nofollow`（参数化权威控制点，默认安全 SetDefault 兜底，对所有响应含 401/404 无差别注入）③ index.html robots meta 兜底。前端兜底件构建期固定不做 env 联动（A1 裁决）。**更根本（属部署形态另记，未做）**：管理后台正路=内网/VPN/IP 白名单/独立域名不挂公网 DNS；noindex 三件套是「万一暴露也不被收录」的兜底。**规范澄清：robots.txt 机器读协议文件不纳入五项代码头注释规范（T-009a 退回修复立）。**
- **🆕 待办（媒体管理·图片/视频/音频·daxing 提·2026-06-15·体量中大需先摸底再拆）**：上传的图片/视频/音频统一有记录、可分类、可上传、可删除/**批量删除**、可查询。**边界划清（守业务中立铁律）**：✅ 进底座 = 「通用文件能力的特化增强」——底座已有 sys_file 通用文件表 + StorageDriver + 鉴权上传/下载（T-004b）+ 文件管理页（T-007f），sys_file 已有 mime 字段（T-007f 核实）→ 按 mime 大类（image/* 视频 video/* 音频 audio/*）分类查询、批量软删、媒体预览属通用能力不绑业务；⛔ 不进底座 = 具体业务的媒体用法（哪张图属哪个商品/会员/内容）留业务侧（BenxinKP）扩展点。**本质 = sys_file 能力增强片**，拆解轮廓：后端（mime 大类分类查询 + **批量软删端点**，现状单条软删无多选见 T-007f §8；缩略图/图片宽高/视频时长等媒体元信息碰解码库，底座是否引单独评估，倾向先不引留扩展点）+ 前端（媒体管理页网格/缩略图视图、类型筛选 tab、多选+批量删、图片预览/视频播放器）。**关键耦合**：x-table 现无多选语义（账本多处记「未触发多选回 PM」）→ **批量删除是 x-table 多选能力首个真消费者**，会逼出 x-table 加 selection 列基建决策（可与累积四次的「x-table 单元格插槽基建片」一并评估）。排期：体量不小 + 耦合 x-table 多选基建，真排时先只读摸底（sys_file mime 现状 + 多选端点 + x-table 多选改造面）再拆。可能拆成「x-table 多选基建片 + 媒体管理消费片」两片。
- **待办（超管全显设计决策）**：前后端均无超管短路，"超管全显"依赖种子完整性——某码未 seed 则超管也看不到。是否做真正超管短路（需后端下发超管标识）作为底座成品独立设计决策评估。
- **待办（验收盲区教训，T-007b/c）**：角色页操作列空一直未被发现（验收没点进去）——肉眼验收会漏。后续 sys 页验收需逐页点操作列 + 真点一次 CRUD 验闭环，不止看按钮显示。T-007d/T-007e/T-007f 已照此执行（T-007d 字典/参数真点增删改；T-007e 详情弹窗逐行点开 + 两页各真点一次清理 + 正反对照；T-007f 上传正常+负例/下载落盘/删除三连真点，且**删除特别复验**——早前"toast 删除成功但列表行未消失"经追查定性为删存量死链行的边缘表现，用干净文件复验行真消失共 2→1 条才放行，坐实"成功提示 ≠ 状态变更"必须当场验通）。T-007g 树形特别验三类型动态表单字段显隐 + 移动节点选择器排除自身子孙 + 删有子节点目录友好拒删（非只删叶子）。
- **待办（日志清理粒度固定·T-007e §8-4）**：日志清理固定删 3 个月前硬编码（handler.go `AddDate(0,-3,0)`），不可配置不可指定天数。够用不阻塞，将来参数化连带做（带入参校验）。
- **待办（低优·登录日志 UA 空·T-007e §8-5）**：登录日志 API 直连（curl）无 UA 落空串，浏览器登录正常。非缺陷仅备忘。
- 待办：图形验证码可被 AI/OCR 识别是根本局限，后期评估行为验证码（滑块/点选/Turnstile），涉第三方与底座中立性，单独评估。
- git 经验：镜像仓建仓勿勾初始化文件否则首推 force；token 走交互式/钥匙串勿写进 remote url；GitHub/Gitee 偶发 SSL_ERROR_SYSCALL 重试即可（近期转频，必要时走代理/SSH）。
- **git 经验（T-007f 新·Clash TUN/fake-IP 劫持 Gitee）**：本机 Clash TUN/fake-IP 模式把 gitee.com 解析劫持为 198.18.0.8（fake-IP 段）且分流出口对 Gitee 不通，表现为双推 Gitee 必失败（非偶发 SSL 抖动）。**根治建议：Clash 给 gitee.com 加 DIRECT 直连规则**，否则每次双推复现。临时绕法（执行端 T-007f 用过）：DoH 查真实 IP（180.76.198.225）→ 本地 CONNECT 隧道 + `git -c http.proxy`（一次性参数，TLS 仍端到端校验、不禁证书、不改 hosts/仓库/全局配置、用后清理）。注（T-007g 观察）：gitee.com 仍被 fake-IP 解析为 198.18.0.8（DIRECT 规则仍未加），但 T-007g 双推经 Clash 代理出口（127.0.0.1:7897）对 Gitee 是通的、直推成功未动用隧道绕法——出口通断不稳定，根治仍以加 DIRECT 规则为准。
- **网络经验（T-005b-4 新·curl 本地服务被 Clash 代理走返 000）**：执行端代跑 curl 本机服务（localhost:8080）时，shell 的 `HTTP_PROXY/ALL_PROXY=127.0.0.1:7897`（Clash）会把 loopback 请求也代理走、导致全 000（连不上）。**修法：跑前 `unset HTTP_PROXY HTTPS_PROXY ALL_PROXY` 或 curl 加 `--noproxy '*'`**。与上面 fake-IP 劫持 gitee 同源（都是 Clash 代理对本地/特定域的副作用）。daxing 浏览器验收不受影响（浏览器不吃 shell 代理变量）、demo 本身健康。
- **待决策（menu 父子类型校验·T-007g §8-3）**：菜单父子类型后端无约束（validateMenuType 只管 perm_code，C 可挂 F 下等越界组合后端放行）；副作用=GetUserMenuTree 过滤 F 后子节点孤儿化提根。T-007g 前端按"不做伪安全"未擅自加限制。PM 倾向后端加 M>C>F 层级校验，作为底座成品独立设计决策评估（前端零改动跟进）。
- ~~**待办（角色管理页缺"分配菜单权限"界面·T-007b 遗留）→ 已排期 T-008c**~~（✅ **T-008c 已偿还，f88d3c7**：角色页 el-tree check-strictly 授权树 + 后端搭车补 GET /:id menu_ids 回填，写路径/policy 自 T-003b 已联动零改。editor 补验闭环兑现 editor/temp05 授 M/C 菜单后登录即时可见。命门项大集合改一处不误删真界面坐实）：原角色管理页只有 CRUD 无「给角色分配菜单/权限」界面，各角色权限仅靠 seed 写入界面无法查改（T-008b 验收坐实 editor seed 只授 1 个 F 码无 M/C 可见菜单 → 空菜单是此缺口真实后果）。**权限体系两段界面（用户挂角色 T-008b + 角色挂菜单 T-008c）均闭环。**
- **待办（PolicySync 非原子性·T-005b-1 记录）**：菜单/role/user 三处写路径共用的 PolicySync，重载失败会留"业务表已改、policy 没跟上"的中间态（返 error + 靠下次写/重启 ReloadAll 幂等恢复，与既有 AssignMenus 同级）。T-005b-1 沿用此口径未引入新问题。若将来要强原子（业务写 + policy 重载同事务回滚），需改 PolicySync 接口让其读 tx，牵连 role/user 三处写路径，单独评估。低优（幂等可恢复、非高频）。
- **经验（后端 Go 切片验收前必须重启 demo·T-005b-1 教训）**：前九片 T-007c~i 纯前端有 Vite 热更新、改完即生效无需重启；**从 T-005b 起改后端 Go 代码，必须重启 demo 重新编译才生效**。T-005b-1 验收时 daxing 未重启 demo（demo 是 Claude Code 后台起的、找不到终端），看到 uploader 仍空（旧二进制），重启后才落值。**后续后端切片（T-005b-3/4 等）daxing 验收前，PM 须提醒先确认 demo 用新代码重启过**（让 Claude Code 重启，或 `lsof -ti :8080 | xargs kill -9` 后 `cd server && go run ./examples/demo`）。
- ~~**待办（用户编辑 status 假能力·T-007h §8-3）**~~（✅ **T-008a 已修，c2e620a**：status 标 createOnly 编辑弹窗物理不渲染 + 行操作「停用/启用」按钮调独立 PUT /:id/status，状态变更唯一路径=独立端点，后端不收 + 前端不渲染双保险）：原用户编辑弹窗有 status 字段但 updateUserReq 无 Status → 改状态提交被静默忽略（"假 UI"）。
- **待办（部门管理页缺位·T-007h §8-4）**：seed 早有 /sys/dept C 菜单（组件 sys/dept/index）但 admin/src/views/sys/dept/ 不存在（T-007b 起占位降级空白页），部门管理页缺位。**若排期部门管理页（树表 CRUD + 移动部门），buildTree/subtreeIds 管线与菜单页同构可零成本复用——这也正是 buildTree 真正被调用的场景**（T-007h 验证了参数化够用但未实际调用）。PM 倾向排，让 buildTree 有真消费者，不急。
- **待办（T-007f §8-5·存量死链行）**：sys_file 存量行 1wR9wYV8（t004d.txt 等历史测试产物）物理文件已丢，点下载报 500/11075（既有数据不一致，非缺陷）。删除时软删元信息成功但视觉易误判"未变化"——验收已记此坑。无需修，验收时顺手删存量死链行即可。
