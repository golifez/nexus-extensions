# Nexus 扩展运行架构

```text
nexus-clinet
├── Extension Manager       # 安装、校验、启动、停止、升级、卸载、日志
├── nexus agent              # 主 Agent 进程
└── Extension Web Host      # 只承载扩展网页（iframe/WebContents）
          │ localhost HTTP
          ▼
nexus-extension-dbcore      # 独立进程，拥有自己的 API、依赖和数据目录
```

## 边界

- Client 不导入扩展的业务 Go/Node/Python 包。
- Client 不直接访问扩展固定端口；统一使用 `/api/extensions/<id>/...` 代理。
- 扩展不能读取 Client 数据库文件，必须使用注入的数据目录和显式协议。
- 扩展的网页由 Client 嵌入，生命周期由 Extension Manager 控制。

## 生命周期

```text
available → installing → installed → starting → running
                                      │             │
                                      └→ failed ← stopped
```

Client 启动时恢复上次标记为 `running` 的扩展。崩溃最多自动重启 3 次并采用退避；状态和日志按扩展隔离。

## 安装来源

官方扩展来自 registry 中的签名 Release 包，也支持开发者导入本地 `.tar.gz`。包必须包含 manifest、可执行入口和校验文件。第一版不在用户机器上现场安装 Go、Node 或 Python 依赖，依赖必须随包提供。

