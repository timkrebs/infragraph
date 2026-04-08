import { nodeColor } from '../utils/colors';

/** Vault-style type badge with color dot */
export default function TypeBadge({ type }: { type: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded border border-neutral-200 bg-neutral-50 px-2 py-0.5 font-mono text-xs font-semibold text-neutral-600">
      <span
        className="inline-block h-2 w-2 rounded-full"
        style={{ background: nodeColor(type) }}
      />
      {type}
    </span>
  );
}
