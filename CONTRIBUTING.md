# 扩展开发标准流程

## 1. 创建扩展

扩展 ID 必须使用小写 kebab-case（例如 `dbcore`、`sql-workbench`），且目录名、manifest `id`、Release 资产前缀保持一致。

每个扩展必须包含：

- `manifest.json`
- `README.md`
- 可执行入口（开发态可为脚本，发布态必须是自包含二进制）
- `GET /healthz`、`GET /manifest`、`POST /shutdown`
- 独立测试和构建说明

提交前在仓库根目录运行 `node scripts/validate-manifests.mjs`；和 GitHub Actions 使用同一套校验。

## 2. 本地开发

扩展不得依赖 Client 的进程内状态。启动时由 Client 注入以下环境变量：

```text
NEXUS_EXTENSION_ID
NEXUS_EXTENSION_VERSION
NEXUS_EXTENSION_PORT
NEXUS_EXTENSION_DATA_DIR
NEXUS_EXTENSION_TOKEN
NEXUS_EXTENSION_LOG_DIR
```

扩展只监听 `127.0.0.1`，所有来自 Client 的请求必须携带 `Authorization: Bearer <NEXUS_EXTENSION_TOKEN>`。

前端页面只请求扩展自己的相对 API；生产环境由 Client 代理到扩展进程，禁止把固定端口写进业务代码。

## 3. manifest 要求

使用 [`protocol/manifest.schema.json`](protocol/manifest.schema.json) 校验。至少填写：

- `id`、`name`、`version`、`apiVersion`
- `description`
- `entry.binary`
- `ui.entryPath`
- `repository`、`sourceRef`
- `permissions`

权限必须最小化声明。新增权限需要在 PR 描述中说明用途和安全影响。

## 4. 提交 PR

PR 必须包含：

- 功能说明和截图（如果包含 UI）
- 测试命令及结果
- manifest 变更说明
- 数据迁移和兼容性说明
- 是否改变权限、网络访问或用户数据行为

CI 会执行 manifest 校验、单元测试、构建和包内容检查。

## 5. 发布

使用以下标签触发 Release：

```text
extension/dbcore/v0.1.0
```

CI 按平台生成：

```text
nexus-extension-<id>-<version>-<os>-<arch>.tar.gz
```

每个资产必须附带 SHA-256 校验文件。签名密钥只存放在 GitHub Actions Secrets，不能提交到仓库。

## 6. 升级与卸载兼容性

- 升级前 Client 先停止旧进程，再解压新版本并运行迁移。
- 业务数据位于 `NEXUS_EXTENSION_DATA_DIR`，默认升级和卸载都保留。
- 卸载代码不能删除业务数据；彻底删除必须由用户显式确认。
- 扩展崩溃不得影响 nexus agent，Client 可独立重启扩展并记录日志。
