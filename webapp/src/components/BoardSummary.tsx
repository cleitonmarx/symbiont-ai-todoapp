import { useState } from 'react';
import { BoardSummary as BoardSummaryType } from '../services/api';
import '../styles/BoardSummary.css';

interface BoardSummaryProps {
  data: BoardSummaryType;
}

export const BoardSummary = ({ data }: BoardSummaryProps) => {
  const [showDetails, setShowDetails] = useState(false);

  const hasDetails = data.next_up.length > 0 || data.overdue.length > 0 || data.near_deadline.length > 0;

  return (
    <div className="board-summary">
      <div className="board-summary-header">
        <h2>Board Summary</h2>
        <span className="ai-badge">AI Generated</span>
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
        </div>
      )}
    </div>
  );
};