import { useState, useEffect } from 'react';
import { accountAPI, Account } from '../api/client';

export default function AccountsTab() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [showBatchModal, setShowBatchModal] = useState(false);
  const [searchToken, setSearchToken] = useState('');
  const [formData, setFormData] = useState({
    account_email: '',
    token: '',
    balance: 0,
    status: 'active',
    product_id: 1,
  });
  const [batchText, setBatchText] = useState('');

  useEffect(() => {
    loadAccounts();
  }, []);

  const loadAccounts = async () => {
    try {
      setLoading(true);
      const data = await accountAPI.list();
      setAccounts(data);
    } catch (error) {
      console.error('加载账号失败:', error);
      alert('加载账号失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!searchToken.trim()) {
      loadAccounts();
      return;
    }
    try {
      setLoading(true);
      const account = await accountAPI.getByToken(searchToken);
      setAccounts([account]);
    } catch (error) {
      console.error('查询失败:', error);
      alert('未找到该账号');
      setAccounts([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await accountAPI.create(formData);
      alert('创建成功');
      setShowModal(false);
      setFormData({
        account_email: '',
        token: '',
        balance: 0,
        status: 'active',
        product_id: 1,
      });
      loadAccounts();
    } catch (error) {
      console.error('保存失败:', error);
      alert('保存失败');
    }
  };

  const handleBatchImport = async () => {
    if (!batchText.trim()) {
      alert('请输入账号信息');
      return;
    }

    try {
      // 解析批量导入数据
      // 格式: email,token,balance 或 email|token|balance (每行一个)
      const lines = batchText.trim().split('\n');
      const accounts = lines.map((line) => {
        const parts = line.split(/[,|]/);
        if (parts.length < 2) {
          throw new Error(`格式错误: ${line}`);
        }
        return {
          account_email: parts[0].trim(),
          token: parts[1].trim(),
          balance: parts[2] ? parseFloat(parts[2].trim()) : 0,
          status: 'active',
          product_id: 1,
        };
      });

      const result = await accountAPI.batchCreate(accounts);
      alert(`导入完成！成功: ${result.success}, 失败: ${result.failed}`);
      setShowBatchModal(false);
      setBatchText('');
      loadAccounts();
    } catch (error) {
      console.error('批量导入失败:', error);
      alert('批量导入失败: ' + (error as Error).message);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个账号吗？')) return;
    try {
      await accountAPI.delete(id);
      alert('删除成功');
      loadAccounts();
    } catch (error) {
      console.error('删除失败:', error);
      alert('删除失败');
    }
  };

  const handleUpdateBalance = async (id: number) => {
    const balance = prompt('请输入新余额:');
    if (balance === null) return;
    const newBalance = parseFloat(balance);
    if (isNaN(newBalance)) {
      alert('余额格式错误');
      return;
    }
    try {
      await accountAPI.updateBalance(id, newBalance);
      alert('更新成功');
      loadAccounts();
    } catch (error) {
      console.error('更新失败:', error);
      alert('更新失败');
    }
  };

  if (loading) {
    return <div className="text-center py-8">加载中...</div>;
  }

  return (
    <div>
      <div className="mb-6 flex justify-between items-center gap-4">
        <div className="flex gap-2 flex-1 max-w-md">
          <input
            type="text"
            placeholder="按 Token 搜索"
            value={searchToken}
            onChange={(e) => setSearchToken(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
          />
          <button
            onClick={handleSearch}
            className="bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded-lg transition"
          >
            搜索
          </button>
          <button
            onClick={loadAccounts}
            className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition"
          >
            重置
          </button>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setShowBatchModal(true)}
            className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg transition"
          >
            批量导入
          </button>
          <button
            onClick={() => setShowModal(true)}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition"
          >
            + 添加账号
          </button>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  ID
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  邮箱
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Token
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  余额
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  创建时间
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {accounts.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-gray-500">
                    暂无数据
                  </td>
                </tr>
              ) : (
                accounts.map((account) => (
                  <tr key={account.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {account.id}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {account.account_email}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      <span className="font-mono text-xs">{account.token.substring(0, 20)}...</span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      ${account.balance.toFixed(2)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                          account.status === 'active'
                            ? 'bg-green-100 text-green-800'
                            : 'bg-red-100 text-red-800'
                        }`}
                      >
                        {account.status === 'active' ? '正常' : '禁用'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(account.created_at).toLocaleString('zh-CN')}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                      <button
                        onClick={() => handleUpdateBalance(account.id)}
                        className="text-blue-600 hover:text-blue-900"
                      >
                        充值
                      </button>
                      <button
                        onClick={() => handleDelete(account.id)}
                        className="text-red-600 hover:text-red-900"
                      >
                        删除
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* 添加账号模态框 */}
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">添加账号</h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">邮箱</label>
                <input
                  type="email"
                  value={formData.account_email}
                  onChange={(e) => setFormData({ ...formData, account_email: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Token</label>
                <textarea
                  value={formData.token}
                  onChange={(e) => setFormData({ ...formData, token: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none font-mono text-xs"
                  rows={3}
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">初始余额</label>
                <input
                  type="number"
                  step="0.01"
                  value={formData.balance}
                  onChange={(e) =>
                    setFormData({ ...formData, balance: parseFloat(e.target.value) })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                />
              </div>
              <div className="flex justify-end space-x-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition"
                >
                  取消
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition"
                >
                  创建
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* 批量导入模态框 */}
      {showBatchModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-2xl">
            <h3 className="text-lg font-semibold mb-4">批量导入账号</h3>
            <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-800">
              <p className="font-medium mb-2">格式说明:</p>
              <p>每行一个账号，使用逗号或竖线分隔</p>
              <p className="font-mono mt-1">email,token,balance (balance 可选)</p>
              <p className="font-mono mt-1">示例: user@example.com,sk-xxx123,10.00</p>
            </div>
            <textarea
              value={batchText}
              onChange={(e) => setBatchText(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none font-mono text-xs"
              rows={10}
              placeholder="user1@example.com,sk-token1,10.00&#10;user2@example.com,sk-token2,20.00"
            />
            <div className="flex justify-end space-x-3 pt-4">
              <button
                type="button"
                onClick={() => setShowBatchModal(false)}
                className="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition"
              >
                取消
              </button>
              <button
                onClick={handleBatchImport}
                className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition"
              >
                导入
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
