import type { ReactNode } from "react";

type Props = {
  title: string;
  children: ReactNode;
  onClose?: () => void;
  footer?: ReactNode;
};

export function Dialog({ title, children, onClose, footer }: Props) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="presentation"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="odm-dialog-title"
        className="w-full max-w-md rounded-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <h2
          id="odm-dialog-title"
          className="text-lg font-semibold text-[var(--color-text)]"
        >
          {title}
        </h2>
        <div className="mt-3 text-sm text-[var(--color-text)]">{children}</div>
        {footer ? <div className="mt-5 flex justify-end gap-2">{footer}</div> : null}
      </div>
    </div>
  );
}
