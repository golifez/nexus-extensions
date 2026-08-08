# Extension HTTP Protocol v1

扩展进程监听 Client 分配的本机端口。

## 必选接口

```http
GET /healthz
GET /manifest
POST /shutdown
```

`/healthz` 返回：

```json
{"ok":true,"extensionId":"dbcore","version":"0.1.0"}
```

`/shutdown` 只接受来自 Client 的 Bearer token，成功返回 `202 Accepted`，扩展随后优雅退出。

## UI

manifest 的 `ui.entryPath` 是扩展网页入口。Client 将扩展页面嵌入自己的扩展容器，并通过 `/api/extensions/<id>/...` 代理 API。扩展前端不得假定外部端口或 Electron API 存在。

## 错误格式

所有非 2xx 响应使用：

```json
{"error":{"code":"SOURCE_NOT_FOUND","message":"数据源不存在"}}
```

