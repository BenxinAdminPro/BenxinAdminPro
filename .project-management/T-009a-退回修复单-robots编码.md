# 退回修复单归档：T-009a robots.txt 编码缺陷

> 归档自 PM 验收退回修复单。修复说明见完成报告 `reports/T-009a-report.md` §10。

## 背景
daxing 浏览器验收 robots.txt 时发现：功能正确（`User-agent: *` / `Disallow: /` 两行生效），但文件顶部的中文代码头注释整片乱码（疑似被存成非 UTF-8 / GBK，浏览器按 UTF-8 解析花屏）。

## 问题定性
1. 编码非 UTF-8，违反 robots.txt 规范（应 UTF-8），有爬虫解析兼容性风险。
2. 更根本——robots.txt 是对外暴露给爬虫读的标准协议文件，不应套用源码的五项中文代码头注释（规范误用）。

## 修法（选项 B·PM 推荐·已采用）
robots.txt 去掉中文代码头注释，只留协议内容 + 一行纯英文注释：
```
# BenxinAdminPro admin - disallow search engine indexing
User-agent: *
Disallow: /
```
**规范澄清（PM 记账本）**：robots.txt 是机器读的协议文件、非源码，**不纳入「五项代码头注释」规范**。

## 执行端复核证据
- `file robots.txt` → `ASCII text`（纯 ASCII，无 UTF-8/GBK 解析歧义）。
- 非 ASCII 字节扫描 → 空；首字节 `0x23`（`#`），无 BOM。
- 重新 `pnpm build` → `dist/robots.txt` 同为 ASCII、内容正确。
- 顺带核查 index.html：首字节 `0x3c`（`<!doctype`）无 BOM、UTF-8；新增 robots meta 行纯 ASCII（中文说明注释合规，index.html 本为 lang=zh-CN 的 UTF-8 文档，未动）。

## 范围
只改 robots.txt 一处（去中文头改纯 ASCII），其余文件零改动；后端中间件 + 前端 meta 功能验收已过未动。

## 流程
PM 复核（纯文本编码修正）通过 → 不需 daxing 再验 → 放行双推。守 T-006 铁律。
