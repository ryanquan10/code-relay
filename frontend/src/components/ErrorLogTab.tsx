import { useEffect, useState } from 'react';
import { errorLogAPI, UpstreamErrorLog } from '../api/client';

export default function ErrorLogTab() {
  const [logs, setLogs] = useState<UpstreamErrorLog[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);

  const loadLogs = async () => {
    try {
      setLoading(true);
      const data = await errorLogAPI.list();
      // 按创建时间倒序
      const sorted = [...(data || [])].sort((a, b) => {
        const ta = new Date(a.created_at).getTime();
        const tb = new Date(b.created_at).getTime();
        return tb - ta;
      });
      setLogs(sorted);
    } catch (e) {
      console.error('加载错误日志失败:', e);
      alert('加载错误日志失败');
      setLogs([]);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = async () => {
    try {
      setRefreshing(true);
      await loadLogs();
    } finally {
      setRefreshing(false);
    }
  };

  useEffect(() => {
    loadLogs();
  }, []);

  if (loading && logs.length === 0) {
    return <div className="text-center py-8">加载中...</div>;
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div className="text-sm text-gray-600">共 {logs.length} 条记录</div>
        <button
          onClick={handleRefresh}
          disabled={refreshing}
          className={`px-4 py-2 rounded-lg text-white transition ${refreshing ? 'bg-gray-400' : 'bg-blue-600 hover:bg-blue-700'}`}
        >
          {refreshing ? '刷新中...' : '刷新'}
        </button>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">时间</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">源ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">账号ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">源类型</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态码</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">请求路径</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">上游URL</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">错误信息</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {logs.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-6 py-8 text-center text-gray-500">暂无数据</td>
                </tr>
              ) : (
                logs.map((log) => (
                  <tr key={log.id} className="hover:bg-gray-50 align-top">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {new Date(log.created_at).toLocaleString('zh-CN')}
                      {log.request_time ? (
                        <div className="text-xs text-gray-500">请求: {new Date(log.request_time).toLocaleString('zh-CN')}</div>
                      ) : null}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{log.source_id}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{log.account_id}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-xs font-mono text-gray-700">{log.source_type}</td>
                    <td className={`px-6 py-4 whitespace-nowrap text-sm font-medium ${log.status_code ? 'text-red-600' : 'text-gray-500'}` }>
                      {log.status_code ?? '-'}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      <div className="max-w-xs truncate" title={log.request_path || ''}>{log.request_path || '-'}</div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      <div className="max-w-xs truncate" title={log.upstream_url || ''}>{log.upstream_url || '-'}</div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      <div className="max-w-md whitespace-pre-wrap break-words" title={log.error_message || ''}>
                        {log.error_message || '-'}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

