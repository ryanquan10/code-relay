import { useState, useEffect } from 'react';
import { productAPI, Product, ProductSource, sourceAPI, Source } from '../api/client';

type ProductFormState = {
  product_code: string;
  product_name: string;
  account_type: string;
  category: string;
  icon: string;
  image_url: string;
  down_stream_url: string;
  description: string;
  price: number;
  original_price: number | null;
  sales_count: number;
  contact_info: string;
  usage_instruction: string;
  validity_days: number;
  shared_limit: number;
  sources: ProductSource[];
  cost_price: number;
  default_balance: number;
  original_balance: number;
  stock: number;
  auto_delivery: boolean;
  sort_order: number;
  status: number;
  platforms_json: string;
};

const buildDefaultSourceSelection = (availableSources: Source[]): ProductSource[] => {
  if (availableSources.length === 0) {
    return [];
  }
  return [{ source_id: availableSources[0].id, weight: 1 }];
};

const getDefaultAccountType = (
  availableSources: Source[],
  selectedSources: ProductSource[]
): string => {
  if (availableSources.length === 0 || selectedSources.length === 0) {
    return '';
  }
  const selectedIds = new Set(selectedSources.map((item) => item.source_id));
  const matchedSource = availableSources.find((source) => selectedIds.has(source.id));
  return matchedSource ? matchedSource.source_type : '';
};

const createDefaultFormData = (availableSources: Source[] = []): ProductFormState => {
  const defaultSources = buildDefaultSourceSelection(availableSources);
  return {
    product_code: '',
    product_name: '',
    account_type: getDefaultAccountType(availableSources, defaultSources),
    category: '',
    icon: '',
    image_url: '',
    down_stream_url: '',
    description: '',
    price: 0,
    original_price: null,
    sales_count: 0,
    contact_info: '',
    usage_instruction: '',
    validity_days: 9,
    shared_limit: 0,
    sources: defaultSources,
    cost_price: 0,
    default_balance: 0,
    original_balance: 0,
    stock: 0,
    auto_delivery: false,
    sort_order: 0,
    status: 1,
    platforms_json: '[]',
  };
};

const formatDateTime = (value?: string | null) => {
  if (!value) return '-';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString('zh-CN');
};

const formatOptionalText = (value?: string | null) => {
  if (!value) return '-';
  const trimmed = value.trim();
  return trimmed === '' ? '-' : trimmed;
};

const formatMoney = (value?: number | null) => {
  if (value == null) return '-';
  if (Number.isNaN(value)) return '-';
  return `¥${value.toFixed(2)}`;
};

export default function ProductsTab() {
  const [products, setProducts] = useState<Product[]>([]);
  const [sources, setSources] = useState<Source[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);
  const [formData, setFormData] = useState<ProductFormState>(createDefaultFormData());
  const sourceLookup = new Map(sources.map((source) => [source.id, source]));

  const formatSourceList = (mappings?: ProductSource[]) => {
    if (!mappings || mappings.length === 0) {
      return '-';
    }
    return mappings
      .map((mapping) => {
        const source = sourceLookup.get(mapping.source_id);
        const label = source ? source.source_name : `ID ${mapping.source_id}`;
        return `${label} (权重 ${mapping.weight})`;
      })
      .join(', ');
  };

  useEffect(() => {
    loadProducts();
    loadSources();
  }, []);

  const loadProducts = async () => {
    try {
      setLoading(true);
      const data = await productAPI.list();
      setProducts(data);
    } catch (error) {
      console.error('加载产品失败:', error);
      alert('加载产品失败');
    } finally {
      setLoading(false);
    }
  };

  const loadSources = async () => {
    try {
      const data = await sourceAPI.list();
      setSources(data);
      setFormData((prev) => {
        if (prev.sources.length > 0 || data.length === 0) {
          return prev;
        }
        const nextSources = buildDefaultSourceSelection(data);
        const nextAccountType = prev.account_type.trim()
          ? prev.account_type
          : getDefaultAccountType(data, nextSources);
        return { ...prev, sources: nextSources, account_type: nextAccountType };
      });
    } catch (error) {
      console.error('加载供应商失败:', error);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (formData.sources.length === 0) {
        alert('请至少选择一个账号供应');
        return;
      }
      if (formData.sources.some((item) => item.source_id <= 0 || item.weight <= 0)) {
        alert('账号供应权重必须大于 0');
        return;
      }
      let platformsParsed: any = [];
      try {
        platformsParsed = formData.platforms_json.trim() ? JSON.parse(formData.platforms_json) : [];
      } catch (err) {
        alert('平台JSON格式错误，请检查');
        return;
      }
      const payload: Partial<Product> = {
        product_code: formData.product_code.trim(),
        product_name: formData.product_name.trim(),
        account_type: formData.account_type.trim(),
        category: formData.category.trim(),
        icon: formData.icon.trim(),
        image_url: formData.image_url.trim(),
        down_stream_url: formData.down_stream_url.trim(),
        description: formData.description.trim(),
        price: formData.price,
        original_price: formData.original_price,
        sales_count: formData.sales_count,
        contact_info: formData.contact_info.trim(),
        usage_instruction: formData.usage_instruction.trim(),
        validity_days: formData.validity_days,
        shared_limit: formData.shared_limit,
        cost_price: formData.cost_price,
        default_balance: formData.default_balance,
        original_balance: formData.original_balance,
        stock: formData.stock,
        auto_delivery: formData.auto_delivery,
        sort_order: formData.sort_order,
        status: formData.status,
        sources: formData.sources,
        platforms: platformsParsed,
      };
      if (editingProduct) {
        await productAPI.update(editingProduct.id, payload);
        alert('更新成功');
      } else {
        await productAPI.create(payload);
        alert('创建成功');
      }
      setShowModal(false);
      resetForm();
      loadProducts();
    } catch (error) {
      console.error('保存失败:', error);
      alert('保存失败');
    }
  };

  const handleEdit = (product: Product) => {
    setEditingProduct(product);
    setFormData({
      product_code: product.product_code,
      product_name: product.product_name,
      account_type: product.account_type,
      category: product.category || '',
      icon: product.icon || '',
      image_url: product.image_url || '',
      down_stream_url: product.down_stream_url || '',
      description: product.description || '',
      price: product.price,
      original_price: product.original_price ?? null,
      sales_count: product.sales_count ?? 0,
      contact_info: product.contact_info || '',
      usage_instruction: product.usage_instruction || '',
      validity_days: product.validity_days,
      shared_limit: product.shared_limit,
      sources: product.sources && product.sources.length > 0
        ? product.sources
        : buildDefaultSourceSelection(sources),
      cost_price: product.cost_price ?? 0,
      default_balance: product.default_balance ?? 0,
      original_balance: product.original_balance ?? 0,
      stock: product.stock ?? 0,
      auto_delivery: product.auto_delivery,
      sort_order: product.sort_order,
      status: product.status,
      platforms_json: JSON.stringify((product as any).platforms ?? [], null, 2),
    });
    setShowModal(true);
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个产品吗？')) return;
    try {
      await productAPI.delete(id);
      alert('删除成功');
      loadProducts();
    } catch (error) {
      console.error('删除失败:', error);
      alert('删除失败');
    }
  };

  const resetForm = () => {
    setFormData(createDefaultFormData(sources));
    setEditingProduct(null);
  };

  const handleCloseModal = () => {
    setShowModal(false);
    resetForm();
  };

  const isSourceSelected = (sourceId: number) =>
    formData.sources.some((item) => item.source_id === sourceId);

  const handleSourceToggle = (sourceId: number) => {
    setFormData((prev) => {
      const isSelected = prev.sources.some((item) => item.source_id === sourceId);
      const nextSources = isSelected
        ? prev.sources.filter((item) => item.source_id !== sourceId)
        : [...prev.sources, { source_id: sourceId, weight: 1 }];
      const currentDerived = getDefaultAccountType(sources, prev.sources);
      const nextDerived = getDefaultAccountType(sources, nextSources);
      const hasManualType = prev.account_type.trim() !== '' && prev.account_type !== currentDerived;
      return {
        ...prev,
        sources: nextSources,
        account_type: hasManualType ? prev.account_type : nextDerived,
      };
    });
  };

  const handleSourceWeightChange = (sourceId: number, weight: number) => {
    setFormData((prev) => ({
      ...prev,
      sources: prev.sources.map((item) =>
        item.source_id === sourceId ? { ...item, weight } : item
      ),
    }));
  };

  if (loading) {
    return <div className="text-center py-8">加载中...</div>;
  }

  return (
    <div>
      <div className="mb-6 flex justify-between items-center">
        <h3 className="text-xl font-semibold text-gray-800">产品列表</h3>
        <button
          onClick={() => setShowModal(true)}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition"
        >
          + 添加产品
        </button>
      </div>

      <div className="bg-white rounded-lg shadow overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                ID
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                产品代码
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                产品名称
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                账号类型
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                分类
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                价格
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                原价
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                成本价
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                初始余额
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                默认余额
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                库存
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                销量
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                有效天数
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                共享限制
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                账号供应
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                自动发货
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                排序
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                状态
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                下游地址
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                图标
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                图片
              </th>

              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                描述
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                联系方式
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                使用说明
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                创建时间
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                更新时间
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                操作
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {products.length === 0 ? (
              <tr>
                <td colSpan={27} className="px-6 py-8 text-center text-gray-500">
                  暂无数据
                </td>
              </tr>
            ) : (
              products.map((product) => (
                <tr key={product.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{product.id}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {product.product_code}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {product.product_name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {product.account_type}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatOptionalText(product.category)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {formatMoney(product.price)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatMoney(product.original_price)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatMoney(product.cost_price)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatMoney(product.original_balance)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatMoney(product.default_balance)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {product.stock}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {product.sales_count}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {product.validity_days}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {product.shared_limit}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatSourceList(product.sources)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        product.auto_delivery
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      {product.auto_delivery ? '是' : '否'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {product.sort_order}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        product.status === 1
                          ? 'bg-green-100 text-green-800'
                          : 'bg-red-100 text-red-800'
                      }`}
                    >
                      {product.status === 1 ? '启用' : '禁用'}
                    </span>
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={product.icon ?? ''}
                  >
                    {formatOptionalText(product.icon)}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={product.image_url ?? ''}
                  >
                    {formatOptionalText(product.image_url)}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={product.down_stream_url ?? ''}
                  >
                    {formatOptionalText(product.down_stream_url)}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={product.description ?? ''}
                  >
                    {formatOptionalText(product.description)}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={product.contact_info ?? ''}
                  >
                    {formatOptionalText(product.contact_info)}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={product.usage_instruction ?? ''}
                  >
                    {formatOptionalText(product.usage_instruction)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDateTime(product.create_time)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDateTime(product.update_time)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                    <button
                      onClick={() => handleEdit(product)}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      编辑
                    </button>
                    <button
                      onClick={() => handleDelete(product.id)}
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

      {/* 添加/编辑模态框 */}
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 overflow-y-auto">
          <div className="bg-white rounded-lg p-6 w-full max-w-2xl m-4 max-h-[90vh] overflow-y-auto">
            <h3 className="text-lg font-semibold mb-4">
              {editingProduct ? '编辑产品' : '添加产品'}
            </h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">账号供应</label>
                {sources.length === 0 ? (
                  <p className="text-sm text-gray-500">暂无可用账号供应</p>
                ) : (
                  <div className="space-y-2">
                    {sources.map((source) => {
                      const selected = formData.sources.find((item) => item.source_id === source.id);
                      const weightValue = selected?.weight ?? 1;
                      return (
                        <div key={source.id} className="flex items-center gap-3">
                          <input
                            type="checkbox"
                            checked={isSourceSelected(source.id)}
                            onChange={() => handleSourceToggle(source.id)}
                          />
                          <span className="text-sm text-gray-700">
                            {source.source_name} ({source.status === 1 ? '启用' : '禁用'})
                          </span>
                          <input
                            type="number"
                            min={1}
                            value={weightValue}
                            onChange={(e) => {
                              const nextWeight = Number(e.target.value);
                              handleSourceWeightChange(
                                source.id,
                                Number.isFinite(nextWeight) && nextWeight > 0 ? nextWeight : 1
                              );
                            }}
                            disabled={!selected}
                            className="w-28 px-2 py-1 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                          />
                          <span className="text-xs text-gray-500">权重</span>
                        </div>
                      );
                    })}
                  </div>
                )}
                <p className="text-xs text-gray-500 mt-1">可多选，填写权重</p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">产品代码 *</label>
                  <input
                    type="text"
                    value={formData.product_code}
                    onChange={(e) => setFormData({ ...formData, product_code: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">产品名称 *</label>
                  <input
                    type="text"
                    value={formData.product_name}
                    onChange={(e) => setFormData({ ...formData, product_name: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    required
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">账号类型</label>
                  <input
                    type="text"
                    value={formData.account_type}
                    onChange={(e) => setFormData({ ...formData, account_type: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="如: ChatGPT, Claude"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">分类</label>
                  <input
                    type="text"
                    value={formData.category}
                    onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="如: AI工具"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">下游地址</label>
                <input
                    type="text"
                    value={formData.down_stream_url}
                    onChange={(e) => setFormData({ ...formData, down_stream_url: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="下游地址 URL"
                />
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">图标</label>
                  <input
                    type="text"
                    value={formData.icon}
                    onChange={(e) => setFormData({ ...formData, icon: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="图标URL或标识"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">图片</label>
                  <input
                    type="text"
                    value={formData.image_url}
                    onChange={(e) => setFormData({ ...formData, image_url: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="图片URL"
                  />
                </div>
              </div>



              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">价格 *</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.price}
                    onChange={(e) => setFormData({ ...formData, price: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">原价</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.original_price ?? ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        original_price: e.target.value === '' ? null : Number(e.target.value),
                      })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">成本价</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.cost_price}
                    onChange={(e) => setFormData({ ...formData, cost_price: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">初始余额</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.original_balance}
                    onChange={(e) => setFormData({ ...formData, original_balance: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">默认余额</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.default_balance}
                    onChange={(e) => setFormData({ ...formData, default_balance: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">库存</label>
                  <input
                    type="number"
                    value={formData.stock}
                    onChange={(e) => setFormData({ ...formData, stock: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">销量</label>
                  <input
                    type="number"
                    value={formData.sales_count}
                    onChange={(e) => setFormData({ ...formData, sales_count: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">有效天数</label>
                  <input
                    type="number"
                    value={formData.validity_days}
                    onChange={(e) => setFormData({ ...formData, validity_days: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">共享限制</label>
                  <input
                    type="number"
                    value={formData.shared_limit}
                    onChange={(e) => setFormData({ ...formData, shared_limit: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">排序</label>
                  <input
                    type="number"
                    value={formData.sort_order}
                    onChange={(e) => setFormData({ ...formData, sort_order: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">状态</label>
                  <select
                    value={formData.status}
                    onChange={(e) => setFormData({ ...formData, status: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  >
                    <option value={1}>启用</option>
                    <option value={0}>禁用</option>
                  </select>
                </div>
                <div className="flex items-center pt-7">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={formData.auto_delivery}
                      onChange={(e) => setFormData({ ...formData, auto_delivery: e.target.checked })}
                      className="mr-2"
                    />
                    <span className="text-sm font-medium text-gray-700">自动发货</span>
                  </label>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">描述</label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  rows={3}
                />
              </div>

                <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">平台映射(JSON)</label>
                    <textarea
                        value={formData.platforms_json}
                        onChange={(e) => setFormData({ ...formData, platforms_json: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none font-mono"
                        rows={8}
                        placeholder='[{"platform":"xianyu","product_code":"ABC"},{"platform":"douyin","product_code":"DEF"}]'
                    />
                    <p className="mt-1 text-xs text-gray-500">填写原生 JSON 数组，例如: {`[{   "product_code": "",   "platform": ""},]`}。保存时将原样提交。</p>
                </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">使用说明</label>
                <textarea
                  value={formData.usage_instruction}
                  onChange={(e) => setFormData({ ...formData, usage_instruction: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  rows={3}
                />
              </div>



              <div className="flex justify-end space-x-3 pt-4">
                <button
                  type="button"
                  onClick={handleCloseModal}
                  className="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition"
                >
                  取消
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition"
                >
                  {editingProduct ? '更新' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}


