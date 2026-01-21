import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import AdminDashboard from './pages/AdminDashboard';

function App() {
    return (
        <BrowserRouter>
            <Routes>
                {/* 根路径重定向到 admin/sources */}
                <Route path="/" element={<Navigate to="/admin/sources" replace />} />

                {/* 登录页 */}
                <Route path="/login" element={<Login />} />

                {/* /admin 重定向到 /admin/sources */}
                <Route path="/admin" element={<Navigate to="/admin/sources" replace />} />

                {/* Admin 子页面（带 tab 参数） */}
                <Route path="/admin/:tab" element={<AdminDashboard />} />

                {/* 404 重定向到 admin/sources */}
                <Route path="*" element={<Navigate to="/admin/sources" replace />} />
            </Routes>
        </BrowserRouter>
    );
}

export default App;