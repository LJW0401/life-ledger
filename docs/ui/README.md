# UI 设计

本目录用于存放 life-ledger 的 UI 设计资料，包括页面结构、交互说明、视觉规范、原型说明和设计决策。

建议按页面或功能模块拆分文档：

- `overview.md`：整体信息架构和导航结构
- `important-dates.md`：重要日期页面设计
- `transactions.md`：账单页面设计
- `decisions.md`：决策记录页面设计

首版只保留三个一级页面：日期页、账单页、决策页。提醒、分析、设置等能力后续迭代处理，不在首版实现或单独占用一级页面。

## 本地预览

`preview/` 目录提供一个静态 UI 原型和本地预览服务，用于快速验证页面结构、导航和信息密度。

启动方式：

```bash
go run docs/ui/preview/server.go
```

默认访问地址：

```text
http://127.0.0.1:4173/important-dates
```
