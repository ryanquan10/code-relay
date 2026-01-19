interface HeaderProps {
  title: string;
}

export default function Header({ title }: HeaderProps) {
  const user = JSON.parse(localStorage.getItem('user') || '{}');

  return (
    <div className="bg-white border-b border-gray-200 px-6 py-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">{title}</h2>
        <div className="flex items-center space-x-4">
          <div className="text-right">
            <p className="text-sm font-medium text-gray-900">{user.username || 'Admin'}</p>
            <p className="text-xs text-gray-500">{user.role || 'Administrator'}</p>
          </div>
          <div className="w-10 h-10 bg-blue-600 rounded-full flex items-center justify-center text-white font-bold">
            A
          </div>
        </div>
      </div>
    </div>
  );
}
