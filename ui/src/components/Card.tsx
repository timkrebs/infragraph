/** Vault-style card with optional title. Uses subtle border, no shadow. */
export default function Card({
  title,
  children,
  className = '',
}: {
  title?: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={`overflow-hidden rounded-lg border border-neutral-200 bg-white ${className}`}>
      {title && (
        <div className="border-b border-neutral-200 px-4 py-3 text-xs font-semibold uppercase tracking-wider text-neutral-500">
          {title}
        </div>
      )}
      <div className="p-4">{children}</div>
    </div>
  );
}
