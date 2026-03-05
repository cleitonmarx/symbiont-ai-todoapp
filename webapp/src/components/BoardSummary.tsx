import { useState } from 'react';
import { BoardSummary as BoardSummaryType } from '../services/api';
import '../styles/BoardSummary.css';

interface BoardSummaryProps {
  data: BoardSummaryType;
}

const formatUpdatedAgo = (generatedAtRaw: string): string => {
  const generatedAtMs = new Date(generatedAtRaw).getTime();
  if (Number.isNaN(generatedAtMs)) {
    return 'Updated';
  }

  const diffMs = Math.max(0, Date.now() - generatedAtMs);
  const minuteMs = 60 * 1000;
  const hourMs = 60 * minuteMs;
  const dayMs = 24 * hourMs;

  if (diffMs < minuteMs) {
    return 'Updated just now';
  }
  if (diffMs < hourMs) {
    const minutes = Math.max(1, Math.floor(diffMs / minuteMs));
    return `Updated ${minutes}m ago`;
  }
  if (diffMs < dayMs) {
    const hours = Math.max(1, Math.floor(diffMs / hourMs));
    return `Updated ${hours}h ago`;
  }

  const days = Math.max(1, Math.floor(diffMs / dayMs));
  return `Updated ${days}d ago`;
};

export const BoardSummary = ({ data }: BoardSummaryProps) => {
  const [showDetails, setShowDetails] = useState(false);
  const updatedLabel = formatUpdatedAgo(data.generated_at);

  const hasDetails = data.next_up.length > 0 || data.overdue.length > 0 || data.near_deadline.length > 0;

  return (
    <div className="board-summary">
      <div className="board-summary-header">
        <h2>Board Overview</h2>
        <span className="ai-badge">{updatedLabel}</span>
      </div>

      <p className="board-summary-text">{data.summary}</p>

      <div className="board-summary-stats">
        <div className="stat">
          <span className="stat-label">Open</span>
          <span className="stat-value">{data.counts.OPEN}</span>
        </div>
        <div className="stat">
          <span className="stat-label">Done</span>
          <span className="stat-value">{data.counts.DONE}</span>
        </div>
        {hasDetails && (
          <button 
            className="toggle-details-btn"
            onClick={() => setShowDetails(!showDetails)}
            aria-expanded={showDetails}
          >
            {showDetails ? 'Hide Details' : 'Show Details'}
          </button>
        )}
      </div>
      
  
      {showDetails && (
        <div className="board-summary-details">
          {data.overdue.length > 0 && (
            <div className="board-summary-section">
              <h3 className="overdue-title">Overdue</h3>
              <ul className="summary-list overdue-list">
                {data.overdue.map((title, idx) => (
                  <li key={idx}>{title}</li>
                ))}
              </ul>
            </div>
          )}

          {data.near_deadline.length > 0 && (
            <div className="board-summary-section">
              <h3 className="near-deadline-title">Near Deadline</h3>
              <ul className="summary-list">
                {data.near_deadline.map((title, idx) => (
                  <li key={idx}>{title}</li>
                ))}
              </ul>
            </div>
          )}

          {data.next_up.length > 0 && (
            <div className="board-summary-section">
              <h3>Next Up</h3>
              <ul className="summary-list">
                {data.next_up.map((item, idx) => (
                  <li key={idx}>
                    <span className="item-title">{item.title}</span>
                    <span className="item-reason">{item.reason}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
