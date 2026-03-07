export interface StoredUser {
  username?: string;
  role?: string;
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

export function clearStoredAuth() {
  localStorage.removeItem('admin_token');
  localStorage.removeItem('user');
}

export function getStoredUser(): StoredUser | null {
  const userRaw = localStorage.getItem('user');
  if (!userRaw) {
    return null;
  }

  try {
    const parsed: unknown = JSON.parse(userRaw);
    if (!isObject(parsed)) {
      clearStoredAuth();
      return null;
    }

    const username = typeof parsed.username === 'string' ? parsed.username : undefined;
    const role = typeof parsed.role === 'string' ? parsed.role : undefined;
    return { username, role };
  } catch (error) {
    console.error('读取用户信息失败:', error);
    clearStoredAuth();
    return null;
  }
}
