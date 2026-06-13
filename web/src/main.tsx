import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import './index.css';
import { AuthProvider, useAuth } from './auth/context';
import { RefetchProvider } from './hooks/useRefetch';
import { setAccessToken } from './api/client';
import AppLayout from './components/layout/AppLayout';
import BootstrapPage from './features/bootstrap/BootstrapPage';
import LoginPage from './features/auth/LoginPage';
import ForgotPasswordPage from './features/auth/ForgotPasswordPage';
import ResetPasswordPage from './features/auth/ResetPasswordPage';
import Dashboard from './features/dashboard/Dashboard';
import QueuePage from './features/queue/QueuePage';
import BoardPage from './features/board/BoardPage';
import ObjectDetailPage from './features/objects/ObjectDetailPage';
import WorkItemNew from './features/work-items/WorkItemNew';
import IncidentList from './features/incidents/IncidentList';
import IncidentDetail from './features/incidents/IncidentDetail';
import TeamSettings from './features/team/TeamSettings';
import AdminUsers from './features/admin/AdminUsers';
import AdminTeams from './features/admin/AdminTeams';
import AdminAudit from './features/admin/AdminAudit';
import AdminOps from './features/admin/AdminOps';
import { AgentsPage } from './features/agents/AgentsPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { authenticated, loading } = useAuth();
  if (loading) return <div className="min-h-screen flex items-center justify-center text-[var(--text-muted)]">Loading...</div>;
  if (!authenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function OwnerRoute({ children }: { children: React.ReactNode }) {
  const { isPlatformOwner, loading } = useAuth();
  if (loading) return null;
  if (!isPlatformOwner) return <Navigate to="/" replace />;
  return <>{children}</>;
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/bootstrap" element={<BootstrapPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route element={<ProtectedRoute><RefetchProvider><AppLayout /></RefetchProvider></ProtectedRoute>}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/queue" element={<QueuePage />} />
            <Route path="/board" element={<BoardPage />} />
            <Route path="/objects/:id" element={<ObjectDetailPage />} />
            <Route path="/work-items/new" element={<WorkItemNew />} />
            <Route path="/incidents" element={<IncidentList />} />
            <Route path="/incidents/:id" element={<IncidentDetail />} />
            <Route path="/settings/team" element={<TeamSettings />} />
            <Route path="/agents" element={<AgentsPage />} />
            <Route path="/admin/users" element={<OwnerRoute><AdminUsers /></OwnerRoute>} />
            <Route path="/admin/teams" element={<OwnerRoute><AdminTeams /></OwnerRoute>} />
            <Route path="/admin/audit" element={<OwnerRoute><AdminAudit /></OwnerRoute>} />
            <Route path="/admin/ops" element={<OwnerRoute><AdminOps /></OwnerRoute>} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>,
);
