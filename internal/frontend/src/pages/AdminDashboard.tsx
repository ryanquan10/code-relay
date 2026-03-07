import { useEffect, useState } from 'react';
import { useNavigate, useParams, Navigate } from 'react-router-dom';
import Sidebar from '../components/Sidebar';
import Header from '../components/Header';
import SourcesTab from '../components/SourcesTab';
import ProductsTab from '../components/ProductsTab';
import AccountsTab from '../components/AccountsTab';
import UsageTab from '../components/UsageTab';
import ErrorLogTab from '../components/ErrorLogTab';
import { clearStoredAuth, getStoredUser } from '../utils/auth';

type MenuConfig = {
  title: string;
  component: React.ComponentType;
};

const menuConfig: Record<string, MenuConfig> = {
  sources: { title: '账号来源管理', component: SourcesTab },
  products: { title: '产品管理', component: ProductsTab },
  accounts: { title: '账号管理', component: AccountsTab },
  usage: { title: '使用量查看', component: UsageTab },
  'error-logs': { title: '错误日志', component: ErrorLogTab },
};

export default function AdminDashboard() {
  const navigate = useNavigate();
  const { tab } = useParams<{ tab: string }>();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const user = getStoredUser();
    const token = localStorage.getItem('admin_token');

    if (!user || !token) {
      navigate('/login', { replace: true });
      return;
    }

    if (user.role !== 'admin') {
      clearStoredAuth();
      alert('无权限访问');
      navigate('/login', { replace: true });
      return;
    }

    setLoading(false);
  }, [navigate]);

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-gray-600">加载中...</div>
      </div>
    );
  }

  // 如果 tab 无效，重定向到默认页
  if (!tab || !menuConfig[tab]) {
    return <Navigate to="/admin/sources" replace />;
  }

  const current = menuConfig[tab];
  const CurrentComponent = current.component;

  const handleMenuChange = (menu: string) => {
    navigate(`/admin/${menu}`);
  };

  return (
    <div className="flex h-screen bg-gray-100">
      <Sidebar activeMenu={tab} onMenuChange={handleMenuChange} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header title={current.title} />
        <div className="flex-1 overflow-y-auto p-6 bg-gray-50">
          <CurrentComponent />
        </div>
      </div>
    </div>
  );
}
