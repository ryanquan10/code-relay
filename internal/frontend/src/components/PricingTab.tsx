import { useEffect, useState } from 'react';

import { Pricing, pricingAPI } from '../api/client';

type PricingRow = Pricing & { dirty?: boolean };

const normalizeUnit = (value: string) => {
  const upper = value.trim().toUpperCase();
  if (upper === 'USD') return 'US';
  if (upper === 'CNY') return 'RMB';
  if (upper === 'RMB') return 'RMB';
  return 'US';
};

export default function PricingTab() {
  const [rows, setRows] = useState<PricingRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingId, setSavingId] = useState<number | null>(null);

  const loadPricings = async () => {
    try {
      setLoading(true);
      const data = await pricingAPI.list();
      setRows(data.map((item) => ({ ...item, dirty: false })));
    } catch (error) {
      console.error('加载计价配置失败:', error);
      alert('加载计价配置失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPricings();
  }, []);

  const updateRow = (id: number, patch: Partial<PricingRow>) => {
    setRows((prev) =>
      prev.map((row) => (row.id === id ? { ...row, ...patch, dirty: true } : row))
    );
  };

  const handleSave = async (row: PricingRow) => {
    try {
      if (row.token_unit <= 0) {
        alert('token_unit 必须大于 0');
        return;
      }
      setSavingId(row.id);
      const saved = await pricingAPI.update(row.id, {
        in_token_unit_price: row.in_token_unit_price,
        out_token_unit_price: row.out_token_unit_price,
        token_unit: row.token_unit,
        unit: normalizeUnit(row.unit),
      });
      setRows((prev) =>
        prev.map((item) =>
          item.id === row.id ? { ...saved, dirty: false } : item
        )
      );
      alert(`已保存 ${row.account_type} 的计价配置`);
    } catch (error) {
      console.error('保存计价配置失败:', error);
      alert('保存计价配置失败');
    } finally {
      setSavingId(null);
    }
  };

  if (loading) {
    return <div className="text-center py-8">加载中...</div>;
  }

  return (
    <div>
      <div className="mb-4">
        <div className="text-sm text-gray-600">
          account_type 来源于 `product` 表，自动按 account_type 去重，无需手动同步。
        </div>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">account_type</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">in_token_unit_price</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">out_token_unit_price</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">token_unit</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">unit</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                    暂无计价数据，请先在产品里配置 account_type。
                  </td>
                </tr>
              ) : (
                rows.map((row) => (
                  <tr key={row.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm text-gray-900">{row.account_type}</td>
                    <td className="px-4 py-3">
                      <input
                        type="number"
                        step="0.00000001"
                        min="0"
                        value={row.in_token_unit_price}
                        onChange={(e) => {
                          const value = Number(e.target.value);
                          updateRow(row.id, { in_token_unit_price: Number.isFinite(value) ? value : 0 });
                        }}
                        className="w-44 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <input
                        type="number"
                        step="0.00000001"
                        min="0"
                        value={row.out_token_unit_price}
                        onChange={(e) => {
                          const value = Number(e.target.value);
                          updateRow(row.id, { out_token_unit_price: Number.isFinite(value) ? value : 0 });
                        }}
                        className="w-44 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <input
                        type="number"
                        step="1"
                        min="1"
                        value={row.token_unit}
                        onChange={(e) => {
                          const value = Number(e.target.value);
                          updateRow(row.id, { token_unit: Number.isFinite(value) ? Math.max(1, Math.round(value)) : 1 });
                        }}
                        className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <select
                        value={normalizeUnit(row.unit)}
                        onChange={(e) => updateRow(row.id, { unit: normalizeUnit(e.target.value) })}
                        className="w-24 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-sm"
                      >
                        <option value="US">US</option>
                        <option value="RMB">RMB</option>
                      </select>
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleSave(row)}
                        disabled={!row.dirty || savingId === row.id}
                        className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white px-3 py-2 rounded-lg text-sm transition"
                      >
                        {savingId === row.id ? '保存中...' : '保存'}
                      </button>
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
