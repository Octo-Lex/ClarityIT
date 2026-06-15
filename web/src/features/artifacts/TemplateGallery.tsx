import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface Props {
  onClose: () => void;
  onInstantiated: (artifactId: string) => void;
}

const TYPE_LABELS: Record<string, string> = {
  document: 'Document',
  report: 'Report',
  meeting_summary: 'Meeting Summary',
  status_report: 'Status Report',
  decision_memo: 'Decision Memo',
  training_deck: 'Training Deck',
  presentation: 'Presentation',
};

const FILTER_OPTIONS = [
  { value: '', label: 'All Types' },
  { value: 'document', label: 'Document' },
  { value: 'report', label: 'Report' },
  { value: 'meeting_summary', label: 'Meeting Summary' },
  { value: 'status_report', label: 'Status Report' },
  { value: 'decision_memo', label: 'Decision Memo' },
  { value: 'training_deck', label: 'Training Deck' },
];

export default function TemplateGallery({ onClose, onInstantiated }: Props) {
  const [templates, setTemplates] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [previewId, setPreviewId] = useState<string | null>(null);
  const [instantiating, setInstantiating] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);

  // Create form state
  const [ctName, setCtName] = useState('');
  const [ctType, setCtType] = useState('document');
  const [ctContent, setCtContent] = useState('');
  const [ctDesc, setCtDesc] = useState('');

  const fetchTemplates = () => {
    setLoading(true);
    api.listTemplates(typeFilter || undefined)
      .then((data: any[]) => { setTemplates(data || []); setLoading(false); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to load templates');
        setLoading(false);
      });
  };

  useEffect(() => { fetchTemplates(); }, [typeFilter]);

  const handleInstantiate = (templateId: string) => {
    setInstantiating(true);
    setError('');
    api.instantiateTemplate(templateId, {})
      .then((resp: any) => {
        setInstantiating(false);
        onInstantiated(resp.artifact_id);
        onClose();
      })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to instantiate template');
        setInstantiating(false);
      });
  };

  const handleCreateTemplate = () => {
    if (!ctName.trim()) { setError('Name is required'); return; }
    if (!ctContent.trim()) { setError('Content is required'); return; }
    api.createTemplate({ template_type: ctType, name: ctName, content_markdown: ctContent, description: ctDesc })
      .then(() => { setShowCreateForm(false); setCtName(''); setCtContent(''); setCtDesc(''); fetchTemplates(); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to create template');
      });
  };

  const previewTemplate = templates.find(t => t.id === previewId);

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" data-testid="template-gallery">
      <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-lg p-6 w-full max-w-4xl max-h-[85vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Template Library</h2>
          <div className="flex gap-2">
            <button onClick={() => setShowCreateForm(!showCreateForm)}
              className="px-3 py-1 bg-blue-600 text-white rounded text-sm"
              data-testid="template-create-btn">+ New Template</button>
            <button onClick={onClose} className="text-[var(--text-muted)] hover:text-white">✕</button>
          </div>
        </div>

        {error && <div className="text-red-400 text-sm mb-3" data-testid="template-error">{error}</div>}

        {/* Create form */}
        {showCreateForm && (
          <div className="border border-[var(--border)] rounded p-4 mb-4 space-y-2" data-testid="template-create-form">
            <h3 className="text-sm font-semibold">Create Custom Template</h3>
            <input type="text" placeholder="Template name" value={ctName}
              onChange={(e) => setCtName(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              data-testid="template-form-name" />
            <select value={ctType} onChange={(e) => setCtType(e.target.value)}
              className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
              data-testid="template-form-type">
              {Object.entries(TYPE_LABELS).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </select>
            <input type="text" placeholder="Description (optional)" value={ctDesc}
              onChange={(e) => setCtDesc(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              data-testid="template-form-desc" />
            <textarea placeholder="Markdown content..." value={ctContent}
              onChange={(e) => setCtContent(e.target.value)} rows={6}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-2 text-sm font-mono"
              data-testid="template-form-content" />
            <button onClick={handleCreateTemplate}
              className="px-3 py-1 bg-green-600 text-white rounded text-sm"
              data-testid="template-form-save">Save Template</button>
          </div>
        )}

        {/* Filter */}
        <div className="mb-3">
          <select value={typeFilter} onChange={(e) => setTypeFilter(e.target.value)}
            className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm"
            data-testid="template-filter">
            {FILTER_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </div>

        {/* Loading/empty */}
        {loading && <div className="text-[var(--text-muted)] text-sm">Loading...</div>}
        {!loading && templates.length === 0 && (
          <div className="card p-8 text-center" data-testid="template-empty">
            <p className="text-[var(--text-muted)]">No templates found.</p>
          </div>
        )}

        {/* Template cards */}
        {!loading && templates.length > 0 && !previewTemplate && (
          <div className="grid grid-cols-2 gap-3" data-testid="template-list">
            {templates.map(t => (
              <div key={t.id} className="card p-3 cursor-pointer hover:border-blue-600"
                onClick={() => setPreviewId(t.id)}
                data-testid={`template-card-${t.id}`}>
                <div className="flex items-center gap-2 mb-1">
                  {t.is_system ? (
                    <span className="px-1.5 py-0.5 text-xs rounded bg-blue-900/40 text-blue-300" data-testid={`template-badge-system-${t.id}`}>SYSTEM</span>
                  ) : (
                    <span className="px-1.5 py-0.5 text-xs rounded bg-green-900/40 text-green-300" data-testid={`template-badge-team-${t.id}`}>TEAM</span>
                  )}
                  <span className="text-xs text-[var(--text-muted)]">{TYPE_LABELS[t.template_type] || t.template_type}</span>
                </div>
                <div className="text-sm font-medium">{t.name}</div>
                {t.description && <div className="text-xs text-[var(--text-muted)] truncate">{t.description}</div>}
              </div>
            ))}
          </div>
        )}

        {/* Preview + Use */}
        {previewTemplate && (
          <div data-testid="template-preview-section">
            <div className="flex items-center gap-2 mb-2">
              <button onClick={() => setPreviewId(null)} className="text-xs text-blue-400">← Back</button>
              <span className="text-sm font-medium">{previewTemplate.name}</span>
              {previewTemplate.is_system && <span className="px-1.5 py-0.5 text-xs rounded bg-blue-900/40 text-blue-300">SYSTEM</span>}
            </div>
            <pre className="bg-[var(--bg-input)] border border-[var(--border)] rounded p-3 text-xs font-mono overflow-auto max-h-[40vh] mb-3"
              data-testid="template-preview-content">{previewTemplate.content_markdown}</pre>
            <button
              onClick={() => handleInstantiate(previewTemplate.id)}
              disabled={instantiating}
              className="px-4 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50"
              data-testid="template-use-btn">
              {instantiating ? 'Creating...' : 'Use Template'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
