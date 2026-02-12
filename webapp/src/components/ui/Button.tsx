import type { ButtonHTMLAttributes } from 'react';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
}

export const Button = ({ variant = 'primary', className = '', ...props }: ButtonProps) => {
  return <button {...props} className={`ui-btn ui-btn-${variant} ${className}`.trim()} />;
};
