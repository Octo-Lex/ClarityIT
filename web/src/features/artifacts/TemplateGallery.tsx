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

const FORMAT_OPTIONS = [
  { value: '', label: 'All Formats' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'document_json', label: 'Document' },
];

function DocBlockPreview({ block }: { block: any }) {
  switch (block.type) {
    case 'heading':
      const sizes: Record<number, string> = { 1: 'text-xl font-bold', 2: 'text-lg font-bold', 3: 'text-base font-bold', 4: 'text-sm font-bold', 5: 'text-xs font-bold', 6: 'text-xs font-semibold' };
      return <div className={sizes[block.level] || sizes[2]}>{block.text}</div>;
    case 'paragraph':
      return <p className="text-sm">{block.text}</p>;
    case 'bullets':
      return <ul className="list-disc pl-4 text-sm">{(block.items || []).map((it: string, i: number) => <li key={i}>{it}</li>)}</ul>;
    case 'numbered_list':
      return <ol className="list-decimal pl-4 text-sm">{(block.items || []).map((it: string, i: number) => <li key={i}>{it}</li>)}</ol>;
    case 'quote':
      return <blockquote className="border-l-2 border-gray-600 pl-2 italic text-sm">{block.text}</blockquote>;
    case 'callout':
      const variantColors: Record<string, string> = { info: 'bg-blue-900/30 text-blue-300', warning: 'bg-yellow-900/30 text-yellow-300', success: 'bg-green-900/30 text-green-300', error: 'bg-red-900/30 text-red-300', note: 'bg-gray-700 text-gray-300', tip: 'bg-purple-900/30 text-purple-300' };
      return <div className={`text-xs rounded px-2 py-1 ${variantColors[block.variant] || variantColors.note}`}>{block.text}</div>;
    case 'page_break':
      return <div className="text-center text-xs text-gray-500">— Page Break —</div>;
    default:
      return null;
  }
}

export default function TemplateGallery({ onClose, onInstantiated }: Props) {
  const [templates, setTemplates] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [formatFilter, setFormatFilter] = useState('');
  const [previewId, setPreviewId] = useState<string | null>(null);
  const [instantiating, setInstantiating] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);

  // Create form state
  const [ctName, setCtName] = useState('');
  const [ctType, setCtType] = useState('document');
  const [ctContent, setCtContent] = useState('');
  const [ctDesc, setCtDesc] = useState('');
  const [ctFormat, setCtFormat] = useState('markdown');
  const [ctDocJSON, setCtDocJSON] = useState('');

  const fetchTemplates = () => {
    setLoading(true);
    api.listTemplates(typeFilter || undefined, formatFilter || undefined)
      .then((data: any[]) => { setTemplates(data || []); setLoading(false); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to load templates');
        setLoading(false);
      });
  };

  useEffect(() => { fetchTemplates(); }, [typeFilter, formatFilter]);

  const handleInstantiate = (template: any) => {
    setInstantiating(true);
    setError('');
    api.instantiateTemplate(template.id, {})
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
    const payload: any = {
      template_type: ctType,
      name: ctName,
      description: ctDesc,
      template_format: ctFormat,
    };
    if (ctFormat === 'markdown') {
      if (!ctContent.trim()) { setError('Content is required for markdown templates'); return; }
      payload.content_markdown = ctContent;
    } else {
      try {
        payload.document_json = JSON.parse(ctDocJSON);
        payload.schema_version = 1;
      } catch {
        setError('Invalid JSON for document_json'); return;
      }
    }
    api.createTemplate(payload)
      .then(() => { setShowCreateForm(false); setCtName(''); setCtContent(''); setCtDesc(''); setCtDocJSON(''); fetchTemplates(); })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to create template');
      });
  };

  const previewTemplate = templates.find(t => t.id === previewId);
  const isDocTemplate = previewTemplate?.template_format === 'document_json';

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
            <div className="flex gap-2">
              <select value={ctType} onChange={(e) => setCtType(e.target.value)}
                className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
                data-testid="template-form-type">
                {Object.entries(TYPE_LABELS).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
              </select>
              <select value={ctFormat} onChange={(e) => setCtFormat(e.target.value)}
                className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
                data-testid="template-form-format">
                <option value="markdown">Markdown</option>
                <option value="document_json">Document (JSON)</option>
              </select>
            </div>
            <input type="text" placeholder="Description (optional)" value={ctDesc}
              onChange={(e) => setCtDesc(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              data-testid="template-form-desc" />
            {ctFormat === 'markdown' ? (
              <textarea placeholder="Markdown content..." value={ctContent}
                onChange={(e) => setCtContent(e.target.value)} rows={6}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-2 text-sm font-mono"
                data-testid="template-form-content" />
            ) : (
              <textarea placeholder='{"schema_version": 1, "title": "...", "document_type": "general_document", "blocks": [...]}' value={ctDocJSON}
                onChange={(e) => setCtDocJSON(e.target.value)} rows={8}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-2 text-sm font-mono"
                data-testid="template-form-doc-json" />
            )}
            <button onClick={handleCreateTemplate}
              className="px-3 py-1 bg-green-600 text-white rounded text-sm"
              data-testid="template-form-save">Save Template</button>
          </div>
        )}

        {/* Filters */}
        <div className="flex gap-2 mb-3">
          <select value={typeFilter} onChange={(e) => setTypeFilter(e.target.value)}
            className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm"
            data-testid="template-filter">
            {FILTER_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
          <select value={formatFilter} onChange={(e) => setFormatFilter(e.target.value)}
            className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-sm"
            data-testid="template-format-filter">
            {FORMAT_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
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
                <div className="flex items-center gap-2 mb-1 flex-wrap">
                  {t.is_system ? (
                    <span className="px-1.5 py-0.5 text-xs rounded bg-blue-900/40 text-blue-300" data-testid={`template-badge-system-${t.id}`}>SYSTEM</span>
                  ) : (
                    <span className="px-1.5 py-0.5 text-xs rounded bg-green-900/40 text-green-300" data-testid={`template-badge-team-${t.id}`}>TEAM</span>
                  )}
                  {t.template_format === 'document_json' ? (
                    <span className="px-1.5 py-0.5 text-xs rounded bg-purple-900/40 text-purple-300" data-testid={`template-badge-doc-${t.id}`}>DOCUMENT</span>
                  ) : (
                    <span className="px-1.5 py-0.5 text-xs rounded bg-gray-700 text-gray-300" data-testid={`template-badge-md-${t.id}`}>MARKDOWN</span>
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
              {isDocTemplate && <span className="px-1.5 py-0.5 text-xs rounded bg-purple-900/40 text-purple-300">DOCUMENT</span>}
            </div>

            {isDocTemplate ? (
              <div className="border border-[var(--border)] rounded p-3 space-y-2 max-h-[40vh] overflow-y-auto mb-3"
                data-testid="template-preview-doc">
                {(previewTemplate.document_json?.blocks || []).map((blk: any, i: number) => (
                  <DocBlockPreview key={i} block={blk} />
                ))}
              </div>
            ) : (
              <pre className="bg-[var(--bg-input)] border border-[var(--border)] rounded p-3 text-xs font-mono overflow-auto max-h-[40vh] mb-3"
                data-testid="template-preview-content">{previewTemplate.content_markdown}</pre>
            )}

            <button
              onClick={() => handleInstantiate(previewTemplate)}
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
