import { useState, useEffect } from 'react';
import { accountAPI, productAPI, Account, Product } from '../api/client';

export default function AccountsTab() {
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [products, setProducts] = useState<Product[]>([]);
    const [loading, setLoading] = useState(true);
    const [productsLoading, setProductsLoading] = useState(false);
    const [showModal, setShowModal] = useState(false);
    const [showEditModal, setShowEditModal] = useState(false);
    const [showBatchModal, setShowBatchModal] = useState(false);
    const [showGenerateModal, setShowGenerateModal] = useState(false);
    const [searchToken, setSearchToken] = useState('');
    const [editingAccount, setEditingAccount] = useState<Account | null>(null);
    const [selectedIds, setSelectedIds] = useState<number[]>([]);
    const [formData, setFormData] = useState({
        account_email: '',
        account_password: '',
        token: '',
        balance: 0,
        status: 'active',
        product_id: 0,
    });
    const [editFormData, setEditFormData] = useState({
        account_email: '',
        token: '',
        account_password: '',
        balance: 0,
        used_balance: 0,
        status: 'active',
        product_id: 0,
        use_status: 0,
    });
    const [batchDefaults, setBatchDefaults] = useState({
        product_id: 0,
        status: 'active',
    });
    const [batchText, setBatchText] = useState('');
    const [batchFieldKeys, setBatchFieldKeys] = useState('account_email,account_password');
    const [batchCount, setBatchCount] = useState('1');

    const generateRandomHex = (bytes: number) => {
        const buffer = new Uint8Array(bytes);
        if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
            crypto.getRandomValues(buffer);
        } else {
            for (let i = 0; i < buffer.length; i += 1) {
                buffer[i] = Math.floor(Math.random() * 256);
            }
        }
        return Array.from(buffer)
            .map((value) => value.toString(16).padStart(2, '0'))
            .join('');
    };

    const normalizeTokenPrefix = (value: string) => value.trim().replace(/-+$/, '');

    const buildDefaultToken = (productCode: string) => {
        const prefix = normalizeTokenPrefix(productCode);
        const random = generateRandomHex(16);
        return prefix ? `${prefix}-${random}` : random;
    };

    const getSelectedProduct = (productId: number) =>
        products.find((product) => product.id === productId) ?? products[0];

    const buildDefaultFormData = (productId?: number) => {
        const product = getSelectedProduct(productId ?? 0);
        if (!product) {
            return {
                account_email: '',
                account_password: '',
                token: '',
                balance: 0,
                status: 'active',
                product_id: productId ?? 0,
            };
        }
        return {
            account_email: '',
            account_password: '',
            token: buildDefaultToken(product.product_code),
            balance: product.default_balance ?? 0,
            status: 'active',
            product_id: product.id,
        };
    };

    const handleOpenModal = () => {
        setFormData(buildDefaultFormData(formData.product_id));
        setShowModal(true);
    };

    const handleProductChange = (productId: number) => {
        const product = getSelectedProduct(productId);
        if (!product) {
            setFormData((prev) => ({ ...prev, product_id: productId }));
            return;
        }
        setFormData((prev) => ({
            ...prev,
            product_id: product.id,
            token: buildDefaultToken(product.product_code),
            balance: product.default_balance ?? 0,
        }));
    };

    const handleRegenerateToken = () => {
        const product = getSelectedProduct(formData.product_id);
        if (!product) {
            return;
        }
        setFormData((prev) => ({
            ...prev,
            token: buildDefaultToken(product.product_code),
        }));
    };

    useEffect(() => {
        loadAccounts();
        loadProducts();
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

    const loadProducts = async () => {
        try {
            setProductsLoading(true);
            const data = await productAPI.list();
            setProducts(data);
            if (data.length > 0) {
                setFormData((prev) => {
                    const product = data.find((item) => item.id === prev.product_id) ?? data[0];
                    const nextToken = prev.token.trim() ? prev.token : buildDefaultToken(product.product_code);
                    const nextBalance = prev.balance === 0 ? (product.default_balance ?? 0) : prev.balance;
                    return {
                        ...prev,
                        product_id: product.id,
                        token: nextToken,
                        balance: nextBalance,
                    };
                });
                setBatchDefaults((prev) => {
                    const nextProductId = data.find((item) => item.id === prev.product_id)?.id ?? data[0].id;
                    return {
                        ...prev,
                        product_id: nextProductId,
                    };
                });
            }
        } catch (error) {
            console.error('加载产品失败:', error);
        } finally {
            setProductsLoading(false);
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
            if (!formData.product_id) {
                alert('请选择产品');
                return;
            }
            const selectedProduct = getSelectedProduct(formData.product_id);
            const trimmedPassword = formData.account_password.trim();
            const normalizedToken = formData.token.trim();


            if (!normalizedToken && !trimmedPassword) {
                alert('Token 或密码必须填写一个');
                return;
            }
            const balance = Number.isFinite(formData.balance)
                ? formData.balance
                : (selectedProduct?.default_balance ?? 0);
            if (!formData.status.trim()) {
                alert('请选择状态');
                return;
            }
            const payload: Partial<Account> = {
                account_email: formData.account_email.trim(),
                balance,
                status: formData.status,
                product_id: formData.product_id,
            };
            if (normalizedToken) {
                payload.token = normalizedToken;
            }
            if (trimmedPassword) {
                payload.account_password = trimmedPassword;
            }
            await accountAPI.create(payload);
            alert('创建成功');
            setShowModal(false);
            setFormData(buildDefaultFormData(formData.product_id));
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

    if (!batchDefaults.product_id) {
        alert('请选择默认产品');
        return;
    }

    try {
        // 使用后端的文本解析模式，而不是前端自己解析
        // 这样可以利用后端的 ExtractAccountsMultiLevel 函数
        const fieldKeys = batchFieldKeys.split(',').map((s) => s.trim()).filter(Boolean);

        const payload = {
            text: batchText,
            field_keys: fieldKeys.length > 0 ? fieldKeys : ['account_email', 'account_password'],
            product_id: batchDefaults.product_id,
            status: batchDefaults.status || 'active',
        };

        const result = await accountAPI.batchCreate(payload as any);
        alert(`导入完成！成功: ${result.success}, 失败: ${result.failed}`);
        setShowBatchModal(false);
        setBatchText('');
        loadAccounts();
    } catch (error) {
        console.error('批量导入失败:', error);
        alert('批量导入失败: ' + (error as Error).message);
    }
};

    const handleBatchCreate = async () => {
        const count = Number(batchCount);
        if (!batchDefaults.product_id) {
            alert('请选择批量新增的产品');
            return;
        }
        if (!Number.isInteger(count) || count <= 0) {
            alert('请输入有效的数量');
            return;
        }

        try {
            const accounts = Array.from({ length: count }, () => ({
                status: batchDefaults.status,
                product_id: batchDefaults.product_id,
            }));
            const result = await accountAPI.batchCreate(accounts);
            alert(`批量新增完成！成功: ${result.success}, 失败: ${result.failed}`);
            setShowBatchModal(false);
            setShowGenerateModal(false);
            setBatchText('');
            loadAccounts();
        } catch (error) {
            console.error('批量新增失败:', error);
            alert('批量新增失败');
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
        const balance = prompt('请输入新额度:');
        if (balance === null) return;
        const newBalance = parseFloat(balance);
        if (isNaN(newBalance)) {
            alert('额度格式错误');
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

    const handleEdit = (account: Account) => {
        setEditingAccount(account);
        setEditFormData({
            account_email: account.account_email,
            token: account.token || '',
            account_password: account.account_password || '',
            balance: account.balance,
            used_balance: account.used_balance || 0,
            status: account.status,
            product_id: account.product_id,
            use_status: account.use_status || 0,
        });
        setShowEditModal(true);
    };

    const handleUpdate = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!editingAccount) return;

        try {
            const trimmedPassword = (editFormData.account_password || '').trim();
            const normalizedToken = (editFormData.token || '').trim();
            if (!normalizedToken && !trimmedPassword) {
                alert('Token 或密码必须填写一个');
                return;
            }
            // 调用更新 API，同时更新 balance 和 used_balance
            const payload: any = {
                status: editFormData.status,
                balance: editFormData.balance,
                used_balance: editFormData.used_balance,
                product_id: editFormData.product_id,
                token: normalizedToken,  // 即使是空也要更新
            };
            if (editFormData.account_password && editFormData.account_password.trim()) {
                payload.account_password = editFormData.account_password.trim();
            }
            await accountAPI.update(editingAccount.id, payload);

            alert('更新成功');
            setShowEditModal(false);
            setEditingAccount(null);
            loadAccounts();
        } catch (error) {
            console.error('更新失败:', error);
            alert('更新失败');
        }
    };

    // 处理单个选中/取消选中
    const handleToggleSelect = (id: number) => {
        setSelectedIds(prev =>
            prev.includes(id)
                ? prev.filter(selectedId => selectedId !== id)
                : [...prev, id]
        );
    };

    // 处理全选/取消全选
    const handleToggleSelectAll = () => {
        if (selectedIds.length === accounts.length) {
            setSelectedIds([]);
        } else {
            setSelectedIds(accounts.map(account => account.id));
        }
    };

    // 批量删除
    const handleBatchDelete = async () => {
        if (selectedIds.length === 0) {
            alert('请先选择要删除的账号');
            return;
        }

        if (!confirm(`确定要删除选中的 ${selectedIds.length} 个账号吗？`)) {
            return;
        }

        try {
            // 调用批量删除 API
            await accountAPI.batchDelete(selectedIds);
            alert(`成功删除 ${selectedIds.length} 个账号`);
            setSelectedIds([]);
            loadAccounts();
        } catch (error) {
            console.error('批量删除失败:', error);
            alert('批量删除失败');
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
                    <button
                        onClick={loadAccounts}
                        className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition"
                    >
                        刷新列表
                    </button>
                </div>
                <div className="flex gap-2">
                    {selectedIds.length > 0 && (
                        <button
                            onClick={handleBatchDelete}
                            className="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-lg transition"
                        >
                            批量删除 ({selectedIds.length})
                        </button>
                    )}
                    <button
                        onClick={() => setShowBatchModal(true)}
                        className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg transition"
                    >
                        批量导入
                    </button>
                    <button
                        onClick={() => setShowGenerateModal(true)}
                        className="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-lg transition"
                    >
                        批量生成
                    </button>
                    <button
                        onClick={handleOpenModal}
                        className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition"
                    >
                        + 添加账号
                    </button>
                </div>
            </div>
            <div>
                1000 tokens ≈ $0.0053（近似）。
            </div>

            <div className="bg-white rounded-lg shadow overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                <input
                                    type="checkbox"
                                    checked={accounts.length > 0 && selectedIds.length === accounts.length}
                                    onChange={handleToggleSelectAll}
                                    className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                                />
                            </th>
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
                                产品ID
                            </th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                供应商ID
                            </th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                额度
                            </th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                已用额度
                            </th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                剩余额度
                            </th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                状态
                            </th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                使用状态
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
                                <td colSpan={13} className="px-6 py-8 text-center text-gray-500">
                                    暂无数据
                                </td>
                            </tr>
                        ) : (
                            accounts.map((account) => (
                                <tr key={account.id} className="hover:bg-gray-50">
                                    <td className="px-6 py-4 whitespace-nowrap">
                                        <input
                                            type="checkbox"
                                            checked={selectedIds.includes(account.id)}
                                            onChange={() => handleToggleSelect(account.id)}
                                            className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                                        />
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                        {account.id}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                        {account.account_email}
                                    </td>
                                    <td className="px-6 py-4 text-sm text-gray-500 max-w-md">
                                        <div className="font-mono text-xs break-all whitespace-normal">
                                            {account.token || '-'}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                        {account.product_id}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                        {account.source_id}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                        ${account.balance.toFixed(2)}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                        ${account.used_balance?.toFixed(2) ?? "0.00"}
                                    </td>
                                    <td className={"px-6 py-4 whitespace-nowrap text-sm " + ((account.balance - (account.used_balance ?? 0)) < 0 ? "text-red-600" : "text-gray-900")}>
                                        ${ (account.balance - (account.used_balance ?? 0)).toFixed(2) }
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
                                    <td className="px-6 py-4 whitespace-nowrap">
                                        <span
                                            className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                                account.use_status === 0
                                                    ? 'bg-yellow-100 text-yellow-800'
                                                    : 'bg-blue-100 text-blue-800'
                                            }`}
                                        >
                                            {account.use_status === 0 ? '未使用' : '已使用'}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {new Date(account.create_time).toLocaleString('zh-CN')}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                                        <button
                                            onClick={() => handleEdit(account)}
                                            className="text-indigo-600 hover:text-indigo-900"
                                        >
                                            编辑
                                        </button>
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
                                />
                                <p className="text-xs text-gray-500 mt-1">可选</p>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">账号密码</label>
                                <input
                                    type="password"
                                    value={formData.account_password}
                                    onChange={(e) => setFormData({ ...formData, account_password: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                />
                                <p className="text-xs text-gray-500 mt-1">可选</p>
                            </div>
                            </div>
                            <div>
                                <div className="flex items-center justify-between mb-1">
                                    <label className="block text-sm font-medium text-gray-700">Token</label>
                                    <button
                                        type="button"
                                        onClick={handleRegenerateToken}
                                        className="text-xs text-blue-600 hover:text-blue-800"
                                    >
                                        重新生成
                                    </button>
                                </div>
                        <textarea
                                    value={formData.token}
                                    onChange={(e) => setFormData({ ...formData, token: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none font-mono text-xs"
                                    rows={3}
                                    required={!formData.account_password.trim()}
                                />
                                <p className="text-xs text-gray-500 mt-1">默认使用产品前缀-随机32位</p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">默认额度</label>
                                <input
                                    type="number"
                                    step="0.01"
                                    value={formData.balance}
                                    onChange={(e) =>
                                        setFormData({ ...formData, balance: parseFloat(e.target.value) })
                                    }
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                />
                                <p className="text-xs text-gray-500 mt-1">默认取产品默认额度</p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">状态</label>
                                <select
                                    value={formData.status}
                                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                >
                                    <option value="active">正常</option>
                                    <option value="inactive">禁用</option>
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">产品</label>
                                <select
                                    value={formData.product_id || ''}
                                    onChange={(e) => handleProductChange(Number(e.target.value))}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                >
                                    {productsLoading ? (
                                        <option value="" disabled>加载中...</option>
                                    ) : products.length === 0 ? (
                                        <option value="" disabled>暂无可用产品</option>
                                    ) : (
                                        products.map((product) => (
                                            <option key={product.id} value={product.id}>
                                                {product.product_name} ({product.product_code})
                                            </option>
                                        ))
                                    )}
                                </select>
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

            {/* 批量生成模态框 */}
            {showGenerateModal && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white rounded-lg p-6 w-full max-w-lg">
                        <h3 className="text-lg font-semibold mb-4">批量生成账号</h3>
                        <div className="mb-4 p-3 bg-emerald-50 border border-emerald-200 rounded-lg text-sm text-emerald-800">
                            选择产品与数量后自动生成 Token，邮箱留空，额度取产品默认额度。
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">产品</label>
                                <select
                                    value={batchDefaults.product_id || ''}
                                    onChange={(e) => setBatchDefaults({ ...batchDefaults, product_id: Number(e.target.value) })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                >
                                    {productsLoading ? (
                                        <option value="" disabled>加载中...</option>
                                    ) : products.length === 0 ? (
                                        <option value="" disabled>暂无可用产品</option>
                                    ) : (
                                        products.map((product) => (
                                            <option key={product.id} value={product.id}>
                                                {product.product_name} ({product.product_code})
                                            </option>
                                        ))
                                    )}
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">数量</label>
                                <input
                                    type="number"
                                    min={1}
                                    value={batchCount}
                                    onChange={(e) => setBatchCount(e.target.value)}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                />
                            </div>
                        </div>
                        <div className="flex justify-end space-x-3">
                            <button
                                type="button"
                                onClick={() => setShowGenerateModal(false)}
                                className="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition"
                            >
                                取消
                            </button>
                            <button
                                onClick={handleBatchCreate}
                                className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition"
                            >
                                生成
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* 批量导入模态框 */}
            {showBatchModal && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white rounded-lg p-6 w-full max-w-2xl">
                        <h3 className="text-lg font-semibold mb-4">批量导入/新增账号</h3>
                        <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-800">
                            <p className="font-medium mb-1">快速批量新增</p>
                            <p>选择产品和数量即可自动生成 Token 与默认额度。</p>
                        </div>
                        <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-800">
                            <p className="font-medium mb-2">格式说明:</p>
                            <p className="mb-2">支持多种格式，后端智能识别：</p>
                            <p className="font-mono text-xs mb-1">• 登录账号：email----登录密码：password</p>
                            <p className="font-mono text-xs mb-1">• 邮箱: email   密码: password</p>
                            <p className="font-mono text-xs mb-1">• email----password</p>
                            <p className="font-mono text-xs mb-1">• email  password (多个空格)</p>
                            <p className="text-xs mt-2 text-gray-600">提示：系统会自动跳过视频教程等无关内容</p>
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">默认产品</label>
                                <select
                                    value={batchDefaults.product_id || ''}
                                    onChange={(e) => setBatchDefaults({ ...batchDefaults, product_id: Number(e.target.value) })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                >
                                    {productsLoading ? (
                                        <option value="" disabled>加载中...</option>
                                    ) : products.length === 0 ? (
                                        <option value="" disabled>暂无可用产品</option>
                                    ) : (
                                        products.map((product) => (
                                            <option key={product.id} value={product.id}>
                                                {product.product_name}
                                            </option>
                                        ))
                                    )}
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">默认状态</label>
                                <select
                                    value={batchDefaults.status}
                                    onChange={(e) => setBatchDefaults({ ...batchDefaults, status: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                >
                                    <option value="active">正常</option>
                                    <option value="inactive">禁用</option>
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">数量</label>
                                <input
                                    type="number"
                                    min={1}
                                    value={batchCount}
                                    onChange={(e) => setBatchCount(e.target.value)}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                />
                            </div>
                        </div>
                        <div className="flex justify-end mb-4">
                            <button
                                onClick={handleBatchCreate}
                                className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition"
                            >
                                批量新增
                            </button>
                        </div>
                        <div className="mb-4">
                            <label className="block text-sm font-medium text-gray-700 mb-1">字段顺序（逗号分隔，可选）</label>
                            <input
                                type="text"
                                value={batchFieldKeys}
                                onChange={(e) => setBatchFieldKeys(e.target.value)}
                                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                placeholder="account_email,account_password"
                            />
                            <p className="text-xs text-gray-500 mt-1">通常无需修改，后端会智能识别格式</p>
                        </div>                        <textarea
                            value={batchText}
                            onChange={(e) => setBatchText(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none font-mono text-xs"
                            rows={10}
                            placeholder="官网登录账号,支持Widsurf 支持claude4,  偶尔支持 claude4.5&#10;基础使用视频教程地址：https://fcn0uln516co.feishu.cn/wiki/EYpiwWZstiYpCgkkYRPcjdDKnic&#10;&#10;邮箱: user@example.com   密码: password123"
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

            {/* 编辑账号模态框 */}
            {showEditModal && editingAccount && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white rounded-lg p-6 w-full max-w-md">
                        <h3 className="text-lg font-semibold mb-4">编辑账号 (ID: {editingAccount.id})</h3>
                        <form onSubmit={handleUpdate} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">邮箱</label>
                                <input
                                    type="email"
                                    value={editFormData.account_email}
                                    onChange={(e) => setEditFormData({ ...editFormData, account_email: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                />
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">账号密码</label>
                                <input
                                    type="text"
                                    value={editFormData.account_password}
                                    onChange={(e) => setEditFormData({ ...editFormData, account_password: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    placeholder="留空表示不修改"
                                />
                                <p className="text-xs text-gray-500 mt-1">留空表示不修改</p>
                            </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">Token</label>
                                <textarea
                                    value={editFormData.token}
                                    onChange={(e) => setEditFormData({ ...editFormData, token: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none font-mono text-xs bg-gray-50"
                                    rows={3}
                                    required={!editFormData.account_password.trim()}

                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">总额度</label>
                                <input
                                    type="number"
                                    step="0.01"
                                    value={editFormData.balance}
                                    onChange={(e) =>
                                        setEditFormData({ ...editFormData, balance: parseFloat(e.target.value) || 0 })
                                    }
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">已用额度</label>
                                <input
                                    type="number"
                                    step="0.01"
                                    value={editFormData.used_balance}
                                    onChange={(e) =>
                                        setEditFormData({ ...editFormData, used_balance: parseFloat(e.target.value) || 0 })
                                    }
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                />
                                <p className="text-xs text-gray-500 mt-1">
                                    剩余额度: ${(editFormData.balance - editFormData.used_balance).toFixed(2)}
                                    {editFormData.balance - editFormData.used_balance < 0 && (
                                        <span className="text-red-600 ml-2">（欠费）</span>
                                    )}
                                </p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">状态</label>
                                <select
                                    value={editFormData.status}
                                    onChange={(e) => setEditFormData({ ...editFormData, status: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                >
                                    <option value="active">正常</option>
                                    <option value="inactive">禁用</option>
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">使用状态</label>
                                <div className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-gray-50 text-gray-700">
                                    {editFormData.use_status === 0 ? '未使用' : '已使用'}
                                </div>
                                <p className="text-xs text-gray-500 mt-1">使用状态不可编辑</p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">产品</label>
                                <select
                                    value={editFormData.product_id}
                                    onChange={(e) => setEditFormData({ ...editFormData, product_id: Number(e.target.value) })}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
                                    required
                                >
                                    {products.map((product) => (
                                        <option key={product.id} value={product.id}>
                                            {product.product_name} ({product.product_code})
                                        </option>
                                    ))}
                                </select>
                            </div>
                            <div className="flex justify-end space-x-3 pt-4">
                                <button
                                    type="button"
                                    onClick={() => {
                                        setShowEditModal(false);
                                        setEditingAccount(null);
                                    }}
                                    className="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition"
                                >
                                    取消
                                </button>
                                <button
                                    type="submit"
                                    className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition"
                                >
                                    保存
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}
        </div>
    );
}






