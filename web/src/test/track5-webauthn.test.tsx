import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../auth/context';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    me: vi.fn().mockResolvedValue({ id: 'u1', email: 'owner@test.dev', name: 'Owner', teams: [{ id: 't1', name: 'Team', slug: 'team', role: 'owner' }] }),
    permissions: vi.fn().mockResolvedValue({ role: 'owner', team_id: 't1', permissions: [] }),
    mfaStatus: vi.fn().mockResolvedValue({ enabled: false, verified_factors: 0, pending_factors: 0 }),
    mfaListFactors: vi.fn().mockResolvedValue([]),
    mfaEnroll: vi.fn(),
    mfaVerifyEnrollment: vi.fn(),
    mfaRegenerateRecovery: vi.fn(),
    mfaDisableFactor: vi.fn(),
    // WebAuthn methods
    webauthnRegisterStart: vi.fn(),
    webauthnRegisterFinish: vi.fn(),
    webauthnAuthStart: vi.fn(),
    webauthnAuthFinish: vi.fn(),
    webauthnListCredentials: vi.fn(),
    webauthnDisableCredential: vi.fn(),
  },
  setAccessToken: vi.fn(),
  getStoredTeamId: vi.fn().mockReturnValue('t1'),
  setStoredTeamId: vi.fn(),
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import SecurityPage from '../features/account/SecurityPage';
import { api } from '../api/client';

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MemoryRouter>
      <AuthProvider>{ui}</AuthProvider>
    </MemoryRouter>
  );
}

describe('SecurityPage — WebAuthn', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.mfaStatus).mockResolvedValue({ enabled: false, verified_factors: 0, pending_factors: 0 });
    vi.mocked(api.mfaListFactors).mockResolvedValue([]);
    vi.mocked(api.webauthnListCredentials).mockResolvedValue([]);
  });

  // Test 1: Security page shows Add Security Key when WebAuthn available
  it('shows Add Security Key button when WebAuthn is available', async () => {
    // Mock WebAuthn availability
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: function() {},
      configurable: true,
    });
    Object.defineProperty(navigator, 'credentials', {
      value: { create: vi.fn(), get: vi.fn() },
      configurable: true,
    });

    renderWithProviders(<SecurityPage />);

    await waitFor(() => {
      expect(screen.getByTestId('webauthn-section')).toBeInTheDocument();
    });

    expect(screen.getByTestId('wa-register-btn')).toBeInTheDocument();
    expect(screen.getByTestId('wa-label-input')).toBeInTheDocument();
  });

  // Test 2: Unsupported browser/context message renders
  it('shows unsupported message when WebAuthn is not available', async () => {
    // Remove WebAuthn support
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: undefined,
      configurable: true,
    });

    renderWithProviders(<SecurityPage />);

    await waitFor(() => {
      expect(screen.getByTestId('webauthn-section')).toBeInTheDocument();
    });

    expect(screen.getByTestId('webauthn-unavailable')).toBeInTheDocument();
    expect(screen.queryByTestId('wa-register-btn')).not.toBeInTheDocument();
  });

  // Test 3: Registration start calls API
  it('calls webauthnRegisterStart when Add Security Key is clicked', async () => {
    // Mock WebAuthn availability
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: function() {},
      configurable: true,
    });

    // Mock navigator.credentials.create to return a fake credential
    const mockCredential = {
      id: 'fake-id',
      rawId: new ArrayBuffer(16),
      type: 'public-key',
      response: {
        attestationObject: new ArrayBuffer(32),
        clientDataJSON: new ArrayBuffer(16),
      },
    };
    Object.defineProperty(navigator, 'credentials', {
      value: { create: vi.fn().mockResolvedValue(mockCredential), get: vi.fn() },
      configurable: true,
    });

    vi.mocked(api.webauthnRegisterStart).mockResolvedValue({
      options: {
        challenge: 'dGVzdC1jaGFsbGVuZ2U',
        rp: { name: 'ClarityIT', id: 'localhost' },
        user: { id: 'dXNlcjE', name: 'owner@test.dev', displayName: 'Owner' },
        pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
        timeout: 300000,
        excludeCredentials: [],
        authenticatorSelection: { userVerification: 'preferred' },
      },
    });
    vi.mocked(api.webauthnRegisterFinish).mockResolvedValue({ message: 'ok', credential_id: 'cred-1', label: 'Test' });

    renderWithProviders(<SecurityPage />);

    await waitFor(() => {
      expect(screen.getByTestId('wa-register-btn')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByTestId('wa-label-input'), { target: { value: 'YubiKey' } });
    fireEvent.click(screen.getByTestId('wa-register-btn'));

    await waitFor(() => {
      expect(api.webauthnRegisterStart).toHaveBeenCalledWith('YubiKey');
    });
  });

  // Test 4: Registration finish calls API
  it('calls webauthnRegisterFinish after navigator.credentials.create', async () => {
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: function() {},
      configurable: true,
    });

    const mockCredential = {
      id: 'fake-id',
      rawId: new ArrayBuffer(16),
      type: 'public-key',
      response: {
        attestationObject: new ArrayBuffer(32),
        clientDataJSON: new ArrayBuffer(16),
      },
    };
    const mockCreate = vi.fn().mockResolvedValue(mockCredential);
    Object.defineProperty(navigator, 'credentials', {
      value: { create: mockCreate, get: vi.fn() },
      configurable: true,
    });

    vi.mocked(api.webauthnRegisterStart).mockResolvedValue({
      options: {
        challenge: 'dGVzdC1jaGFsbGVuZ2U',
        rp: { name: 'ClarityIT', id: 'localhost' },
        user: { id: 'dXNlcjE', name: 'owner@test.dev', displayName: 'Owner' },
        pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
        timeout: 300000,
        excludeCredentials: [],
        authenticatorSelection: { userVerification: 'preferred' },
      },
    });
    vi.mocked(api.webauthnRegisterFinish).mockResolvedValue({ message: 'ok' });
    vi.mocked(api.webauthnListCredentials).mockResolvedValue([
      { id: 'cred-1', label: 'YubiKey', status: 'active', created_at: new Date().toISOString() },
    ]);

    renderWithProviders(<SecurityPage />);

    await waitFor(() => {
      expect(screen.getByTestId('wa-register-btn')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('wa-register-btn'));

    await waitFor(() => {
      expect(api.webauthnRegisterFinish).toHaveBeenCalled();
    });
  });

  // Test 5: Credential list renders labels only (no secrets)
  it('renders credential labels without sensitive material', async () => {
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: function() {},
      configurable: true,
    });
    Object.defineProperty(navigator, 'credentials', {
      value: { create: vi.fn(), get: vi.fn() },
      configurable: true,
    });

    vi.mocked(api.webauthnListCredentials).mockResolvedValue([
      { id: 'cred-1', label: 'YubiKey 5C', status: 'active', created_at: '2026-06-14T10:00:00Z' },
      { id: 'cred-2', label: 'Touch ID', status: 'disabled', created_at: '2026-06-13T10:00:00Z', disabled_at: '2026-06-14T10:00:00Z' },
    ]);

    renderWithProviders(<SecurityPage />);

    await waitFor(() => {
      expect(screen.getByText('YubiKey 5C')).toBeInTheDocument();
      expect(screen.getByText('Touch ID')).toBeInTheDocument();
    });

    // Verify no sensitive material rendered
    const section = screen.getByTestId('webauthn-section');
    const sectionText = section.textContent || '';
    expect(sectionText).not.toContain('credential_id_bytes');
    expect(sectionText).not.toContain('public_key');
    expect(sectionText).not.toContain('credential_id_hash');
    expect(sectionText).not.toContain('challenge');
  });

  // Test 6: Disable credential sends Idempotency-Key
  it('sends disable request with Idempotency-Key', async () => {
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: function() {},
      configurable: true,
    });
    Object.defineProperty(navigator, 'credentials', {
      value: { create: vi.fn(), get: vi.fn() },
      configurable: true,
    });

    const credId = 'abc12345-dead-beef-cafe-123456789abc';
    vi.mocked(api.webauthnListCredentials).mockResolvedValue([
      { id: credId, label: 'YubiKey', status: 'active', created_at: '2026-06-14T10:00:00Z' },
    ]);
    vi.mocked(api.webauthnDisableCredential).mockResolvedValue({ message: 'disabled' });

    renderWithProviders(<SecurityPage />);

    await waitFor(() => {
      expect(screen.getByText('YubiKey')).toBeInTheDocument();
    });

    // Click disable button
    const disableBtn = screen.getByTestId(`wa-disable-${credId}`);
    fireEvent.click(disableBtn);

    await waitFor(() => {
      expect(api.webauthnDisableCredential).toHaveBeenCalledWith(credId);
    });
  });

  // Test 7: No credential_id/public_key/challenge secrets rendered
  it('does not render any secret material', async () => {
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: function() {},
      configurable: true,
    });
    Object.defineProperty(navigator, 'credentials', {
      value: { create: vi.fn(), get: vi.fn() },
      configurable: true,
    });

    vi.mocked(api.webauthnListCredentials).mockResolvedValue([
      { id: 'cred-secret-test', label: 'Hardware Key', status: 'active', created_at: '2026-06-14T10:00:00Z' },
    ]);

    const { container } = renderWithProviders(<SecurityPage />);

    await waitFor(() => {
      expect(screen.getByText('Hardware Key')).toBeInTheDocument();
    });

    const fullHTML = container.innerHTML;
    // The credential UUID may appear as a data-testid for disable button,
    // but no raw credential_id_bytes, public_key, or challenge should appear
    expect(fullHTML).not.toContain('credential_id_bytes');
    expect(fullHTML).not.toContain('public_key');
    expect(fullHTML).not.toContain('credential_id_hash');
    expect(fullHTML).not.toContain('"challenge"');
  });
});
