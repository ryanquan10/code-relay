import axios from 'axios';

const api = axios.create({
  baseURL: '/api/admin',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    // 从 localStorage 获取 token 并添加到请求头
    const token = localStorage.getItem('admin_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response.data;
  },
  (error) => {
    if (error.response?.status === 401) {
      // 清除 token 和用户信息
      localStorage.removeItem('admin_token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;

// API 接口定义

// 账号供应管理
export interface Source {
  id: number;
  source_name: string;
  upstream_url?: string | null;
  upstream_token?: string | null;
  source_type: string;
  config: Record<string, unknown> | string | null;
  priority: number;
  auto_rental: boolean;
  status: number;
  remark?: string | null;
  create_time: string;
  update_time: string;
}

export const sourceAPI = {
  list: () => api.get<any, Source[]>('/sources'),
  create: (data: Partial<Source>) => api.post<any, Source>('/sources', data),
  update: (id: number, data: Partial<Source>) => api.put<any, Source>(`/sources/${id}`, data),
  delete: (id: number) => api.delete(`/sources/${id}`),
};

// 产品管理
export interface Product {
  id: number;
  product_code: string;
  product_name: string;
  account_type: string;
  category?: string | null;
  icon?: string | null;
  image_url?: string | null;
  down_stream_url?: string | null;
  description?: string | null;
  price: number;
  original_price?: number | null;
  sales_count: number;
  contact_info?: string | null;
  usage_instruction?: string | null;
  validity_days: number;
  shared_limit: number;
  auto_delivery: boolean;
  sort_order: number;
  status: number;
  cost_price: number;
  default_balance: number;
  original_balance: number;
  stock: number;
  version: number;
  platforms?: any;
  sources?: ProductSource[];
  create_time: string;
  update_time: string;
}

export interface ProductSource {
  source_id: number;
  weight: number;
}

export const productAPI = {
  list: () => api.get<any, Product[]>('/products'),
  create: (data: Partial<Product>) => api.post<any, Product>('/products', data),
  update: (id: number, data: Partial<Product>) => api.put<any, Product>(`/products/${id}`, data),
  delete: (id: number) => api.delete(`/products/${id}`),
};

// 账号管理
export interface Account {
  id: number;
  account_email: string;
  account_password?: string | null;
  token?: string | null;
  product_id: number;
  source_id: number;
  user_id?: number | null;
  balance: number;
  used_balance?: number;  // 已使用余额
  start_time?: string | null;
  expire_days?: number;
  use_status: number;
  status: string;
  expire_date?: string | null;
  last_recharge_time?: string | null;
  remark?: string | null;
  version: number;
  create_time: string;
  update_time: string;
}

export const accountAPI = {
  list: (params?: any) => api.get<any, Account[]>('/accounts', { params }),
  getByToken: (token: string) => api.get<any, Account>(`/accounts/token/${token}`),
  create: (data: Partial<Account>) => api.post<any, Account>('/accounts', data),
  batchCreate: (data: any) => {
    // 如果是数组，包装为 { accounts }；如果是对象（包含 text 字段），直接发送
    const payload = Array.isArray(data) ? { accounts: data } : data;
    return api.post<any, { success: number; failed: number }>('/accounts/batch', payload);
  },
  update: (id: number, data: Partial<Account>) => api.put<any, Account>(`/accounts/${id}`, data),
  delete: (id: number) => api.delete(`/accounts/${id}`),
  batchDelete: (ids: number[]) => api.post<any, { message: string; count: number }>('/accounts/batch-delete', { ids }),
  updateBalance: (id: number, balance: number, usedBalance?: number) =>
    api.put(`/accounts/${id}/balance`, { balance, used_balance: usedBalance }),
  exportCsv: (params?: any) => api.get<any, Blob>(
    '/accounts/export',
    { params, responseType: "blob" }
  ),
};

// 使用量查看
export interface Usage {
  id: number;
  user_id: number;
  account_id: number;
  customer_key: string;
  tokens: number;
  consume: number;
  date: string;
  created_at: string;
}

export const usageAPI = {
  list: (params?: any) => api.get<any, Usage[]>('/usage', { params }),
  getByToken: (customerKey: string, dates?: string[]) =>
    api.post<any, { total_consume: number; details: Usage[] }>('/usage/query', { customer_key: customerKey, dates }),
  getStats: (startDate?: string, endDate?: string) =>
    api.get<any, any>('/usage/stats', { params: { start_date: startDate, end_date: endDate } }),
};


// Public API client for non-admin endpoints
const publicApi = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: { 'Content-Type': 'application/json' },
});
publicApi.interceptors.response.use(
  (response) => response.data,
  (error) => Promise.reject(error)
);

export interface DailyUsage {
  date: string;
  total_consume: number;
  record_count: number;
}

export interface DailyUsageSummary {
  total_consume: number;
  total_records: number;
  days: number;
}

export interface DailyUsageResponse {
  success: boolean;
  data: {
    daily_usages: DailyUsage[];
    summary: DailyUsageSummary;
  };
}

export const usageDailyAPI = {
  byDays: (customerKey: string, days: number = 7) =>
    publicApi.get<any, DailyUsageResponse>('/usage/daily', {
      params: { days },
      headers: { Authorization: `Bearer ${customerKey}` },
    }),
  byRange: (customerKey: string, startDate: string, endDate: string) =>
    publicApi.get<any, DailyUsageResponse>('/usage/daily/range', {
      params: { start_date: startDate, end_date: endDate },
      headers: { Authorization: `Bearer ${customerKey}` },
    }),
};








// 上游错误日志
export interface UpstreamErrorLog {
  id: number;
  account_id: number;
  source_id: number;
  source_type: string;
  upstream_url?: string | null;
  request_path?: string | null;
  status_code?: number | null;
  error_message?: string | null;
  request_time?: string | null;
  created_at: string;
}

export const errorLogAPI = {
  list: (params?: any) => api.get<any, UpstreamErrorLog[]>('/error-logs', { params }),
};


// 控制台日志（服务器 stdout/SQL/Gin）
export interface ConsoleLogsResponse {
  lines: string[];
}

export const consoleLogAPI = {
  list: (limit: number = 200) => api.get<any, ConsoleLogsResponse>(
    '/console-logs', { params: { limit } }
  ),
};
