# 开发规划

本目录用于存放 life-ledger 的开发规划文档，包括 MVP 范围、阶段拆分、API 设计草案、数据模型设计和实现决策。

建议文档结构：

- `prd.md`：产品需求文档，定义 MVP 范围和验收标准
- `sad.md`：软件架构文档，定义技术栈、模块边界、数据流、部署和架构决策
- `sad-diagram.html`：软件架构图，使用 HTML + SVG 绘制
- `roadmap.md`：总体开发路线和阶段边界
- `phase-01-mvp.md`：第一阶段 MVP 开发计划
- `api-design.md`：HTTP API 设计草案
- `data-model.md`：SQLite 数据模型设计
- `deployment.md`：云服务器、Caddy、配置文件、备份和手动恢复说明

开发规划文档只记录会影响实现顺序、模块边界或验收标准的内容。单纯 UI 表现和交互细节放在 `docs/ui/`。
