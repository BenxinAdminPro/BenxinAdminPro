# CREDITS — 第三方素材与许可（admin 前端）

> 宪法级约束：仅引入开源素材；不引商业字体、Font Awesome Pro、第三方 admin 模板。
> 本文件记录 admin 前端引入的运行时依赖与素材及其许可。

## 运行时依赖（npm，均为开源许可）

| 包 | 用途 | 许可 |
|---|---|---|
| vue | 框架 | MIT |
| vue-router | 路由 | MIT |
| pinia | 状态管理 | MIT |
| element-plus | UI 组件库 | MIT |
| @element-plus/icons-vue | 图标（仅用其中开源图标） | MIT |
| axios | HTTP 客户端 | MIT |
| vue-i18n | 国际化 | MIT |
| @vueuse/core | 组合式工具（useDark/useWindowSize 等） | MIT |
| unocss | 原子化 CSS | MIT |
| sass | SCSS 预处理（devDependency） | MIT |
| vite / @vitejs/plugin-vue | 构建工具 | MIT |
| typescript / vue-tsc | 类型检查 | Apache-2.0 / MIT |

## 字体
- 未引入任何商业/自定义字体；样式使用系统字体栈（Helvetica Neue / PingFang SC / Microsoft YaHei / Arial 等系统内置），无字体文件分发，无许可负担。

## 图标
- 仅使用 Element Plus 官方图标包 `@element-plus/icons-vue`（MIT）。未引入 Font Awesome Pro 或任何商业图标集。

## 模板
- 自建脚手架，未使用任何第三方 admin 模板（vue-element-admin / arco-pro 等一律未引入），依赖可控。

## 验证
- `pnpm licenses ls` 可复核依赖许可（均为 MIT/Apache/ISC 等宽松开源许可）。
