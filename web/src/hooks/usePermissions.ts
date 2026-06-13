import { useAuth } from '../auth/context';

export function usePermissions() {
  const { hasPermission, isPlatformOwner } = useAuth();
  return { hasPermission, isPlatformOwner };
}
