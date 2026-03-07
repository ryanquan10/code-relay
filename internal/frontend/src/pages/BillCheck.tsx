import { useEffect, useMemo, useState } from 'react';
import { publicMetaAPI, publicUsageAPI, PublicUsageItem } from '../api/client';

export default function BillCheck() {
  const [token, setToken] = useState<string>('');
  const [sourceTypes, setSourceTypes] = useState<string[]>([]);
  const [selectedType, setSelectedType] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [items, setItems] = useState<PublicUsageItem[]>([]);

  useEffect(() => {
    publicMetaAPI
      .accountSourceTypes()
      .then((res) => setSourceTypes(res.types || []))
      .catch(() => setSourceTypes([]));
  }, []);

  const canQuery = useMemo(() => token.trim().length > 0 && selectedType.trim().length > 0, [token, selectedType]);

  const onQuery = async () => {
    if (!canQuery) return;
    setLoading(true);
    setError(null);
    try {
      const res = await publicUsageAPI.list(token.trim(), selectedType.trim());
      setItems(res.items || []);
    } catch (e: any) {
      setError(e?.response?.data?.error || e?.message || '查询失败');
      setItems([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-5xl mx-auto px-6 py-8">
        <h1 className="text-2xl font-bold mb-2">使用额度查询</h1>
        <p className="text-gray-600 mb-6">输入您的 token AI类型，查看使用流水（consume、tokens、时间）。</p>

        <div className="bg-white rounded shadow p-4 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div>
              <label className="block text-sm text-gray-700 mb-1">客户 Token</label>
              <input
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="粘贴您的 token"
                className="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:border-blue-400"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-700 mb-1">AI类型</label>
              <select
                value={selectedType}
                onChange={(e) => setSelectedType(e.target.value)}
                className="w-full border rounded px-3 py-2 bg-white focus:outline-none focus:ring focus:border-blue-400"
              >
                <option value="">选择类型</option>
                {sourceTypes.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-end">
              <button
                onClick={onQuery}
                disabled={!canQuery || loading}
                className="w-full md:w-auto px-4 py-2 bg-blue-600 text-white rounded disabled:opacity-60"
              >
                {loading ? '查询中...' : '查询'}
              </button>
            </div>
          </div>

          {error && (
            <div className="text-sm text-red-600">{error}</div>
          )}
        </div>

        <div className="mt-6 bg-white rounded shadow overflow-hidden">
          <div className="px-4 py-3 border-b text-gray-700 font-medium">使用流水</div>
          <div className="overflow-x-auto">
            <table className="min-w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Consume</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tokens</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Model</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Create Time</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Update Time</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200 text-sm">
                {items.length === 0 && !loading ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-8 text-center text-gray-500">
                      暂无数据
                    </td>
                  </tr>
                ) : (
                  items.map((it, idx) => (
                    <tr key={idx}>
                      <td className="px-6 py-3 text-gray-800">{it.consume?.toFixed?.(4) ?? it.consume}</td>
                      <td className="px-6 py-3 text-gray-800">{it.tokens}</td>
                      <td className="px-6 py-3 text-gray-800">{it.model ?? '-'}</td>
                      <td className="px-6 py-3 text-gray-600">{new Date(it.create_time).toLocaleString()}</td>
                      <td className="px-6 py-3 text-gray-600">{new Date(it.update_time).toLocaleString()}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

        </div>
      </div>
    </div>
  );
}
