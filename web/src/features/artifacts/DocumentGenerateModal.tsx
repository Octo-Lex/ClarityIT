import { useState } from 'react';
import { api, ApiError } from '../../api/client';

const DOC_TYPES = [
  { value: 'general_document', label: 'General Document' },
  { value: 'decision_memo', label: 'Decision Memo' },
  { value: 'implementation_plan', label: 'Implementation Plan' },
  { value: 'incident_summary', label: 'Incident Summary' },
  { value: 'training_doc', label: 'Training Doc' },
  { value: 'architecture_doc', label: 'Architecture Doc' },
  { value: 'project_report', label: 'Project Report' },
  { value: 'status_report', label: 'Status Report' },
  { value: 'meeting_summary', label: 'Meeting Summary' },
  { value: 'executive_brief', label: 'Executive Brief' },
];

const TONES = [
  { value: 'technical', label: 'Technical' },
  { value: 'executive', label: 'Executive' },
  { value: 'casual', label: 'Casual' },
  { value: 'formal', label: 'Formal' },
];

interface Props {
  onClose: () => void;
  onGenerated: (artifactId: string) => void;
}

export default function DocumentGenerateModal({ onClose, onGenerated }: Props) {
  const [title, setTitle] = useState('');
  const [docType, setDocType] = useState('implementation_plan');
  const [prompt, setPrompt] = useState('');
  const [tone, setTone] = useState('technical');
  const [sections, setSections] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const addSection = () => {
    if (sections.length >= 20) return;
    setSections([...sections, '']);
  };

  const updateSection = (i: number, val: string) => {
    setSections(sections.map((s, idx) => idx === i ? val : s));
  };

  const removeSection = (i: number) => {
    setSections(sections.filter((_, idx) => idx !== i));
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError('');
    try {
      const result = await api.generateDocument({
        title,
        document_type: docType,
        prompt,
        tone,
        sections: sections.filter(s => s.trim()),
      });
      onGenerated(result.artifact_id as string);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to generate document');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50" data-testid="generate-modal">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-lg max-h-[85vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Generate Document</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground text-xl" data-testid="generate-close">×</button>
        </div>

        {/* Title */}
        <div className="mb-3">
          <label className="text-xs text-muted-foreground block mb-1">Title *</label>
          <input
            data-testid="generate-title"
            type="text"
            value={title}
            onChange={e => setTitle(e.target.value)}
            maxLength={200}
            placeholder="e.g., Q3 Platform Implementation Plan"
            className="w-full text-sm bg-surface border border-border rounded px-2 py-1.5"
          />
        </div>

        {/* Document type */}
        <div className="mb-3">
          <label className="text-xs text-muted-foreground block mb-1">Document Type *</label>
          <select
            data-testid="generate-doc-type"
            value={docType}
            onChange={e => setDocType(e.target.value)}
            className="w-full text-sm bg-surface border border-border rounded px-2 py-1.5"
          >
            {DOC_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
          </select>
        </div>

        {/* Prompt */}
        <div className="mb-3">
          <label className="text-xs text-muted-foreground block mb-1">Prompt *</label>
          <textarea
            data-testid="generate-prompt"
            value={prompt}
            onChange={e => setPrompt(e.target.value)}
            maxLength={2000}
            rows={3}
            placeholder="Describe what the document should cover..."
            className="w-full text-sm bg-surface border border-border rounded px-2 py-1.5 resize-y"
          />
        </div>

        {/* Tone */}
        <div className="mb-3">
          <label className="text-xs text-muted-foreground block mb-1">Tone</label>
          <select
            data-testid="generate-tone"
            value={tone}
            onChange={e => setTone(e.target.value)}
            className="w-full text-sm bg-surface border border-border rounded px-2 py-1.5"
          >
            {TONES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
          </select>
        </div>

        {/* Sections */}
        <div className="mb-3">
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs text-muted-foreground">Sections (optional)</label>
            <button
              data-testid="generate-add-section"
              onClick={addSection}
              disabled={sections.length >= 20}
              className="text-xs px-2 py-0.5 bg-surface border border-border rounded hover:bg-muted disabled:opacity-50"
            >+ Add</button>
          </div>
          {sections.map((s, i) => (
            <div key={i} className="flex gap-1 mb-1">
              <input
                data-testid={`generate-section-${i}`}
                type="text"
                value={s}
                onChange={e => updateSection(i, e.target.value)}
                maxLength={100}
                placeholder={`Section ${i + 1}`}
                className="flex-1 text-sm bg-surface border border-border rounded px-2 py-1"
              />
              <button
                data-testid={`generate-remove-section-${i}`}
                onClick={() => removeSection(i)}
                className="text-xs px-2 py-1 text-destructive hover:text-destructive"
              >×</button>
            </div>
          ))}
        </div>

        {/* Error */}
        {error && (
          <div className="text-xs text-destructive bg-destructive/10 border border-destructive/40 rounded p-2 mb-3" data-testid="generate-error">
            {error}
          </div>
        )}

        {/* Loading */}
        {loading && (
          <div className="text-center text-xs text-muted-foreground py-2" data-testid="generate-loading">
            Generating document...
          </div>
        )}

        {/* Actions */}
        <div className="flex gap-2 justify-end">
          <button
            onClick={onClose}
            disabled={loading}
            className="px-3 py-1.5 text-sm bg-surface border border-border rounded hover:bg-muted"
          >Cancel</button>
          <button
            data-testid="generate-submit"
            onClick={handleSubmit}
            disabled={loading || !title.trim() || !prompt.trim()}
            className="px-3 py-1.5 text-sm bg-primary text-white rounded hover:opacity-90 disabled:opacity-50"
          >{loading ? 'Generating...' : 'Generate'}</button>
        </div>
      </div>
    </div>
  );
}
