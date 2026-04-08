import { Server } from 'lucide-react';
import type { ReactNode } from 'react';

/** Vault-style centered empty state with icon, title, description, optional CTA */
export default function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: ReactNode;
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-neutral-100 text-neutral-400">
        {icon ?? <Server size={24} />}
      </div>
      <h3 className="text-base font-semibold text-neutral-700">{title}</h3>
      <p className="mt-1 max-w-sm text-sm text-neutral-500">{description}</p>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
