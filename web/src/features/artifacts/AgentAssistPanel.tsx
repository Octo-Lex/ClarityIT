import { useState } from 'react';
import { api, ApiError } from '../../api/client';

const ASSIST_MODES = [
  { value: 'rewrite', label: 'Rewrite' },
  { value: 'summarize', label: 'Summarize' },
  { value: 'expand', label: 'Expand' },
  { value: 'make_concise', label: 'Make Concise' },
  { value: 'make_executive', label: 'Make Executive' },
  { value: 'make_technical', label: 'Make Technical' },
  { value: 'draft_section', label: 'Draft Section' },
  { value: 'create_outline', label: 'Create Outline' },
  { value: 'extract_action_items', label: 'Extract Action Items' },
  { value: 'improve_headings', label: 'Improve Headings' },
];

interface SuggestedBlock {
  id?: string;
  type: string;
  level?: number;
  text?: string;
  items?: string[];
  variant?: string;
}

interface Props {
  artifactId: string;
  selectedBlockId: string | null;
  selectedBlockText: string;
  documentType: string;
  onInsertBelow: (blockIndex: number, blocks: SuggestedBlock[]) => void;
  onReplaceBlock: (blockIndex: number, blocks: SuggestedBlock[]) => void;
}

export default function AgentAssistPanel({
  artifactId, selectedBlockId, selectedBlockText, documentType,
  onInsertBelow, onReplaceBlock,
}: Props) {
  const [mode, setMode] = useState('rewrite');
  const [instruction, setInstruction] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [suggestion, setSuggestion] = useState<{ suggested_blocks: SuggestedBlock[]; summary: string } | null>(null);
  const [copied, setCopied] = useState(false);

  const handleSubmit = async () => {
    setLoading(true);
    setError('');
    setSuggestion(null);
    try {
      const result = await api.documentAssist(artifactId, {
        mode,
        block_id: selectedBlockId || '',
        selected_text: selectedBlockText || '',
        instruction,
        document_type: documentType,
        max_words: 300,
      });
      setSuggestion(result);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to get suggestion');
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = () => {
    if (!suggestion) return;
    const text = suggestion.suggested_blocks
      .map(b => b.text || (b.items || []).join('\n'))
      .join('\n\n');
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDismiss = () => {
    setSuggestion(null);
    setError('');
  };

  // Find block index from ID (passed by parent)
  const blockIndex = selectedBlockId ? parseInt(selectedBlockId.split('_').pop() || '0', 36) : 0;

  return (
    <div className="w-72 border-l border-[var(--border)] flex flex-col overflow-hidden" data-testid="agent-assist">
      <div className="p-3 border-b border-[var(--border)]">
        <h3 className="text-sm font-semibold flex items-center gap-1">
          <span>🤖</span> Agent Assist
        </h3>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {/* Mode selector */}
        <div>
          <label className="text-xs text-[var(--text-muted)] block mb-1">Mode</label>
          <select
            data-testid="assist-mode-select"
            value={mode}
            onChange={e => setMode(e.target.value)}
            className="w-full text-sm bg-[var(--card)] border border-[var(--border)] rounded px-2 py-1"
          >
            {ASSIST_MODES.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
          </select>
        </div>

        {/* Selected block context */}
        {selectedBlockId && (
          <div className="text-xs text-[var(--text-muted)] bg-[var(--card)] border border-[var(--border)] rounded p-2">
            <span className="font-medium">Block:</span> {selectedBlockId}
          </div>
        )}

        {/* Instruction */}
        <div>
          <label className="text-xs text-[var(--text-muted)] block mb-1">Instruction (optional)</label>
          <textarea
            data-testid="assist-instruction"
            value={instruction}
            onChange={e => setInstruction(e.target.value)}
            placeholder="e.g., Make this clearer..."
            rows={2}
            className="w-full text-sm bg-[var(--card)] border border-[var(--border)] rounded px-2 py-1 resize-y"
          />
        </div>

        {/* Submit */}
        <button
          data-testid="assist-submit"
          onClick={handleSubmit}
          disabled={loading}
          className="w-full px-3 py-1.5 text-sm bg-[var(--primary)] text-white rounded hover:opacity-90 disabled:opacity-50"
        >
          {loading ? 'Generating...' : 'Get Suggestion'}
        </button>

        {/* Loading state */}
        {loading && (
          <div className="text-center text-xs text-[var(--text-muted)] py-2" data-testid="assist-loading">
            Asking the agent...
          </div>
        )}

        {/* Error state */}
        {error && (
          <div className="text-xs text-red-400 bg-red-950/30 border border-red-900 rounded p-2" data-testid="assist-error">
            {error}
          </div>
        )}

        {/* Suggestion preview */}
        {suggestion && (
          <div data-testid="assist-suggestion" className="space-y-2">
            <div className="text-xs text-[var(--text-muted)] italic">{suggestion.summary}</div>
            <div className="border border-[var(--border)] rounded p-2 space-y-1 bg-[var(--card)]">
              {suggestion.suggested_blocks.map((blk, i) => (
                <div key={i} className="text-sm">
                  {blk.type === 'heading' && <div className="font-bold">{blk.text}</div>}
                  {blk.type === 'paragraph' && <p>{blk.text}</p>}
                  {blk.type === 'bullets' && <ul className="list-disc pl-4">{(blk.items || []).map((it, j) => <li key={j}>{it}</li>)}</ul>}
                  {blk.type === 'numbered_list' && <ol className="list-decimal pl-4">{(blk.items || []).map((it, j) => <li key={j}>{it}</li>)}</ol>}
                  {blk.type === 'quote' && <blockquote className="border-l-2 border-[var(--border)] pl-2 italic">{blk.text}</blockquote>}
                  {blk.type === 'callout' && <div className="text-xs">{blk.variant}: {blk.text}</div>}
                  {blk.type === 'page_break' && <div className="text-center text-xs text-[var(--text-muted)]">— Page Break —</div>}
                </div>
              ))}
            </div>

            {/* Actions */}
            <div className="flex flex-wrap gap-1">
              {selectedBlockId && (
                <button
                  data-testid="assist-insert-below"
                  onClick={() => onInsertBelow(blockIndex, suggestion.suggested_blocks)}
                  className="px-2 py-1 text-xs bg-green-900 text-white rounded hover:bg-green-800"
                >Insert Below</button>
              )}
              {selectedBlockId && (
                <button
                  data-testid="assist-replace-block"
                  onClick={() => onReplaceBlock(blockIndex, suggestion.suggested_blocks)}
                  className="px-2 py-1 text-xs bg-blue-900 text-white rounded hover:bg-blue-800"
                >Replace Block</button>
              )}
              <button
                data-testid="assist-copy"
                onClick={handleCopy}
                className="px-2 py-1 text-xs bg-[var(--card)] border border-[var(--border)] rounded hover:bg-[var(--border)]"
              >{copied ? '✓ Copied' : 'Copy'}</button>
              <button
                data-testid="assist-dismiss"
                onClick={handleDismiss}
                className="px-2 py-1 text-xs bg-[var(--card)] border border-[var(--border)] rounded hover:bg-[var(--border)]"
              >Dismiss</button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
