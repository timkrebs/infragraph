/** Vault-style status badge */
export default function StatusBadge({ status }: { status: string }) {
  const cls =
    status === 'healthy'
      ? 'bg-success-bg text-success border-success-border'
      : status === 'degraded'
        ? 'bg-warning-bg text-warning border-warning-border'
        : 'bg-neutral-100 text-neutral-500 border-neutral-200';

  return (
    <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${cls}`}>
      <span className={`inline-block h-1.5 w-1.5 rounded-full ${
        status === 'healthy' ? 'bg-success' : status === 'degraded' ? 'bg-warning' : 'bg-neutral-400'
      }`} />
      {status}
    </span>
  );
}
