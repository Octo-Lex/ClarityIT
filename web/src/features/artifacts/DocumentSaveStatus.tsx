interface Props {
  status: 'saved' | 'unsaved' | 'saving' | 'error';
  lastSaved?: string | null;
}

export default function DocumentSaveStatus({ status, lastSaved }: Props) {
  const config = {
    saved: { color: 'text-green-400', icon: '✓', label: 'Saved' },
    unsaved: { color: 'text-yellow-400', icon: '●', label: 'Unsaved changes' },
    saving: { color: 'text-blue-400', icon: '⟳', label: 'Saving...' },
    error: { color: 'text-red-400', icon: '✗', label: 'Save failed' },
  };
  const c = config[status];
  return (
    <div className="flex items-center gap-2 text-xs" data-testid="save-status">
      <span className={c.color}>{c.icon}</span>
      <span className={c.color}>{c.label}</span>
      {lastSaved && status === 'saved' && (
        <span className="text-[var(--text-muted)]">at {new Date(lastSaved).toLocaleTimeString()}</span>
      )}
    </div>
  );
}
