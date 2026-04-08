interface Props {
  status: string;
}

export default function StatusBadge({ status }: Props) {
  const cls =
    status === 'healthy'
      ? 'badge-healthy'
      : status === 'degraded'
        ? 'badge-degraded'
        : 'badge-unknown';
  return <span className={`badge ${cls}`}>{status}</span>;
}
