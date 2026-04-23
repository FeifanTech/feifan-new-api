# Seat Binding API 契约（D1-B4）

## 1. 列表

- `GET /api/seat_binding`
- 鉴权：Admin
- 响应（成功）：

```json
{
  "success": true,
  "message": "",
  "data": {
    "items": [
      {
        "id": 1,
        "tenant_id": "default",
        "user_id": 1001,
        "seat_id": "seat-001",
        "encrypted_token": "***",
        "account_type": "individual",
        "status": "active"
      }
    ],
    "total": 1
  }
}
```

## 2. 新建/更新

- `POST /api/seat_binding`
- 鉴权：Admin
- 请求体：

```json
{
  "tenant_id": "default",
  "user_id": 1001,
  "seat_id": "seat-001",
  "github_token": "ghp_xxx",
  "account_type": "individual"
}
```

- 必填：`user_id`, `seat_id`, `github_token`

## 3. 删除

- `DELETE /api/seat_binding/:id`
- 鉴权：Admin

## 4. 前端约定

- 页面路由：`/console/seat-binding`
- 菜单 key：`seat_binding`
- 失败响应统一按 `message` 提示。

