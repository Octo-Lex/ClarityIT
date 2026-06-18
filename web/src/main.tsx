import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Routes, Route, Navigate, useParams } from 'react-router-dom';
import './index.css';
import { AuthProvider, useAuth } from './auth/context';
import { QueryProvider } from './api/QueryProvider';
import { ThemeProvider } from './components/theme/ThemeProvider';
import { Toaster } from './components/Toaster';
import { useRealtimeInvalidation } from './hooks/useRealtimeInvalidation';
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
import AdminMetrics from './features/admin/AdminMetrics';
import AdminIntegrations from './features/admin/AdminIntegrations';
import AdminSetup from './features/admin/AdminSetup';
import SecurityPage from './features/account/SecurityPage';
import AdminApprovals from './features/admin/AdminApprovals';
import AdminAssetActions from './features/admin/AdminAssetActions';
import RemediationPanel from './features/incidents/RemediationPanel';
import AssetActions from './features/assets/AssetActions';
import { AgentsPage } from './features/agents/AgentsPage';
import ArtifactsPage from './features/artifacts/ArtifactsPage';
import { KnowledgeSearchPage } from './features/knowledge/KnowledgeSearchPage';
import { KnowledgeCollectionsPage } from './features/knowledge/KnowledgeCollectionsPage';
import { KnowledgeCollectionDetailPage } from './features/knowledge/KnowledgeCollectionDetailPage';
import { SavedKnowledgeAnswersPage } from './features/knowledge/SavedKnowledgeAnswersPage';
import { SavedKnowledgeAnswerDetailPage } from './features/knowledge/SavedKnowledgeAnswerDetailPage';
import { KnowledgeQualityPage } from './features/knowledge/KnowledgeQualityPage';
import DocumentEditorPage from './features/artifacts/DocumentEditorPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { authenticated, loading, activeTeamId } = useAuth();
  // Wire WS → React Query invalidation once authenticated + a team is active.
  useRealtimeInvalidation(activeTeamId);
  if (loading) return <div className="min-h-screen flex items-center justify-center text-muted-foreground">Loading...</div>;
  if (!authenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function OwnerRoute({ children }: { children: React.ReactNode }) {
  const { isPlatformOwner, loading } = useAuth();
  if (loading) return null;
  if (!isPlatformOwner) return <Navigate to="/" replace />;
  return <>{children}</>;
}

function PermissionRoute({ perm, children }: { perm: string; children: React.ReactNode }) {
  const { hasPermission, loading } = useAuth();
  if (loading) return null;
  if (!hasPermission(perm)) return <Navigate to="/" replace />;
  return <>{children}</>;
}

function AssetActionRoute() {
  const { id } = useParams<{ id: string }>();
  return <AssetActions assetId={id || ''} />;
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <ThemeProvider>
        <QueryProvider>
          <AuthProvider>
            <Routes>
          <Route path="/bootstrap" element={<BootstrapPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/queue" element={<QueuePage />} />
            <Route path="/board" element={<BoardPage />} />
            <Route path="/objects/:id" element={<ObjectDetailPage />} />
            <Route path="/work-items/new" element={<WorkItemNew />} />
            <Route path="/incidents" element={<IncidentList />} />
            <Route path="/incidents/:id" element={<IncidentDetail />} />
            <Route path="/settings/team" element={<TeamSettings />} />
            <Route path="/agents" element={<AgentsPage />} />
            <Route path="/artifacts" element={<ArtifactsPage />} />
          <Route path="/knowledge" element={<PermissionRoute perm="knowledge.search"><KnowledgeSearchPage /></PermissionRoute>} />
          <Route path="/knowledge/collections" element={<PermissionRoute perm="knowledge.collections.read"><KnowledgeCollectionsPage /></PermissionRoute>} />
          <Route path="/knowledge/collections/:collectionId" element={<PermissionRoute perm="knowledge.collections.read"><KnowledgeCollectionDetailPage /></PermissionRoute>} />
          <Route path="/knowledge/saved-answers" element={<PermissionRoute perm="knowledge.collections.read"><SavedKnowledgeAnswersPage /></PermissionRoute>} />
          <Route path="/knowledge/saved-answers/:answerId" element={<PermissionRoute perm="knowledge.collections.read"><SavedKnowledgeAnswerDetailPage /></PermissionRoute>} />
          <Route path="/knowledge/quality" element={<PermissionRoute perm="knowledge.read"><KnowledgeQualityPage /></PermissionRoute>} />
            <Route path="/teams/:teamId/artifacts/documents/:artifactId" element={<DocumentEditorPage />} />
            <Route path="/artifacts/documents/:artifactId" element={<DocumentEditorPage />} />
            <Route path="/account/security" element={<SecurityPage />} />
            <Route path="/admin/approvals" element={<PermissionRoute perm="approvals.read"><AdminApprovals /></PermissionRoute>} />
            <Route path="/admin/asset-actions" element={<PermissionRoute perm="assets.actions.read"><AdminAssetActions /></PermissionRoute>} />
            <Route path="/incidents/:id/remediation" element={<RemediationPanel />} />
            <Route path="/assets/:id/actions" element={<AssetActionRoute />} />
            <Route path="/admin/users" element={<OwnerRoute><AdminUsers /></OwnerRoute>} />
            <Route path="/admin/teams" element={<OwnerRoute><AdminTeams /></OwnerRoute>} />
            <Route path="/admin/audit" element={<OwnerRoute><AdminAudit /></OwnerRoute>} />
            <Route path="/admin/ops" element={<OwnerRoute><AdminOps /></OwnerRoute>} />
            <Route path="/admin/metrics" element={<OwnerRoute><AdminMetrics /></OwnerRoute>} />
            <Route path="/admin/integrations" element={<OwnerRoute><AdminIntegrations /></OwnerRoute>} />
            <Route path="/admin/setup" element={<OwnerRoute><AdminSetup /></OwnerRoute>} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
          </AuthProvider>
          <Toaster />
        </QueryProvider>
      </ThemeProvider>
    </BrowserRouter>
  </StrictMode>,
);
