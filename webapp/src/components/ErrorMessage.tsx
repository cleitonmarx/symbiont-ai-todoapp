import React from 'react';

interface ErrorMessageProps {
  error?: string | null;
}

const ErrorMessage: React.FC<ErrorMessageProps> = ({ error }) => {
  if (!error) {
    return null;
  }

  return (
    <div className="error">
      {error}
    </div>
  );
};

export default ErrorMessage;