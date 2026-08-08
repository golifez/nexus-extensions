# Nexus Extensions

官方 Nexus 扩展源码仓库。

本仓库的目标是让扩展拥有独立的源码、版本、构建产物和运行进程，而 `nexus-clinet` 只负责扩展生命周期管理和网页承载。

## 仓库结构

```text
nexus-extensions/
├── extensions/              # 官方扩展，每个扩展一个独立目录
│   └── dbcore/
├── protocol/                # Client ↔ 扩展的稳定协议
├── registry/                # 扩展目录索引
├── templates/               # 新扩展脚手架
└── .github/workflows/       # 校验、构建和 Release 自动化
```

## 开发者快速开始

1. 从 `templates/extension` 复制一个目录到 `extensions/<extension-id>`。
2. 完成 `manifest.json`、`README.md`、健康检查和关闭接口。
3. 在本机运行扩展的独立进程，并通过 `protocol/extension-http-v1.md` 验证接口。
4. 提交 PR。CI 会校验 manifest、目录命名、测试和构建产物。
5. 使用 `extension/<id>/vX.Y.Z` 标签发布；CI 将为支持的平台生成 Release 包。

完整流程见 [`CONTRIBUTING.md`](CONTRIBUTING.md)，运行时边界见 [`ARCHITECTURE.md`](ARCHITECTURE.md)。

## 版本规则

- 扩展使用独立的 SemVer 版本，不跟随 Client 或 Server 版本。
- Tag 格式：`extension/<id>/v<major>.<minor>.<patch>`。
- 破坏性协议变更必须提升 `apiVersion`，并保留至少一个迁移版本。
- 用户安装 Release 产物；源码通过 manifest 的 `repository` 和 `sourceRef` 可追溯。

## 当前官方扩展

| ID | 名称 | 状态 |
| --- | --- | --- |
| `dbcore` | DbCore | 迁移中 |

