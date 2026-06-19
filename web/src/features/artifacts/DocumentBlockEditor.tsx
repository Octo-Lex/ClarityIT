import { useState } from 'react';

export interface Block {
  id: string;
  type: string;
  level?: number;
  text?: string;
  items?: string[];
  headers?: string[];
  rows?: string[][];
  variant?: string;
}

interface Props {
  block: Block;
  onChange: (updated: Block) => void;
  onDelete: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onDuplicate: () => void;
  isFirst: boolean;
  isLast: boolean;
  preview?: boolean;
}

const CALLOUT_VARIANTS = ['info', 'warning', 'success', 'error', 'note', 'tip'];

function genId() {
  return 'blk_' + Date.now().toString(36) + '_' + Math.random().toString(36).slice(2, 8);
}

export default function DocumentBlockEditor({
  block, onChange, onDelete, onMoveUp, onMoveDown, onDuplicate, isFirst, isLast, preview,
}: Props) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [showConfirmDelete, setShowConfirmDelete] = useState(false);

  const update = (partial: Partial<Block>) => onChange({ ...block, ...partial });

  // ─── Render helpers ───
  const renderBlockContent = () => {
    switch (block.type) {
      case 'heading':
        if (preview) {
          const Tag = `h${block.level || 1}` as keyof React.JSX.IntrinsicElements;
          return <Tag className="font-bold mt-4" data-testid={`block-preview-${block.id}`}>{block.text}</Tag>;
        }
        return (
          <div className="flex items-center gap-2">
            <select
              data-testid={`block-level-${block.id}`}
              value={block.level || 1}
              onChange={e => update({ level: parseInt(e.target.value) })}
              className="text-xs bg-surface border border-border rounded px-1 py-1"
            >
              {[1, 2, 3, 4, 5, 6].map(l => <option key={l} value={l}>H{l}</option>)}
            </select>
            <input
              type="text"
              data-testid={`block-text-${block.id}`}
              value={block.text || ''}
              onChange={e => update({ text: e.target.value })}
              placeholder="Heading text..."
              className="flex-1 bg-surface border border-border rounded px-2 py-1 text-sm"
            />
          </div>
        );

      case 'paragraph':
        if (preview)
          return <p className="text-sm my-2" data-testid={`block-preview-${block.id}`}>{block.text}</p>;
        return (
          <textarea
            data-testid={`block-text-${block.id}`}
            value={block.text || ''}
            onChange={e => update({ text: e.target.value })}
            placeholder="Write something..."
            rows={3}
            className="w-full bg-surface border border-border rounded px-2 py-1 text-sm resize-y"
          />
        );

      case 'bullets':
      case 'numbered_list': {
        const ordered = block.type === 'numbered_list';
        if (preview) {
          const Tag = ordered ? 'ol' : 'ul';
          return (
            <Tag className={`text-sm my-2 ${ordered ? 'list-decimal' : 'list-disc'} pl-5`} data-testid={`block-preview-${block.id}`}>
              {(block.items || []).map((item, i) => <li key={i}>{item}</li>)}
            </Tag>
          );
        }
        const items = block.items || [];
        return (
          <div data-testid={`block-items-${block.id}`} className="space-y-1">
            {items.map((item, i) => (
              <div key={i} className="flex items-center gap-1">
                <span className="text-xs text-muted-foreground">{ordered ? `${i + 1}.` : '•'}</span>
                <input
                  type="text"
                  data-testid={`block-item-${block.id}-${i}`}
                  value={item}
                  onChange={e => {
                    const newItems = [...items];
                    newItems[i] = e.target.value;
                    update({ items: newItems });
                  }}
                  className="flex-1 bg-surface border border-border rounded px-2 py-1 text-sm"
                />
                <button
                  data-testid={`block-item-remove-${block.id}-${i}`}
                  onClick={() => update({ items: items.filter((_, idx) => idx !== i) })}
                  className="text-xs text-destructive hover:text-destructive px-1"
                >✕</button>
              </div>
            ))}
            <button
              data-testid={`block-item-add-${block.id}`}
              onClick={() => update({ items: [...items, 'New item'] })}
              className="text-xs text-primary hover:opacity-80"
            >+ Add item</button>
          </div>
        );
      }

      case 'table': {
        const headers = block.headers || [];
        const rows = block.rows || [];
        if (preview) {
          return (
            <div className="my-2" data-testid={`block-preview-${block.id}`}>
              <table className="w-full text-sm border-collapse">
                <thead><tr>{headers.map((h, i) => <th key={i} className="border border-border px-2 py-1 text-left">{h}</th>)}</tr></thead>
                <tbody>{rows.map((row, ri) => <tr key={ri}>{row.map((cell, ci) => <td key={ci} className="border border-border px-2 py-1">{cell}</td>)}</tr>)}</tbody>
              </table>
            </div>
          );
        }
        return (
          <div data-testid={`block-table-${block.id}`} className="my-2 overflow-x-auto">
            <table className="w-full text-sm border-collapse">
              <thead>
                <tr>{headers.map((h, i) => (
                  <th key={i} className="border border-border p-0">
                    <input
                      data-testid={`block-header-${block.id}-${i}`}
                      value={h}
                      onChange={e => { const nh = [...headers]; nh[i] = e.target.value; update({ headers: nh }); }}
                      className="w-full bg-surface px-2 py-1 text-left text-xs"
                    />
                  </th>
                ))}</tr>
              </thead>
              <tbody>{rows.map((row, ri) => (
                <tr key={ri}>
                  {row.map((cell, ci) => (
                    <td key={ci} className="border border-border p-0">
                      <input
                        data-testid={`block-cell-${block.id}-${ri}-${ci}`}
                        value={cell}
                        onChange={e => {
                          const nr = [...rows]; nr[ri] = [...row]; nr[ri][ci] = e.target.value;
                          update({ rows: nr });
                        }}
                        className="w-full bg-surface px-2 py-1 text-xs"
                      />
                    </td>
                  ))}
                  <td className="px-1">
                    <button
                      data-testid={`block-row-remove-${block.id}-${ri}`}
                      onClick={() => update({ rows: rows.filter((_, idx) => idx !== ri) })}
                      className="text-xs text-destructive"
                    >✕</button>
                  </td>
                </tr>
              ))}</tbody>
            </table>
            <div className="flex gap-2 mt-1">
              <button
                data-testid={`block-row-add-${block.id}`}
                onClick={() => update({ rows: [...rows, headers.map(() => '')] })}
                className="text-xs text-primary"
              >+ Row</button>
              <button
                data-testid={`block-col-add-${block.id}`}
                onClick={() => update({ headers: [...headers, 'New'], rows: rows.map(r => [...r, '']) })}
                className="text-xs text-primary"
              >+ Column</button>
              {headers.length > 1 && (
                <button
                  data-testid={`block-col-remove-${block.id}`}
                  onClick={() => update({ headers: headers.slice(0, -1), rows: rows.map(r => r.slice(0, -1)) })}
                  className="text-xs text-destructive"
                >− Column</button>
              )}
            </div>
          </div>
        );
      }

      case 'quote':
        if (preview)
          return <blockquote className="border-l-4 border-border pl-4 italic text-sm my-2" data-testid={`block-preview-${block.id}`}>{block.text}</blockquote>;
        return (
          <textarea
            data-testid={`block-text-${block.id}`}
            value={block.text || ''}
            onChange={e => update({ text: e.target.value })}
            placeholder="Quote text..."
            rows={2}
            className="w-full bg-surface border border-border rounded px-2 py-1 text-sm italic resize-y"
          />
        );

      case 'callout':
        if (preview) {
          const colors: Record<string, string> = {
            info: 'border-info/50 bg-info/10', warning: 'border-warning/50 bg-warning/10',
            success: 'border-success/50 bg-success/10', error: 'border-destructive/50 bg-destructive/10',
            note: 'border-border bg-muted/50', tip: 'border-primary bg-primary/10',
          };
          return (
            <div className={`border-l-4 ${colors[block.variant || 'info']} px-3 py-2 rounded text-sm my-2`} data-testid={`block-preview-${block.id}`}>
              <span className="text-xs uppercase font-bold mr-2">{block.variant}</span>
              {block.text}
            </div>
          );
        }
        return (
          <div className="space-y-1">
            <select
              data-testid={`block-variant-${block.id}`}
              value={block.variant || 'info'}
              onChange={e => update({ variant: e.target.value })}
              className="text-xs bg-surface border border-border rounded px-1 py-1"
            >
              {CALLOUT_VARIANTS.map(v => <option key={v} value={v}>{v}</option>)}
            </select>
            <textarea
              data-testid={`block-text-${block.id}`}
              value={block.text || ''}
              onChange={e => update({ text: e.target.value })}
              placeholder="Callout text..."
              rows={2}
              className="w-full bg-surface border border-border rounded px-2 py-1 text-sm resize-y"
            />
          </div>
        );

      case 'page_break':
        return (
          <div data-testid={`block-preview-${block.id}`} className="border-t-2 border-dashed border-border my-4 py-1 text-center">
            <span className="text-xs text-muted-foreground">— Page Break —</span>
          </div>
        );

      default:
        return <div className="text-xs text-destructive">Unknown block type: {block.type}</div>;
    }
  };

  if (preview) {
    return <div>{renderBlockContent()}</div>;
  }

  return (
    <div
      data-testid={`block-editor-${block.id}`}
      className="group relative border border-transparent hover:border-border rounded p-2 transition-colors"
    >
      {/* Block actions */}
      <div className="absolute -left-1 top-1 opacity-0 group-hover:opacity-100 flex flex-col gap-0.5 transition-opacity">
        <button
          data-testid={`block-up-${block.id}`}
          onClick={onMoveUp}
          disabled={isFirst}
          className="text-xs text-muted-foreground hover:text-white disabled:opacity-30"
          title="Move up"
        >↑</button>
        <button
          data-testid={`block-down-${block.id}`}
          onClick={onMoveDown}
          disabled={isLast}
          className="text-xs text-muted-foreground hover:text-white disabled:opacity-30"
          title="Move down"
        >↓</button>
      </div>
      <div className="absolute right-0 top-1 opacity-0 group-hover:opacity-100 flex gap-1 transition-opacity">
        <button
          data-testid={`block-duplicate-${block.id}`}
          onClick={onDuplicate}
          className="text-xs text-muted-foreground hover:text-white"
          title="Duplicate"
        >⎘</button>
        <button
          data-testid={`block-delete-${block.id}`}
          onClick={() => setShowConfirmDelete(true)}
          className="text-xs text-destructive hover:text-destructive"
          title="Delete"
        >🗑</button>
      </div>

      {renderBlockContent()}

      {/* Delete confirmation */}
      {showConfirmDelete && (
        <div className="mt-2 flex items-center gap-2 p-2 bg-surface rounded border border-border" data-testid={`block-confirm-delete-${block.id}`}>
          <span className="text-xs">Delete this block?</span>
          <button
            data-testid={`block-confirm-yes-${block.id}`}
            onClick={onDelete}
            className="text-xs px-2 py-0.5 bg-destructive text-white rounded hover:bg-destructive"
          >Yes, delete</button>
          <button
            data-testid={`block-confirm-no-${block.id}`}
            onClick={() => setShowConfirmDelete(false)}
            className="text-xs px-2 py-0.5 bg-muted rounded"
          >Cancel</button>
        </div>
      )}
    </div>
  );
}

export { genId };
