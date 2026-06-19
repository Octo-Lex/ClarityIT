export type BlockType = 'heading' | 'paragraph' | 'bullets' | 'numbered_list' | 'table' | 'quote' | 'callout' | 'page_break';

const BLOCK_OPTIONS: { type: BlockType; label: string; icon: string }[] = [
  { type: 'heading', label: 'Heading', icon: 'H' },
  { type: 'paragraph', label: 'Paragraph', icon: '¶' },
  { type: 'bullets', label: 'Bullet List', icon: '•' },
  { type: 'numbered_list', label: 'Numbered List', icon: '1.' },
  { type: 'table', label: 'Table', icon: '⊞' },
  { type: 'quote', label: 'Quote', icon: '"' },
  { type: 'callout', label: 'Callout', icon: '!' },
  { type: 'page_break', label: 'Page Break', icon: '—' },
];

interface Props {
  onAddBlock: (type: BlockType) => void;
  previewMode: boolean;
  onTogglePreview: () => void;
  onSave?: () => void;
  saveDisabled?: boolean;
}

export default function DocumentToolbar({ onAddBlock, previewMode, onTogglePreview, onSave, saveDisabled }: Props) {
  return (
    <div className="flex items-center gap-2 py-2 px-3 border-b border-border" data-testid="document-toolbar">
      {!previewMode && (
        <div className="flex items-center gap-1">
          <span className="text-xs text-muted-foreground mr-1">Add:</span>
          {BLOCK_OPTIONS.map(opt => (
            <button
              key={opt.type}
              data-testid={`toolbar-add-${opt.type}`}
              title={opt.label}
              onClick={() => onAddBlock(opt.type)}
              className="px-2 py-1 text-xs rounded bg-surface border border-border hover:bg-muted hover:text-white"
            >
              {opt.icon}
            </button>
          ))}
        </div>
      )}
      <div className="flex-1" />
      <button
        data-testid="toolbar-preview"
        onClick={onTogglePreview}
        className="px-3 py-1 text-xs rounded bg-surface border border-border hover:bg-muted"
      >
        {previewMode ? 'Edit' : 'Preview'}
      </button>
      {onSave && (
        <button
          data-testid="toolbar-save"
          onClick={onSave}
          disabled={saveDisabled}
          className="px-3 py-1 text-xs rounded bg-primary text-white hover:opacity-90 disabled:opacity-50"
        >
          Save
        </button>
      )}
    </div>
  );
}
