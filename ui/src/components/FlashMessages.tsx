import { useCallback, useRef, useState } from 'react';
import { X, CheckCircle, AlertTriangle, AlertCircle, Info } from 'lucide-react';

export interface FlashMessage {
  id: string;
  type: 'success' | 'warning' | 'danger' | 'info';
  title: string;
  message?: string;
}

const iconMap = {
  success: CheckCircle,
  warning: AlertTriangle,
  danger: AlertCircle,
  info: Info,
};

const colorMap = {
  success: 'border-success-border bg-success-bg text-success',
  warning: 'border-warning-border bg-warning-bg text-warning',
  danger: 'border-danger-border bg-danger-bg text-danger',
  info: 'border-brand-light bg-brand-light text-brand',
};

interface FlashMessagesProps {
  messages: FlashMessage[];
  onDismiss: (id: string) => void;
}

export default function FlashMessages({ messages, onDismiss }: FlashMessagesProps) {
  if (messages.length === 0) return null;

  return (
    <div className="flex flex-col gap-2 px-6 pt-3">
      {messages.map((msg) => {
        const Icon = iconMap[msg.type];
        return (
          <div
            key={msg.id}
            className={`flex items-start gap-3 rounded-md border px-4 py-3 text-sm ${colorMap[msg.type]}`}
          >
            <Icon size={16} className="mt-0.5 shrink-0" />
            <div className="flex-1">
              <p className="font-semibold">{msg.title}</p>
              {msg.message && <p className="mt-0.5 opacity-80">{msg.message}</p>}
            </div>
            <button onClick={() => onDismiss(msg.id)} className="shrink-0 opacity-60 hover:opacity-100">
              <X size={14} />
            </button>
          </div>
        );
      })}
    </div>
  );
}

/** Simple flash message hook */
export function useFlash() {
  const [messages, setMessages] = useState<FlashMessage[]>([]);
  const counter = useRef(0);

  const push = useCallback((type: FlashMessage['type'], title: string, message?: string) => {
    const id = String(++counter.current);
    setMessages((prev) => [...prev, { id, type, title, message }]);
    setTimeout(() => setMessages((prev) => prev.filter((m) => m.id !== id)), 6000);
  }, []);

  const dismiss = useCallback((id: string) => {
    setMessages((prev) => prev.filter((m) => m.id !== id));
  }, []);

  return { messages, push, dismiss };
}
