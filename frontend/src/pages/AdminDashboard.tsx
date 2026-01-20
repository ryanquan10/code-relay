import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Sidebar from '../components/Sidebar';
import Header from '../components/Header';
import SourcesTab from '../components/SourcesTab';
import ProductsTab from '../components/ProductsTab';
import AccountsTab from '../components/AccountsTab';
import UsageTab from '../components/UsageTab';

type MenuConfig = {
  title: string;
  component: React.ComponentType;
};

const menuConfig: Record<string, MenuConfig> = {
  sources: { title: '发卡来源管理', component: SourcesTab },
  products: { title: '产品管理', component: ProductsTab },
  accounts: { title: '账号管理', component: AccountsTab },
  usage: { title: '使用量查看', component: UsageTab },
};

export default function AdminDashboard() {
  const navigate = useNavigate();
  const [activeMenu, setActiveMenu] = useState('sources');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const userStr = localStorage.getItem('user');
    if (!userStr) {
      navigate('/login');
      return;
    }

    const user = JSON.parse(userStr);
    if (user.role !== 'admin') {
      alert('无权限访问');
      navigate('/login');
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

  const current = menuConfig[activeMenu] || menuConfig.sources;
  const CurrentComponent = current.component;

  return (
    <div className="flex h-screen bg-gray-100">
      <Sidebar activeMenu={activeMenu} onMenuChange={setActiveMenu} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header title={current.title} />
        <div className="flex-1 overflow-y-auto p-6 bg-gray-50">
          <CurrentComponent />
        </div>
      </div>
    </div>
  );
}
