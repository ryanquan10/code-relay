# 账号管理-添加编辑功能

## 功能概述

为账号管理表格的每一行添加"编辑"按钮，允许管理员编辑账号的额度、已用额度和状态。

## 修改内容

### 1. 前端组件 (frontend/src/components/AccountsTab.tsx)

#### 新增状态变量
```typescript
const [showEditModal, setShowEditModal] = useState(false);
const [editingAccount, setEditingAccount] = useState<Account | null>(null);
const [editFormData, setEditFormData] = useState({
  account_email: '',
  token: '',
  balance: 0,
  used_balance: 0,
  status: 'active',
  product_id: 0,
});
```

#### 新增函数

**1. handleEdit - 打开编辑模态框**
```typescript
const handleEdit = (account: Account) => {
  setEditingAccount(account);
  setEditFormData({
    account_email: account.account_email,
    token: account.token || '',
    balance: account.balance,
    used_balance: account.used_balance || 0,
    status: account.status,
    product_id: account.product_id,
  });
  setShowEditModal(true);
};
```

**2. handleUpdate - 提交编辑**
```typescript
const handleUpdate = async (e: React.FormEvent) => {
  e.preventDefault();
  if (!editingAccount) return;

  try {
    // 调用更新 API，同时更新 balance 和 used_balance
    await accountAPI.updateBalance(
      editingAccount.id,
      editFormData.balance,
      editFormData.used_balance
    );

    alert('更新成功');
    setShowEditModal(false);
    setEditingAccount(null);
    loadAccounts();
  } catch (error) {
    console.error('更新失败:', error);
    alert('更新失败');
  }
};
```

#### 操作列添加编辑按钮 (AccountsTab.tsx:492-497)
```tsx
<button
  onClick={() => handleEdit(account)}
  className="text-indigo-600 hover:text-indigo-900"
>
  编辑
</button>
```

#### 编辑模态框 (AccountsTab.tsx:777-884)

**可编辑字段：**
- ✅ 总额度 (balance)
- ✅ 已用额度 (used_balance)
- ✅ 状态 (status)

**不可编辑字段：**
- ❌ 邮箱 (account_email)
- ❌ Token (token)
- ❌ 产品 (product_id)

**特殊功能：**
- 实时显示剩余额度计算
- 欠费时显示红色提示
- 灰色背景显示不可编辑字段

### 2. API 接口 (frontend/src/api/client.ts:133-134)

**更新 updateBalance 方法签名**
```typescript
updateBalance: (id: number, balance: number, usedBalance?: number) =>
  api.put(`/accounts/${id}/balance`, { balance, used_balance: usedBalance }),
```

**请求体格式：**
```json
{
  "balance": 100.00,
  "used_balance": 50.00
}
```

### 3. 后端 API (internal/controller/admin_account.go)

**已支持的端点：**
- PUT `/api/accounts/:id/balance`
- 请求体包含 `balance` 和可选的 `used_balance`

**处理逻辑 (admin_account.go:273-296)：**
```go
var req struct {
    Balance     float64  `json:"balance" binding:"required"`
    UsedBalance *float64 `json:"used_balance"`
}

// 更新 balance
repo.UpdateBalance(id, req.Balance)

// 如果提供了 used_balance，也更新它
if req.UsedBalance != nil {
    repo.UpdateUsedBalance(id, *req.UsedBalance)
}
```

### 4. Repository 层 (internal/repository/account_repository.go:162-190)

**新增方法：**
```go
// UpdateUsedBalance 直接设置已使用余额（用于初始化或修正数据）
func (r *AccountRepository) UpdateUsedBalance(id uint64, usedBalance float64) error
```

## UI 展示

### 操作列按钮

| 按钮 | 颜色 | 功能 |
|------|------|------|
| 编辑 | 紫色 (indigo) | 打开编辑模态框 |
| 充值 | 蓝色 (blue) | 快速充值（prompt） |
| 删除 | 红色 (red) | 删除账号 |

### 编辑模态框界面

```
┌─────────────────────────────────────┐
│ 编辑账号 (ID: 123)                  │
├─────────────────────────────────────┤
│ 邮箱                                │
│ [user@example.com] (不可编辑)      │
│                                     │
│ Token                               │
│ [sk-ant-xxx...] (不可编辑)         │
│                                     │
│ 总额度 *                            │
│ [100.00]                            │
│                                     │
│ 已用额度 *                          │
│ [50.00]                             │
│ 剩余额度: $50.00                    │
│                                     │
│ 状态 *                              │
│ [▼ 正常]                            │
│                                     │
│ 产品                                │
│ [产品A (product-a)] (不可编辑)     │
│                                     │
│           [取消]  [保存]            │
└─────────────────────────────────────┘
```

### 剩余额度显示规则

- **正常状态**：黑色文字
  ```
  剩余额度: $50.00
  ```

- **欠费状态**：红色文字 + 提示
  ```
  剩余额度: $-20.00 （欠费）
  ```

## 使用流程

### 编辑账号

1. 点击账号行的"编辑"按钮
2. 在弹出的模态框中修改：
   - 总额度
   - 已用额度
   - 状态（正常/禁用）
3. 点击"保存"按钮提交
4. 成功后自动刷新列表

### 示例场景

**场景1：调整账户余额**
- 原始：balance=100, used_balance=50
- 充值：balance=150, used_balance=50
- 结果：剩余 $100.00

**场景2：修正已用额度**
- 原始：balance=100, used_balance=120 (欠费)
- 修正：balance=100, used_balance=80
- 结果：剩余 $20.00

**场景3：禁用账号**
- 修改 status 从 "active" 到 "inactive"
- 账号被禁用，无法使用

## API 请求示例

### 编辑账号（只更新 balance）
```bash
PUT /api/accounts/123/balance
Content-Type: application/json

{
  "balance": 150.00
}
```

### 编辑账号（同时更新 balance 和 used_balance）
```bash
PUT /api/accounts/123/balance
Content-Type: application/json

{
  "balance": 150.00,
  "used_balance": 75.50
}
```

### 响应
```json
{
  "message": "updated"
}
```

## 权限和安全

### 不可编辑字段

以下字段出于安全考虑**不允许编辑**：

1. **邮箱** - 可能与认证系统关联
2. **Token** - 敏感凭证，不应随意修改
3. **产品ID** - 可能影响计费和权限

### 可编辑字段

以下字段允许编辑：

1. **总额度 (balance)** - 管理员充值/扣费
2. **已用额度 (used_balance)** - 修正错误数据
3. **状态 (status)** - 启用/禁用账号

### 建议

- 修改 `used_balance` 应谨慎，仅用于修正错误数据
- 正常消费应通过系统自动累加（IncrementUsedBalance）
- 定期审计余额修改记录

## 编译和部署

### 前端
```bash
cd frontend
npm run build
```

### 后端
```bash
go build -o codex-relay.exe
```

### 启动服务
```bash
./codex-relay.exe
```

### 访问
```
http://localhost:8087
```

## 测试清单

- [ ] 点击"编辑"按钮打开模态框
- [ ] 模态框正确显示账号信息
- [ ] 不可编辑字段为灰色且禁用
- [ ] 可编辑字段可正常输入
- [ ] 剩余额度实时计算显示
- [ ] 欠费时显示红色提示
- [ ] 点击"取消"关闭模态框
- [ ] 点击"保存"成功更新数据
- [ ] 更新后列表自动刷新
- [ ] 错误时显示错误提示

## 完成状态

✅ 前端添加编辑按钮
✅ 前端添加编辑模态框
✅ 前端实现编辑逻辑
✅ 更新 API 调用方法
✅ 后端支持 used_balance 更新
✅ 前端编译成功
✅ 后端编译成功
✅ 文档完成

## 效果截图位置

操作列现在显示：
```
[编辑] [充值] [删除]
```

编辑按钮为紫色（indigo），与其他按钮区分。
