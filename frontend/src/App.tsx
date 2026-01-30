import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Login from './pages/Login';
import AdminDashboard from './pages/AdminDashboard';
import Guide from './pages/Guide';
import BillCheck from './pages/BillCheck';
import Import from './pages/Import';

function App() {
    return (
        <BrowserRouter>
            <Routes>
                {/* 根路径重定向到 admin/sources */}
                <Route path="/" element={<Navigate to="/admin/sources" replace />} />

                {/* 登录页 */}
                <Route path="/login" element={<Login />} />

                {/* 教程页（公开访问） */}
                <Route path="/guide" element={<Guide />} />

                {/* 公开导入（无需登录） */}
                <Route path="/import" element={<Import />} />

                {/* /admin 重定向到 /admin/sources */}
                <Route path="/admin" element={<Navigate to="/admin/sources" replace />} />

                {/* Admin 子页面（带 tab 参数） */}
                <Route path="/admin/:tab" element={<AdminDashboard />} />

                {/* 公开账单查询 */}
                <Route path="/balance" element={<BillCheck />} />
                <Route path="/bill-check" element={<BillCheck />} />

                {/* 404 重定向到 admin/sources */}
                <Route path="*" element={<Navigate to="/admin/sources" replace />} />
            </Routes>
        </BrowserRouter>
    );
}

export default App;
