import { useEffect, useRef, useState } from 'react';
import { errorLogAPI, UpstreamErrorLog, consoleLogAPI } from '../api/client';

export default function ErrorLogTab() {
  const [logs, setLogs] = useState<UpstreamErrorLog[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [autoRefresh, setAutoRefresh] = useState<boolean>(true);
  const [follow, setFollow] = useState<boolean>(true);
  const [view, setView] = useState<'table' | 'stream'>('table');
  const [streamSource, setStreamSource] = useState<'console' | 'upstream'>('console');
  const [consoleLines, setConsoleLines] = useState<string[]>([]);
  const logContainerRef = useRef<HTMLDivElement | null>(null);

  const loadLogs = async (silent: boolean = false) => {
    try {
      if (!silent) setLoading(true);
      const data = await errorLogAPI.list();
      const sorted = [...(data || [])].sort((a, b) => {
        const ta = new Date(a.created_at).getTime();
        const tb = new Date(b.created_at).getTime();
        return tb - ta;
      });
      setLogs(sorted);
    } catch (e) {
      console.error('加载错误日志失败:', e);
      if (!silent) alert('加载错误日志失败');
      if (!silent) setLogs([]);
    } finally {
      if (!silent) setLoading(false);
    }
  };

  const loadConsoleLogs = async () => {
    try {
      const data = await consoleLogAPI.list(500);
      setConsoleLines(data?.lines ?? []);
    } catch (e) {
      console.error('加载控制台日志失败:', e);
      setConsoleLines([]);
    }
  };

  const handleRefresh = async () => {
    try {
      setRefreshing(true);
      if (view === 'stream' && streamSource === 'console') {
        await loadConsoleLogs();
      } else {
        await loadLogs();
      }
    } finally {
      setRefreshing(false);
    }
  };

  // 初次加载
  useEffect(() => {
    loadLogs();
    loadConsoleLogs();
  }, []);

  // 自动刷新
  useEffect(() => {
    if (!autoRefresh) return;
    const id = setInterval(() => {
      if (view === 'stream') {
        if (streamSource === 'console') {
          loadConsoleLogs();
        } else {
          loadLogs(true);
        }
      }
    }, 5000);
    return () => clearInterval(id);
  }, [autoRefresh, view, streamSource]);

  // 跟随滚动到底部
  useEffect(() => {
    if (view === 'stream' && follow && logContainerRef.current) {
      try {
        logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
      } catch {}
    }
  }, [logs, consoleLines, follow, view]);

  if (loading && logs.length === 0) {
    return <div className="text-center py-8">加载中...</div>;
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div className="text-sm text-gray-600">共 {logs.length} 条记录</div>
        <div className="flex items-center">
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className={`px-4 py-2 rounded-lg text-white transition ${refreshing ? 'bg-gray-400' : 'bg-blue-600 hover:bg-blue-700'}`}
          >
            {refreshing ? '刷新中...' : '刷新'}
          </button>
          <div className="ml-3 flex items-center space-x-3 text-sm">
            <label className="flex items-center space-x-1 cursor-pointer">
              <input type="checkbox" className="form-checkbox" checked={autoRefresh} onChange={e => setAutoRefresh(e.target.checked)} />
              <span className="text-gray-600">自动刷新</span>
            </label>
            <label className={`flex items-center space-x-1 ${view === 'stream' ? 'cursor-pointer' : 'opacity-50 cursor-not-allowed'}`}
              title={view === 'stream' ? '' : '仅在“滚动日志”视图生效'}>
              <input type="checkbox" className="form-checkbox" checked={follow} disabled={view !== 'stream'} onChange={e => setFollow(e.target.checked)} />
              <span className="text-gray-600">跟随滚动</span>
            </label>
          </div>
        </div>
      </div>

      <div className="mb-4 border-b border-gray-200">
        <nav className="-mb-px flex space-x-6" aria-label="Tabs">
          <button
            onClick={() => setView('table')}
            className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
              view === 'table' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            表格视图
          </button>
          <button
            onClick={() => setView('stream')}
            className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
              view === 'stream' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            滚动日志
          </button>
        </nav>
      </div>

      {view === 'table' ? (
        <div className="bg-white rounded-lg shadow overflow-hidden mb-6">
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
                      <td className={`px-6 py-4 whitespace-nowrap text-sm font-medium ${log.status_code ? 'text-red-600' : 'text-gray-500'}`}>
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
      ) : (
        <div className="bg-white rounded-lg shadow">
          <div className="px-4 py-3 border-b border-gray-200 flex items-center justify-between">
            <div className="text-sm text-gray-600">滚动日志</div>
            <div className="space-x-4">
              <button
                onClick={() => setStreamSource('console')}
                className={`py-1 px-2 border-b-2 text-sm ${streamSource === 'console' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}
              >控制台</button>
              <button
                onClick={() => setStreamSource('upstream')}
                className={`py-1 px-2 border-b-2 text-sm ${streamSource === 'upstream' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}
              >上游错误</button>
            </div>
          </div>
          <div ref={logContainerRef} className="h-[60vh] md:h-[70vh] lg:h-[75vh] overflow-y-auto px-4 py-3 bg-gray-50">
            <pre className="text-xs font-mono text-gray-800 whitespace-pre-wrap">
              {streamSource === 'console'
                ? (consoleLines || []).join('\n')
                : logs
                    .slice()
                    .sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
                    .map((log) => {
                      const ts = new Date(log.created_at).toLocaleString('zh-CN');
                      const sc = log.status_code ?? '-';
                      const path = log.request_path || '-';
                      const msg = (log.error_message || '-').replace(/\n/g, ' ');
                      const src = log.source_type || '-';
                      return `[${ts}] [${src}] [${sc}] ${path} - ${msg}`;
                    })
                    .join('\n')}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}

