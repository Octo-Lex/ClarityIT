import { useState, useEffect } from 'react';
import { api, ApiError } from '../../api/client';

interface Props {
  onClose: () => void;
  onGenerated: () => void;
}

const TEMPLATES = [
  { value: 'default', label: 'Default' },
  { value: 'modern', label: 'Modern' },
  { value: 'minimal', label: 'Minimal' },
  { value: 'corporate', label: 'Corporate' },
];

const TONES = [
  { value: 'professional', label: 'Professional' },
  { value: 'casual', label: 'Casual' },
  { value: 'confident', label: 'Confident' },
  { value: 'educational', label: 'Educational' },
];

const LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'es', label: 'Spanish' },
  { value: 'fr', label: 'French' },
  { value: 'de', label: 'German' },
  { value: 'ja', label: 'Japanese' },
];

export default function PresentationModal({ onClose, onGenerated }: Props) {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [numSlides, setNumSlides] = useState(8);
  const [template, setTemplate] = useState('default');
  const [tone, setTone] = useState('professional');
  const [language, setLanguage] = useState('en');
  const [exportAs, setExportAs] = useState('pptx');
  const [instructions, setInstructions] = useState('');
  const [status, setStatus] = useState<{ enabled: boolean; reachable: boolean; message: string } | null>(null);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api.getPresentonStatus()
      .then((s: any) => setStatus(s))
      .catch(() => setStatus({ enabled: false, reachable: false, message: 'Unable to check status' }));
  }, []);

  const handleGenerate = () => {
    if (!title.trim()) { setError('Title is required'); return; }
    if (!content.trim()) { setError('Content is required'); return; }
    setGenerating(true);
    setError('');

    api.generatePresentation({
      title, content, num_slides: numSlides,
      template, tone, language, export_as: exportAs,
      instructions: instructions || undefined,
    })
      .then(() => {
        setGenerating(false);
        onGenerated();
        onClose();
      })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to generate presentation');
        setGenerating(false);
      });
  };

  const disabled = status && (!status.enabled || !status.reachable);

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" data-testid="presentation-modal">
      <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-lg p-6 w-full max-w-2xl max-h-[85vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Generate Presentation</h2>
          <button onClick={onClose} className="text-[var(--text-muted)] hover:text-white">✕</button>
        </div>

        {/* Status banner */}
        {status && !status.enabled && (
          <div className="bg-yellow-900/30 border border-yellow-700/50 rounded p-3 mb-4 text-sm text-yellow-300" data-testid="presenton-disabled-banner">
            {status.message}
          </div>
        )}
        {status && status.enabled && !status.reachable && (
          <div className="bg-orange-900/30 border border-orange-700/50 rounded p-3 mb-4 text-sm text-orange-300" data-testid="presenton-unreachable-banner">
            {status.message}
          </div>
        )}

        {error && <div className="text-red-400 text-sm mb-3">{error}</div>}

        <div className="space-y-3">
          <div>
            <label className="text-xs text-[var(--text-muted)]">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              placeholder="Weekly Team Update"
              data-testid="presentation-title"
            />
          </div>

          <div>
            <label className="text-xs text-[var(--text-muted)]">Content / Prompt</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              rows={4}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-2 text-sm"
              placeholder="Describe the content of your presentation..."
              data-testid="presentation-content"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-[var(--text-muted)]">Slides (1-30)</label>
              <input
                type="number"
                min={1}
                max={30}
                value={numSlides}
                onChange={(e) => setNumSlides(Math.min(30, Math.max(1, parseInt(e.target.value) || 1)))}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
                data-testid="presentation-slides"
              />
            </div>
            <div>
              <label className="text-xs text-[var(--text-muted)]">Export Format</label>
              <select
                value={exportAs}
                onChange={(e) => setExportAs(e.target.value)}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
                data-testid="presentation-format"
              >
                <option value="pptx">PPTX (PowerPoint)</option>
                <option value="pdf">PDF</option>
              </select>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="text-xs text-[var(--text-muted)]">Template</label>
              <select
                value={template}
                onChange={(e) => setTemplate(e.target.value)}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
                data-testid="presentation-template"
              >
                {TEMPLATES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </div>
            <div>
              <label className="text-xs text-[var(--text-muted)]">Tone</label>
              <select
                value={tone}
                onChange={(e) => setTone(e.target.value)}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
                data-testid="presentation-tone"
              >
                {TONES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
              </select>
            </div>
            <div>
              <label className="text-xs text-[var(--text-muted)]">Language</label>
              <select
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1.5 text-sm"
                data-testid="presentation-language"
              >
                {LANGUAGES.map(l => <option key={l.value} value={l.value}>{l.label}</option>)}
              </select>
            </div>
          </div>

          <div>
            <label className="text-xs text-[var(--text-muted)]">Instructions (optional)</label>
            <input
              type="text"
              value={instructions}
              onChange={(e) => setInstructions(e.target.value)}
              className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
              placeholder="Additional guidance for the AI..."
              data-testid="presentation-instructions"
            />
          </div>

          <div className="flex gap-2 justify-end pt-2">
            <button
              onClick={handleGenerate}
              disabled={generating || !!disabled}
              className="px-4 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50"
              data-testid="presentation-generate-btn"
            >
              {generating ? (
                <span data-testid="presentation-spinner">Generating...</span>
              ) : 'Generate'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
