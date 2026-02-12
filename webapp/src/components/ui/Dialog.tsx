import type { ReactNode } from 'react';

interface DialogProps {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  className?: string;
}

export const Dialog = ({ open, title, onClose, children, footer, className }: DialogProps) => {
  if (!open) {
    return null;
  }

  return (
    <div className="ui-dialog-overlay" onClick={onClose} role="presentation">
      <section
        className={`ui-dialog ${className || ''}`.trim()}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onClick={(event) => event.stopPropagation()}
      >
        <header className="ui-dialog-header">
          <h2>{title}</h2>
          <button className="ui-icon-btn" type="button" onClick={onClose} aria-label="Close dialog">
            ×
          </button>
        </header>
        <div className="ui-dialog-content">{children}</div>
        {footer ? <footer className="ui-dialog-footer">{footer}</footer> : null}
      </section>
    </div>
  );
};
