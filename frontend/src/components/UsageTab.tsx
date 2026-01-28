import { useState, useEffect } from 'react';
import { usageAPI, Usage, usageDailyAPI, DailyUsage, DailyUsageSummary } from '../api/client';

export default function UsageTab() {
  const [usages, setUsages] = useState<Usage[]>([]);
  const [page, setPage] = useState<number>(1);
  const [size] = useState<number>(100);
  const [total, setTotal] = useState<number>(0);
  const [dailyUsages, setDailyUsages] = useState<DailyUsage[]>([]);
  const [dailySummary, setDailySummary] = useState<DailyUsageSummary | null>(null);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [searchKey, setSearchKey] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  useEffect(() => {
    if (!searchKey.trim()) {
      loadUsages();
      loadStats();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  const loadUsages = async () => {
    try {
      setLoading(true);
      const data = await usageAPI.list({ page, size });
      setUsages(data.items || []);
      setTotal(data.total || 0);
    } catch (error) {
      console.error('加载使用量失败:', error);
      alert('加载使用量失败');
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const data = await usageAPI.getStats(startDate, endDate);
      setStats(data);
    } catch (error) {
      console.error('加载统计失败:', error);
    }
  };

  const handleDailyQuery = async () => {
    if (!searchKey.trim()) {
      alert('请输入 Customer Key');
      return;
    }
    try {
      setLoading(true);
      if (startDate && endDate) {
        const resp = await usageDailyAPI.byRange(searchKey.trim(), startDate, endDate);
        setDailyUsages(resp.data.daily_usages || []);
        setDailySummary(resp.data.summary || null);
      } else {
        const resp = await usageDailyAPI.byDays(searchKey.trim(), 7);
        setDailyUsages(resp.data.daily_usages || []);
        setDailySummary(resp.data.summary || null);
      }
    } catch (error) {
      console.error('每日统计查询失败:', error);
      alert('每日统计查询失败');
      setDailyUsages([]);
      setDailySummary(null);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!searchKey.trim()) {
      setPage(1);
      loadUsages();
      loadStats();
      return;
    }
    try {
      setLoading(true);
      const resp = await usageAPI.getByToken(searchKey.trim());
      setUsages(resp.details || []);
    } catch (error) {
      console.error('查询失败:', error);
      alert('查询失败');
      setUsages([]);
    } finally {
      setLoading(false);
    }
  };

  const handleReset = () => {
    setPage(1);
    setSearchKey('');
    setStartDate('');
    setEndDate('');
    loadUsages();
    loadStats();
  };

  if (loading && usages.length === 0) {
    return <div className="text-center py-8">加载中...</div>;
  }

  return (
    <div>
      {/* 统计卡片 */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm text-gray-600 mb-1">总消费金额</div>
            <div className="text-3xl font-bold text-blue-600">
              ${stats.total_consume?.toFixed(4) || '0.0000'}
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm text-gray-600 mb-1">总 Token 数</div>
            <div className="text-3xl font-bold text-green-600">
              {stats.total_tokens?.toLocaleString() || '0'}
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm text-gray-600 mb-1">使用记录数</div>
            <div className="text-3xl font-bold text-purple-600">{stats?.record_count ?? total}</div>
          </div>
        </div>
      )}

      {/* 搜索栏 */}
      <div className="mb-6 bg-white rounded-lg shadow p-4">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Customer Key</label>
            <input
              type="text"
              placeholder="输入 Customer Key"
              value={searchKey}
              onChange={(e) => setSearchKey(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">开始日期</label>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">结束日期</label>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none"
            />
          </div>
          <div className="flex items-end gap-2">
            <button
              onClick={handleSearch}
              className="flex-1 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition"
            >
              搜索
            </button>
            <button
              onClick={handleDailyQuery}
              className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg transition"
            >
              每日统计
            </button>
            <button
              onClick={handleReset}
              className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition"
            >
              重置
            </button>
          </div>
        </div>
      </div>

      {/* 每日统计 */}
      {dailyUsages.length > 0 && (
        <div className="bg-white rounded-lg shadow overflow-hidden mb-6">
          <div className="p-4 border-b">
            <div className="flex items-center justify-between">
              <div className="font-medium">每日统计</div>
              {dailySummary && (
                <div className="text-sm text-gray-600">
                  合计 {dailySummary.days} 天，消费 ${dailySummary.total_consume.toFixed(4)}（{dailySummary.total_records} 条）
                </div>
              )}
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">日期</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">消费(USD)</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">记录数</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {dailyUsages.map((d) => (
                  <tr key={d.date} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{d.date}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-blue-600">${d.total_consume.toFixed(4)}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{d.record_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* 使用量列表 */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Customer Key</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tokens</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">消费金额</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">日期</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {usages.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-gray-500">暂无数据</td>
                </tr>
              ) : (
                usages.map((u) => (
                  <tr key={u.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{u.id}</td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      <span className="font-mono text-xs">{u.customer_key}</span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{u.tokens.toLocaleString()}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-blue-600">${u.consume.toFixed(6)}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{u.date}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(u.created_at).toLocaleString('zh-CN')}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {(!searchKey.trim()) && (
        <div className="flex items-center justify-between mt-4">
          <div className="text-sm text-gray-600">第 {page} / {Math.max(1, Math.ceil(total / size))} 页，每页 {size}，共 {total} 条</div>
          <div className="space-x-2">
            <button className="px-4 py-2 bg-gray-100 rounded disabled:opacity-50" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>上一页</button>
            <button className="px-4 py-2 bg-gray-100 rounded disabled:opacity-50" disabled={page >= Math.ceil(total / size)} onClick={() => setPage((p) => p + 1)}>下一页</button>
          </div>
        </div>
      )}
    </div>
  );
}
