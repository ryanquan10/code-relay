import { useState, useEffect } from 'react';
import { sourceAPI, Source } from '../api/client';

type SourceFormState = {
  source_name: string;
  upstream_url: string;
  upstream_token: string;
  source_type: string;
  config: string;
  priority: number;
  auto_rental: boolean;
  status: number;
  remark: string;
};

const createDefaultFormData = (): SourceFormState => ({
  source_name: '',
  upstream_url: '',
  upstream_token: '',
  source_type: '',
  config: '',
  priority: 1,
  auto_rental: false,
  status: 1,
  remark: '',
});

const formatConfigText = (config: Source['config']) => {
  if (config == null) return '';
  if (typeof config === 'string') {
    const trimmed = config.trim();
    if (!trimmed) return '';
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return config;
    }
  }
  try {
    return JSON.stringify(config, null, 2);
  } catch {
    return String(config);
  }
};

const formatConfigPreview = (config: Source['config']) => {
  if (config == null) return '-';
  const raw = typeof config === 'string' ? config : JSON.stringify(config);
  const singleLine = raw.replace(/\s+/g, ' ').trim();
  if (!singleLine) return '-';
  return singleLine.length > 80 ? `${singleLine.slice(0, 77)}...` : singleLine;
};

const formatConfigTitle = (config: Source['config']) => {
  if (config == null) return '';
  if (typeof config === 'string') return config;
  return JSON.stringify(config, null, 2);
};

const parseConfigText = (text: string): Source['config'] => {
  const trimmed = text.trim();
  if (!trimmed) return null;
  return JSON.parse(trimmed) as Source['config'];
};

const formatDateTime = (value?: string) => {
  if (!value) return '-';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString('zh-CN');
};

export default function SourcesTab() {
  const [sources, setSources] = useState<Source[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingSource, setEditingSource] = useState<Source | null>(null);
  const [formData, setFormData] = useState<SourceFormState>(createDefaultFormData());

  useEffect(() => {
    loadSources();
  }, []);

  const loadSources = async () => {
    try {
      setLoading(true);
      const data = await sourceAPI.list();
      setSources(data);
    } catch (error) {
      console.error('加载供应商失败:', error);
      alert('加载供应商失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    let parsedConfig: Source['config'] = null;
    try {
      parsedConfig = parseConfigText(formData.config);
    } catch (error) {
      console.error('配置 JSON 解析失败:', error);
      alert('配置必须是有效的 JSON');
      return;
    }

    const payload: Partial<Source> = {
      source_name: formData.source_name,
      upstream_url: formData.upstream_url.trim() || null,
      upstream_token: formData.upstream_token.trim() || null,
      source_type: formData.source_type,
      config: parsedConfig,
      priority: formData.priority,
      auto_rental: formData.auto_rental,
      status: formData.status,
      remark: formData.remark.trim() || null,
    };

    try {
      if (editingSource) {
        await sourceAPI.update(editingSource.id, payload);
        alert('更新成功');
      } else {
        await sourceAPI.create(payload);
        alert('创建成功');
      }
      setShowModal(false);
      setFormData(createDefaultFormData());
      setEditingSource(null);
      loadSources();
    } catch (error) {
      console.error('保存失败:', error);
      alert('保存失败');
    }
  };

  const handleEdit = (source: Source) => {
    setEditingSource(source);
    setFormData({
      source_name: source.source_name,
      upstream_url: source.upstream_url ?? '',
      upstream_token: source.upstream_token ?? '',
      source_type: source.source_type,
      config: formatConfigText(source.config),
      priority: source.priority,
      auto_rental: source.auto_rental,
      status: source.status,
      remark: source.remark ?? '',
    });
    setShowModal(true);
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个账号供应吗？')) return;
    try {
      await sourceAPI.delete(id);
      alert('删除成功');
      loadSources();
    } catch (error) {
      console.error('删除失败:', error);
      alert('删除失败');
    }
  };

  const handleCloseModal = () => {
    setShowModal(false);
    setEditingSource(null);
    setFormData(createDefaultFormData());
  };

  if (loading) {
    return <div className="text-center py-8">加载中...</div>;
  }

  return (
    <div>
      <div className="mb-6 flex justify-between items-center">
        <h3 className="text-xl font-semibold text-gray-800">账号供应列表</h3>
        <button
          onClick={() => setShowModal(true)}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition"
        >
          + 添加账号供应
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
                供应商名称
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                上流地址
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                上流Token
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                类型
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                配置
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                优先级
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                自动租用
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                状态
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                备注
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
            {sources.length === 0 ? (
              <tr>
                <td colSpan={13} className="px-6 py-8 text-center text-gray-500">
                  暂无数据
                </td>
              </tr>
            ) : (
              sources.map((source) => (
                <tr key={source.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{source.id}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {source.source_name}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={source.upstream_url ?? ''}
                  >
                    {source.upstream_url || '-'}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={source.upstream_token ?? ''}
                  >
                    {source.upstream_token || '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {source.source_type}
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={formatConfigTitle(source.config)}
                  >
                    {formatConfigPreview(source.config)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {source.priority}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        source.auto_rental
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      {source.auto_rental ? '是' : '否'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        source.status === 1
                          ? 'bg-green-100 text-green-800'
                          : 'bg-red-100 text-red-800'
                      }`}
                    >
                      {source.status === 1 ? '启用' : '禁用'}
                    </span>
                  </td>
                  <td
                    className="px-6 py-4 text-sm text-gray-500 max-w-xs truncate"
                    title={source.remark ?? ''}
                  >
                    {source.remark || '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDateTime(source.create_time)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDateTime(source.update_time)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                    <button
                      onClick={() => handleEdit(source)}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      编辑
                    </button>
                    <button
                      onClick={() => handleDelete(source.id)}
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
              {editingSource ? '编辑账号供应' : '添加账号供应'}
            </h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">供应商名称 *</label>
                  <input
                    type="text"
                    value={formData.source_name}
                    onChange={(e) => setFormData({ ...formData, source_name: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">上流地址</label>
                  <input
                    type="text"
                    value={formData.upstream_url}
                    onChange={(e) => setFormData({ ...formData, upstream_url: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="https://api.example.com"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">上流Token</label>
                  <input
                    type="text"
                    value={formData.upstream_token}
                    onChange={(e) => setFormData({ ...formData, upstream_token: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="上流 Token / API Key"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">类型 *</label>
                  <input
                    type="text"
                    value={formData.source_type}
                    onChange={(e) => setFormData({ ...formData, source_type: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    required
                    placeholder="如: OpenAI、Claude"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">配置 (JSON)</label>
                <textarea
                  value={formData.config}
                  onChange={(e) => setFormData({ ...formData, config: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none font-mono text-xs"
                  rows={5}
                  placeholder='{"base_url":"https://api.example.com","api_key":"***"}'
                />
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">优先级</label>
                  <input
                    type="number"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: Number(e.target.value) })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    min={1}
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
                      checked={formData.auto_rental}
                      onChange={(e) => setFormData({ ...formData, auto_rental: e.target.checked })}
                      className="mr-2"
                    />
                    <span className="text-sm font-medium text-gray-700">自动租用</span>
                  </label>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">备注</label>
                <textarea
                  value={formData.remark}
                  onChange={(e) => setFormData({ ...formData, remark: e.target.value })}
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
                  {editingSource ? '更新' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
