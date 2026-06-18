import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import ForgotPasswordPage from '../features/auth/ForgotPasswordPage';
import ResetPasswordPage from '../features/auth/ResetPasswordPage';

describe('ForgotPasswordPage', () => {
  it('shows the success screen regardless of whether the email exists (no enumeration)', async () => {
    let calledEmail: string | undefined;
    server.use(
      http.post('*/api/auth/forgot-password', async ({ request }) => {
        calledEmail = (await request.json() as { email: string }).email;
        // Backend always returns 200 — simulate that.
        return HttpResponse.json({ message: 'If that email exists, a reset link has been sent' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<ForgotPasswordPage />, { route: '/forgot-password' });

    await user.type(screen.getByTestId('forgot-email'), 'nobody@example.com');
    await user.click(screen.getByTestId('forgot-submit'));

    await waitFor(() => expect(screen.getByText('Check Your Email')).toBeInTheDocument());
    expect(calledEmail).toBe('nobody@example.com');
  });

  it('still shows success when the request fails (prevents enumeration)', async () => {
    server.use(
      http.post('*/api/auth/forgot-password', () =>
        HttpResponse.json({ detail: 'rate limited' }, { status: 429 }),
      ),
    );
    const user = userEvent.setup();
    renderWithProviders(<ForgotPasswordPage />, { route: '/forgot-password' });

    await user.type(screen.getByTestId('forgot-email'), 'someone@example.com');
    await user.click(screen.getByTestId('forgot-submit'));

    // Must NOT show an error — security contract.
    await waitFor(() => expect(screen.getByText('Check Your Email')).toBeInTheDocument());
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});

describe('ResetPasswordPage', () => {
  it('rejects mismatched passwords client-side', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password?token=abc' });

    await user.type(screen.getByTestId('reset-password'), 'password123');
    await user.type(screen.getByTestId('reset-confirm'), 'password456');
    await user.click(screen.getByTestId('reset-submit'));

    expect(screen.getByText('Passwords do not match')).toBeInTheDocument();
  });

  it('rejects a short password client-side', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password?token=abc' });

    await user.type(screen.getByTestId('reset-password'), 'short');
    await user.type(screen.getByTestId('reset-confirm'), 'short');
    await user.click(screen.getByTestId('reset-submit'));

    expect(screen.getByText('Password must be at least 8 characters')).toBeInTheDocument();
  });

  it('rejects a missing token client-side', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password' });

    await user.type(screen.getByTestId('reset-password'), 'password123');
    await user.type(screen.getByTestId('reset-confirm'), 'password123');
    await user.click(screen.getByTestId('reset-submit'));

    expect(screen.getByText('Invalid or missing reset token')).toBeInTheDocument();
  });

  it('submits the token + password and shows success on 200', async () => {
    let posted: { token: string; password: string } | undefined;
    server.use(
      http.post('*/api/auth/reset-password', async ({ request }) => {
        posted = await request.json() as { token: string; password: string };
        return HttpResponse.json({ message: 'ok' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<ResetPasswordPage />, { route: '/reset-password?token=tok-123' });

    await user.type(screen.getByTestId('reset-password'), 'newpassword123');
    await user.type(screen.getByTestId('reset-confirm'), 'newpassword123');
    await user.click(screen.getByTestId('reset-submit'));

    await waitFor(() => expect(screen.getByText('Your password has been reset successfully.')).toBeInTheDocument());
    expect(posted).toMatchObject({ token: 'tok-123', password: 'newpassword123' });
  });
});
