# AccountsTab 显示优化完成

## 修改内容

### 1. 前端 API 接口更新 (frontend/src/api/client.ts:116)

**添加 `used_balance` 字段到 Account 接口**

```typescript
export interface Account {
  id: number;
  account_email: string;
  account_password?: string | null;
  token?: string | null;
  product_id: number;
  source_id: number;
  user_id?: number | null;
  balance: number;
  used_balance?: number;  // ✨ 新增：已使用余额
  status: string;
  expire_date?: string | null;
  last_recharge_time?: string | null;
  remark?: string | null;
  version: number;
  create_time: string;
  update_time: string;
}
```

### 2. 前端 Token 列完整显示 (frontend/src/components/AccountsTab.tsx:412-416)

**移除省略号，完整显示 Token**

```tsx
<td className="px-6 py-4 text-sm text-gray-500 max-w-md">
  <div className="font-mono text-xs break-all whitespace-normal">
    {account.token || '-'}
  </div>
</td>
```

**关键样式说明：**
- `max-w-md` - 设置最大宽度（避免 Token 过长撑开整个表格）
- `break-all` - 允许在任何字符处断行
- `whitespace-normal` - 允许文本换行（替换原来的 whitespace-nowrap）
- `font-mono` - 等宽字体，便于阅读 Token
- `text-xs` - 小字号，节省空间

### 3. 后端 API 已支持 (internal/controller/admin_account.go:304,323)

**后端已经返回 `used_balance` 字段**

```go
type accountResponse struct {
    ID               uint64     `json:"id"`
    AccountEmail     string     `json:"account_email"`
    Token            *string    `json:"token"`
    ProductID        int64      `json:"product_id"`
    SourceID         int64      `json:"source_id"`
    UserID           *uint64    `json:"user_id"`
    Status           string     `json:"status"`
    ExpireDate       *time.Time `json:"expire_date"`
    Balance          float64    `json:"balance"`
    UsedBalance      float64    `json:"used_balance"`  // ✅ 已存在
    LastRechargeTime *time.Time `json:"last_recharge_time"`
    Remark           *string    `json:"remark"`
    Version          int        `json:"version"`
    CreateTime       time.Time  `json:"create_time"`
    UpdateTime       time.Time  `json:"update_time"`
}
```

## 前端表格列显示

表格现在显示以下列（已在 AccountsTab.tsx:376-384）：

| 列名 | 字段 | 说明 |
|------|------|------|
| ID | `account.id` | 账户ID |
| 邮箱 | `account.account_email` | 账户邮箱 |
| Token | `account.token` | **完整显示，允许换行** |
| 产品ID | `account.product_id` | 产品ID |
| 供应商ID | `account.source_id` | 供应商ID |
| 额度 | `account.balance` | **总余额** |
| 已用额度 | `account.used_balance` | **已使用余额** |
| 剩余额度 | `balance - used_balance` | **计算得出，欠费时红色显示** |
| 状态 | `account.status` | 正常/禁用 |
| 创建时间 | `account.create_time` | 账户创建时间 |
| 操作 | - | 充值/删除按钮 |

## 剩余额度显示逻辑 (AccountsTab.tsx:429-431)

```tsx
<td className={
  "px-6 py-4 whitespace-nowrap text-sm " +
  ((account.balance - (account.used_balance ?? 0)) < 0
    ? "text-red-600"   // 欠费：红色
    : "text-gray-900"  // 正常：黑色
  )
}>
  ${ (account.balance - (account.used_balance ?? 0)).toFixed(2) }
</td>
```

**特性：**
- 自动计算剩余额度：`balance - used_balance`
- 欠费时（剩余额度 < 0）显示为红色
- 正常时显示为黑色
- 格式化为 2 位小数

## 示例数据显示

### 正常账户
```
额度: $100.00
已用额度: $50.00
剩余额度: $50.00 (黑色)
```

### 欠费账户
```
额度: $100.00
已用额度: $120.00
剩余额度: $-20.00 (红色)
```

## 数据库字段 (entities.go:89)

```go
type Account struct {
    // ...
    Balance          float64    `db:"balance" json:"balance" gorm:"type:decimal(10,2);default:0"`
    UsedBalance      float64    `db:"used_balance" json:"used_balance" gorm:"type:decimal(10,2);default:0;comment:'已使用余额'"`
    // ...
}
```

## 测试步骤

1. **执行数据库迁移**
   ```bash
   mysql -u root -p your_database < migrations/add_used_balance.sql
   ```

2. **重新编译后端**
   ```bash
   go build -o codex-relay.exe
   ```

3. **重新编译前端**
   ```bash
   cd frontend
   npm run build
   ```

4. **启动服务**
   ```bash
   ./codex-relay.exe
   ```

5. **访问管理界面**
   - 打开浏览器访问 `http://localhost:8087`
   - 进入"账号管理"标签
   - 查看 Token 是否完整显示（允许换行）
   - 查看是否显示"额度"、"已用额度"、"剩余额度"三列
   - 检查欠费账户是否显示红色

## 注意事项

1. **Token 列宽度**：设置了 `max-w-md`（最大宽度），避免超长 Token 撑开表格
2. **换行显示**：Token 会在单元格内换行显示，不会有省略号
3. **等宽字体**：使用 `font-mono` 保持 Token 的可读性
4. **颜色提示**：剩余额度为负数时自动显示红色，提醒欠费状态

## 完成状态

✅ 前端 API 接口添加 `used_balance` 字段
✅ 前端 Token 列完整显示（无省略号）
✅ 后端 API 已返回 `used_balance` 字段
✅ 前端表格显示"额度"、"已用额度"、"剩余额度"三列
✅ 欠费账户剩余额度显示红色
