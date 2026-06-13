import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { api, setAccessToken, getStoredTeamId, setStoredTeamId, type User, type Permissions, ApiError } from '../api/client';

interface AuthState {
  user: User | null;
  permissions: string[];
  loading: boolean;
  authenticated: boolean;
  activeTeamId: string | null;
  isPlatformOwner: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => void;
  switchTeam: (teamId: string) => Promise<void>;
  refresh: () => Promise<void>;
  hasPermission: (perm: string) => boolean;
}

const AuthCtx = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [permissions, setPermissions] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTeamId, setActiveTeamId] = useState<string | null>(getStoredTeamId());

  const [isPlatformOwner, setIsPlatformOwner] = useState(false);

  const loadSession = useCallback(async () => {
    try {
      const me = await api.me();
      setUser(me);
      const tid = me.teams?.[0]?.id ?? null;
      if (tid) {
        setActiveTeamId(tid);
        setStoredTeamId(tid);
        try {
          const p = await api.permissions();
          setPermissions(p.permissions || []);
          setIsPlatformOwner(p.role === 'owner');
        } catch { setPermissions([]); }
      }
    } catch {
      setAccessToken(null);
      setUser(null);
      setPermissions([]);
    }
    setLoading(false);
  }, []);

  useEffect(() => { loadSession(); }, [loadSession]);

  useEffect(() => {
    const h = () => { setUser(null); setAccessToken(null); setPermissions([]); };
    window.addEventListener('auth:logout', h);
    return () => window.removeEventListener('auth:logout', h);
  }, []);

  const login = async (email: string, password: string) => {
    const r = await api.login({ email, password });
    setAccessToken(r.access_token);
    await loadSession();
  };

  const register = async (name: string, email: string, password: string) => {
    const r = await api.register({ name, email, password });
    setAccessToken(r.access_token);
    await loadSession();
  };

  const logout = () => {
    api.logout().catch(() => {});
    setAccessToken(null);
    setUser(null);
    setPermissions([]);
    setStoredTeamId(null);
    setActiveTeamId(null);
  };

  const switchTeam = async (teamId: string) => {
    const r = await api.switchTeam(teamId);
    setAccessToken(r.access_token);
    setActiveTeamId(teamId);
    setStoredTeamId(teamId);
    const p = await api.permissions();
    setPermissions(p.permissions || []);
    const me = await api.me();
    setUser(me);
  };

  const hasPermission = (perm: string) => {
    if (isPlatformOwner) return true;
    return permissions.includes(perm);
  };

  return (
    <AuthCtx.Provider value={{
      user, permissions, loading, authenticated: !!user,
      activeTeamId, isPlatformOwner,
      login, register, logout, switchTeam, refresh: loadSession, hasPermission,
    }}>
      {children}
    </AuthCtx.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error('useAuth must be within AuthProvider');
  return ctx;
}
