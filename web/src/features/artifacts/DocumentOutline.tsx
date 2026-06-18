interface BlockItem {
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
  blocks: BlockItem[];
  activeBlockId?: string;
  onSelect?: (id: string) => void;
}

export default function DocumentOutline({ blocks, activeBlockId, onSelect }: Props) {
  const headings = blocks.filter(b => b.type === 'heading');

  if (headings.length === 0) {
    return (
      <div className="text-xs text-[var(--text-muted)] p-3" data-testid="document-outline">
        No headings yet
      </div>
    );
  }

  return (
    <div className="p-3" data-testid="document-outline">
      <h3 className="text-xs font-semibold text-[var(--text-muted)] uppercase mb-2">Outline</h3>
      <ul className="space-y-1">
        {headings.map(h => (
          <li
            key={h.id}
            data-testid={`outline-item-${h.id}`}
            className={`text-xs cursor-pointer hover:text-white truncate ${
              activeBlockId === h.id ? 'text-[var(--primary)] font-medium' : 'text-[var(--text-muted)]'
            }`}
            style={{ paddingLeft: `${(h.level || 1) - 1}rem` }}
            onClick={() => onSelect?.(h.id)}
          >
            {h.text || '(empty heading)'}
          </li>
        ))}
      </ul>
    </div>
  );
}
