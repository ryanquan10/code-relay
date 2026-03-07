import { useNavigate } from 'react-router-dom';

import { clearStoredAuth } from '../utils/auth';

interface SidebarProps {
  activeMenu: string;
  onMenuChange: (menu: string) => void;
}

export default function Sidebar({ activeMenu, onMenuChange }: SidebarProps) {
  const navigate = useNavigate();

  const menuItems = [
    { id: 'sources', label: '账号供应管理', icon: '📦' },
    { id: 'products', label: '产品管理', icon: '🛍️' },
    { id: 'accounts', label: '账号管理', icon: '👤' },
    { id: 'usage', label: '使用量查看', icon: '📊' },
    { id: 'error-logs', label: '错误日志', icon: '🚨' },
  ];

  const handleLogout = () => {
    if (confirm('确定要退出登录吗？')) {
      clearStoredAuth();
      navigate('/login');
    }
  };

  return (
    <div className="w-64 bg-gray-900 text-white flex flex-col">
      <div className="p-6 border-b border-gray-800">
        <h1 className="text-xl font-bold">账户管理系统</h1>
        <p className="text-sm text-gray-400 mt-1">管理员面板</p>
      </div>

      <nav className="flex-1 p-4 space-y-2">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => onMenuChange(item.id)}
            className={`w-full flex items-center space-x-3 px-4 py-3 rounded-lg transition ${
              activeMenu === item.id
                ? 'bg-blue-600 text-white'
                : 'text-gray-300 hover:bg-gray-800'
            }`}
          >
            <span className="text-xl">{item.icon}</span>
            <span className="font-medium">{item.label}</span>
          </button>
        ))}
      </nav>

      <div className="p-4 border-t border-gray-800">
        <button
          onClick={handleLogout}
          className="w-full flex items-center space-x-3 px-4 py-3 rounded-lg text-gray-300 hover:bg-gray-800 transition"
        >
          <span className="text-xl">🚪</span>
          <span className="font-medium">退出登录</span>
        </button>
      </div>
    </div>
  );
}
