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
    // 这里可以添加 token 等认证信息
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
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;

// API 接口定义

// 货源管理
export interface Source {
  id: number;
  name: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
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
  category: string;
  price: number;
  original_price?: number;
  validity_days: number;
  shared_limit: number;
  sales_count: number;
  auto_delivery: boolean;
  sort_order: number;
  status: string;
  description: string;
  created_at: string;
  updated_at: string;
  source_ids: number[];
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
  user_id: number;
  product_id: number;
  account_email: string;
  token: string;
  balance: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export const accountAPI = {
  list: (params?: any) => api.get<any, Account[]>('/accounts', { params }),
  getByToken: (token: string) => api.get<any, Account>(`/accounts/token/${token}`),
  create: (data: Partial<Account>) => api.post<any, Account>('/accounts', data),
  batchCreate: (accounts: Partial<Account>[]) => api.post<any, { success: number; failed: number }>('/accounts/batch', { accounts }),
  update: (id: number, data: Partial<Account>) => api.put<any, Account>(`/accounts/${id}`, data),
  delete: (id: number) => api.delete(`/accounts/${id}`),
  updateBalance: (id: number, balance: number) => api.put(`/accounts/${id}/balance`, { balance }),
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
