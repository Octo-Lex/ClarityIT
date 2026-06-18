import { useState } from 'react';
import { api, ApiError } from '../../api/client';

interface Props {
  onClose: () => void;
  onGenerated: () => void;
}

const ALL_SECTIONS = [
  { value: 'summary', label: 'Summary' },
  { value: 'milestones', label: 'Milestones' },
  { value: 'risks', label: 'Risks' },
  { value: 'incidents', label: 'Incidents' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'asset_actions', label: 'Asset Actions' },
  { value: 'remediations', label: 'Remediations' },
];

export default function StatusReportModal({ onClose, onGenerated }: Props) {
  const [title, setTitle] = useState('');
  const [periodStart, setPeriodStart] = useState('');
  const [periodEnd, setPeriodEnd] = useState('');
  const [selectedSections, setSelectedSections] = useState<string[]>(['summary', 'incidents', 'metrics']);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');
  const [generatedMarkdown, setGeneratedMarkdown] = useState('');
  const [artifactId, setArtifactId] = useState('');

  const toggleSection = (section: string) => {
    if (selectedSections.includes(section)) {
      setSelectedSections(selectedSections.filter(s => s !== section));
    } else {
      setSelectedSections([...selectedSections, section]);
    }
  };

  const handleGenerate = () => {
    if (!title.trim()) { setError('Title is required'); return; }
    if (!periodStart || !periodEnd) { setError('Date range is required'); return; }
    if (selectedSections.length === 0) { setError('Select at least one section'); return; }
    setGenerating(true);
    setError('');

    api.generateStatusReport({
      title, period_start: periodStart, period_end: periodEnd,
      include_sections: selectedSections,
    })
      .then((resp: any) => {
        setGenerating(false);
        setGeneratedMarkdown(resp.content_markdown || '');
        setArtifactId(resp.artifact_id || '');
        onGenerated();
      })
      .catch((e: unknown) => {
        if (e instanceof ApiError) setError(e.message);
        else setError('Failed to generate status report');
        setGenerating(false);
      });
  };

  const handleDownload = () => {
    const blob = new Blob([generatedMarkdown], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);
    const a = window.document.createElement('a');
    a.href = url;
    a.download = `${title.replace(/\s+/g, '-').toLowerCase()}.md`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" data-testid="status-report-modal">
      <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-lg p-6 w-full max-w-3xl max-h-[85vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Generate Status Report</h2>
          <button onClick={onClose} className="text-[var(--text-muted)] hover:text-white">✕</button>
        </div>

        {error && <div className="text-red-400 text-sm mb-3">{error}</div>}

        {!generatedMarkdown ? (
          <div className="space-y-4">
            <div>
              <label className="text-xs text-[var(--text-muted)]">Title</label>
              <input type="text" value={title} onChange={(e) => setTitle(e.target.value)}
                className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
                placeholder="Weekly Platform Status" data-testid="report-title" />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs text-[var(--text-muted)]">Period Start</label>
                <input type="date" value={periodStart} onChange={(e) => setPeriodStart(e.target.value)}
                  className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
                  data-testid="report-period-start" />
              </div>
              <div>
                <label className="text-xs text-[var(--text-muted)]">Period End</label>
                <input type="date" value={periodEnd} onChange={(e) => setPeriodEnd(e.target.value)}
                  className="w-full bg-[var(--bg-input)] border border-[var(--border)] rounded px-3 py-1.5 text-sm"
                  data-testid="report-period-end" />
              </div>
            </div>

            <div>
              <label className="text-xs text-[var(--text-muted)]">Sections</label>
              <div className="grid grid-cols-2 gap-2 mt-1" data-testid="report-sections">
                {ALL_SECTIONS.map(s => (
                  <label key={s.value} className="flex items-center gap-2 text-sm">
                    <input type="checkbox" checked={selectedSections.includes(s.value)}
                      onChange={() => toggleSection(s.value)}
                      data-testid={`report-section-${s.value}`} />
                    {s.label}
                  </label>
                ))}
              </div>
            </div>

            <div className="flex justify-end">
              <button onClick={handleGenerate} disabled={generating}
                className="px-4 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50"
                data-testid="report-generate-btn">
                {generating ? 'Generating...' : 'Generate Report'}
              </button>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm text-green-400">✓ Report generated successfully</span>
              <div className="flex gap-2">
                <button onClick={handleDownload}
                  className="px-3 py-1 bg-green-700 text-white rounded text-sm hover:bg-green-600"
                  data-testid="report-download-md">Download Markdown</button>
                <button onClick={onClose}
                  className="px-3 py-1 bg-gray-700 rounded text-sm">Close</button>
              </div>
            </div>
            <pre className="bg-[var(--bg-input)] border border-[var(--border)] rounded p-3 text-xs font-mono overflow-auto max-h-[50vh]"
              data-testid="report-preview">{generatedMarkdown}</pre>
          </div>
        )}
      </div>
    </div>
  );
}
