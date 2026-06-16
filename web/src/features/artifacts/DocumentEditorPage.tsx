import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api, ApiError } from '../../api/client';
import { useAuth } from '../../auth/context';
import DocumentBlockEditor, { Block, genId } from './DocumentBlockEditor';
import DocumentOutline from './DocumentOutline';
import DocumentToolbar, { BlockType } from './DocumentToolbar';
import DocumentSaveStatus from './DocumentSaveStatus';
import AgentAssistPanel from './AgentAssistPanel';
import VersionHistoryDrawer from './VersionHistoryDrawer';

const DOCUMENT_TYPE_LABELS: Record<string, string> = {
  general_document: 'General Document',
  decision_memo: 'Decision Memo',
  implementation_plan: 'Implementation Plan',
  incident_summary: 'Incident Summary',
  training_doc: 'Training Document',
  architecture_doc: 'Architecture Document',
  project_report: 'Project Report',
  status_report: 'Status Report',
  meeting_summary: 'Meeting Summary',
  executive_brief: 'Executive Brief',
};

type SaveState = 'saved' | 'unsaved' | 'saving' | 'error';

export default function DocumentEditorPage() {
  const { teamId, artifactId } = useParams<{ teamId: string; artifactId: string }>();
  const navigate = useNavigate();

  const [doc, setDoc] = useState<any>(null);
  const [title, setTitle] = useState('');
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [docType, setDocType] = useState('general_document');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [saveState, setSaveState] = useState<SaveState>('saved');
  const [lastSaved, setLastSaved] = useState<string | null>(null);
  const [previewMode, setPreviewMode] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [showAssist, setShowAssist] = useState(false);
  const [selectedBlockId, setSelectedBlockId] = useState<string | null>(null);
  const [selectedBlockText, setSelectedBlockText] = useState('');
  const [archived, setArchived] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState('');
  const [showVersions, setShowVersions] = useState(false);
  const auth = useAuth();

  // ─── Load document ───
  useEffect(() => {
    if (!artifactId) return;
    setLoading(true);
    setLoadError('');
    api.getDocument(artifactId)
      .then(data => {
        setDoc(data);
        setTitle(data.title || '');
        setDocType(data.document_type || 'general_document');
        const dj = data.document_json || {};
        setBlocks(dj.blocks || []);
        setLoading(false);
      })
      .catch(err => {
        setLoadError(err instanceof ApiError ? err.message : 'Failed to load document');
        setLoading(false);
      });
  }, [artifactId]);

  // ─── Dirty tracking ───
  const markDirty = useCallback(() => {
    setDirty(true);
    setSaveState('unsaved');
  }, []);

  // ─── Unsaved navigation warning ───
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (dirty) { e.preventDefault(); e.returnValue = ''; }
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [dirty]);

  // ─── Block operations ───
  const updateBlock = (index: number, updated: Block) => {
    setBlocks(prev => { const next = [...prev]; next[index] = updated; return next; });
    markDirty();
  };

  // ─── Assist handlers ───
  const handleInsertBelow = (blockIndex: number, suggestedBlocks: any[]) => {
    const newBlocks = suggestedBlocks.map(b => ({ ...b, id: genId() }));
    setBlocks(prev => {
      const next = [...prev];
      next.splice(blockIndex + 1, 0, ...newBlocks);
      return next;
    });
    markDirty();
  };

  const handleReplaceBlock = (blockIndex: number, suggestedBlocks: any[]) => {
    const newBlocks = suggestedBlocks.map(b => ({ ...b, id: genId() }));
    setBlocks(prev => {
      const next = [...prev];
      next.splice(blockIndex, 1, ...newBlocks);
      return next;
    });
    markDirty();
  };

  const handleBlockFocus = (index: number) => {
    const blk = blocks[index];
    if (blk) {
      setSelectedBlockId(blk.id);
      setSelectedBlockText(blk.text || '');
    }
  };

  const deleteBlock = (index: number) => {
    setBlocks(prev => prev.filter((_, i) => i !== index));
    markDirty();
  };

  const moveBlock = (index: number, dir: 'up' | 'down') => {
    setBlocks(prev => {
      const next = [...prev];
      const target = dir === 'up' ? index - 1 : index + 1;
      if (target < 0 || target >= next.length) return prev;
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
    markDirty();
  };

  const duplicateBlock = (index: number) => {
    setBlocks(prev => {
      const copy = { ...prev[index], id: genId() };
      const next = [...prev];
      next.splice(index + 1, 0, copy);
      return next;
    });
    markDirty();
  };

  const addBlock = (type: BlockType) => {
    const newBlock: Block = { id: genId(), type } as Block;
    switch (type) {
      case 'heading': newBlock.level = 2; newBlock.text = ''; break;
      case 'paragraph': newBlock.text = ''; break;
      case 'bullets': case 'numbered_list': newBlock.items = ['']; break;
      case 'table': newBlock.headers = ['Column 1']; newBlock.rows = [['']]; break;
      case 'quote': newBlock.text = ''; break;
      case 'callout': newBlock.variant = 'info'; newBlock.text = ''; break;
      case 'page_break': break;
    }
    setBlocks(prev => [...prev, newBlock]);
    markDirty();
  };

  // ─── Save ───
  const handleSave = async () => {
    if (!artifactId) return;
    setSaveState('saving');
    try {
      const documentJson = {
        schema_version: 1,
        title,
        document_type: docType,
        blocks,
      };
      await api.updateDocument(artifactId, { title, document_json: documentJson });
      setDirty(false);
      setSaveState('saved');
      setLastSaved(new Date().toISOString());
    } catch (err) {
      setSaveState('error');
    }
  };

  // ─── Export (v1.4 Track 6) ───
  const doExport = async (format: 'markdown' | 'pdf' | 'docx') => {
    if (!artifactId) return;
    setExporting(true);
    setExportError('');
    try {
      const token = auth.token;
      if (!token) throw new Error('Not authenticated');
      const url = api.exportDocumentUrl(artifactId, format);
      const resp = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!resp.ok) throw new Error('Export failed');
      const blob = await resp.blob();
      const ext = format === 'markdown' ? 'md' : format;
      const downloadUrl = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = downloadUrl;
      a.download = `${title || 'document'}.${ext}`;
      a.click();
      URL.revokeObjectURL(downloadUrl);
    } catch (err) {
      setExportError('Export failed');
    } finally {
      setExporting(false);
    }
  };

  const handleTitleChange = (newTitle: string) => {
    setTitle(newTitle);
    markDirty();
  };

  // ─── Render ───
  if (loading) {
    return <div className="flex items-center justify-center h-full text-[var(--text-muted)]" data-testid="doc-loading">Loading document...</div>;
  }

  if (loadError) {
    return (
      <div className="flex flex-col items-center justify-center h-full" data-testid="doc-error">
        <p className="text-red-400 mb-4">{loadError}</p>
        <button onClick={() => navigate(-1)} className="px-4 py-2 bg-[var(--card)] border border-[var(--border)] rounded text-sm hover:bg-[var(--border)]">Go back</button>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col" data-testid="doc-editor-page">
      {/* Header */}
      <div className="flex items-center gap-3 py-2 px-3 border-b border-[var(--border)]">
        <button
          data-testid="doc-back"
          onClick={() => navigate(-1)}
          className="text-sm text-[var(--text-muted)] hover:text-white"
        >← Back</button>
        <span className="text-xs px-2 py-0.5 rounded bg-[var(--card)] border border-[var(--border)] text-[var(--text-muted)]" data-testid="doc-type-badge">
          {DOCUMENT_TYPE_LABELS[docType] || docType}
        </span>
        <DocumentSaveStatus status={saveState} lastSaved={lastSaved} />
        <button
          data-testid="toggle-assist"
          onClick={() => setShowAssist(!showAssist)}
          className={`px-2 py-0.5 text-xs rounded ${showAssist ? 'bg-[var(--primary)] text-white' : 'bg-[var(--card)] border border-[var(--border)]'}`}
        >🤖 Assist</button>
        {/* v1.4 Track 6: Export buttons */}
        {!previewMode && !archived && (
          <>
            <button data-testid="export-md" onClick={() => doExport('markdown')} disabled={exporting} className="px-2 py-0.5 text-xs bg-[var(--card)] border border-[var(--border)] rounded hover:bg-[var(--border)]">📄 MD</button>
            <button data-testid="export-pdf" onClick={() => doExport('pdf')} disabled={exporting} className="px-2 py-0.5 text-xs bg-[var(--card)] border border-[var(--border)] rounded hover:bg-[var(--border)]">📄 PDF</button>
            <button data-testid="export-docx" onClick={() => doExport('docx')} disabled={exporting} className="px-2 py-0.5 text-xs bg-[var(--card)] border border-[var(--border)] rounded hover:bg-[var(--border)]">📄 DOCX</button>
          </>
        )}
        {exporting && <span className="text-xs text-[var(--text-muted)]" data-testid="export-loading">Exporting...</span>}
        {exportError && <span className="text-xs text-red-400" data-testid="export-error">{exportError}</span>}
        {/* v1.4 Track 7: Version History */}
        <button
          data-testid="version-history-btn"
          onClick={() => setShowVersions(true)}
          className="px-2 py-0.5 text-xs bg-[var(--card)] border border-[var(--border)] rounded hover:bg-[var(--border)]"
        >📋 History</button>
      </div>

      {/* Toolbar */}
      <DocumentToolbar
        onAddBlock={addBlock}
        previewMode={previewMode}
        onTogglePreview={() => setPreviewMode(!previewMode)}
        onSave={handleSave}
        saveDisabled={!dirty || saveState === 'saving'}
      />

      {/* Editor body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Outline sidebar */}
        {!previewMode && (
          <div className="w-48 border-r border-[var(--border)] overflow-y-auto">
            <DocumentOutline blocks={blocks} />
          </div>
        )}

        {/* Document area */}
        <div className="flex-1 overflow-y-auto p-6 max-w-3xl mx-auto w-full">
          {/* Title */}
          {!previewMode ? (
            <input
              type="text"
              data-testid="doc-title-input"
              value={title}
              onChange={e => handleTitleChange(e.target.value)}
              placeholder="Document title..."
              className="w-full text-2xl font-bold bg-transparent border-none outline-none mb-4"
            />
          ) : (
            <h1 className="text-2xl font-bold mb-4" data-testid="doc-title-preview">{title}</h1>
          )}

          {/* Blocks */}
          {blocks.length === 0 ? (
            <div className="text-center py-12 text-[var(--text-muted)]" data-testid="doc-empty-state">
              <p className="text-sm">This document is empty.</p>
              <p className="text-xs mt-1">Use the toolbar above to add blocks.</p>
            </div>
          ) : (
            <div className="space-y-2">
              {blocks.map((blk, i) => (
                <DocumentBlockEditor
                  key={blk.id}
                  block={blk}
                  onChange={updated => updateBlock(i, updated)}
                  onDelete={() => deleteBlock(i)}
                  onMoveUp={() => moveBlock(i, 'up')}
                  onMoveDown={() => moveBlock(i, 'down')}
                  onDuplicate={() => duplicateBlock(i)}
                  isFirst={i === 0}
                  isLast={i === blocks.length - 1}
                  preview={previewMode}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Agent Assist Panel */}
      {!previewMode && showAssist && (
        <AgentAssistPanel
          artifactId={artifactId || ''}
          selectedBlockId={selectedBlockId}
          selectedBlockText={selectedBlockText}
          documentType={docType}
          onInsertBelow={handleInsertBelow}
          onReplaceBlock={handleReplaceBlock}
        />
      )}

      {/* v1.4 Track 7: Version History Drawer */}
      <VersionHistoryDrawer
        artifactId={artifactId || ''}
        open={showVersions}
        onClose={() => setShowVersions(false)}
        archived={archived}
        onRestored={(docJson, wc) => {
          if (docJson?.blocks) {
            setBlocks(docJson.blocks);
          }
          if (docJson?.title) {
            setTitle(docJson.title);
          }
          setDirty(false);
          setSaveState('saved');
        }}
      />
    </div>
  );
}
