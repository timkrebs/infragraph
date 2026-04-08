interface Props {
  type: string;
}

export default function TypeBadge({ type }: Props) {
  const cls = `type-${type}`;
  return <span className={`badge-type ${cls}`}>{type}</span>;
}
