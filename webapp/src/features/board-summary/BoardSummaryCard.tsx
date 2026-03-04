import { useState } from 'react';
import type { BoardSummary } from '../../services/boardApi';

interface BoardSummaryCardProps {
  data: BoardSummary;
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

export const BoardSummaryCard = ({ data }: BoardSummaryCardProps) => {
  const [expanded, setExpanded] = useState(false);

  const hasDetails = data.next_up.length > 0 || data.overdue.length > 0 || data.near_deadline.length > 0;
  const updatedLabel = formatUpdatedAgo(data.generated_at);

  return (
    <section className="ui-summary-card">
      <header className="ui-summary-header">
        <div>
          <h2>Board Overview</h2>
          <p>Latest task highlights</p>
        </div>
        <span className="ui-summary-badge">{updatedLabel}</span>
      </header>

      <p className="ui-summary-text">{data.summary}</p>

      <div className="ui-summary-stats">
        <article>
          <span>Open</span>
          <strong>{data.counts.OPEN}</strong>
        </article>
        <article>
          <span>Done</span>
          <strong>{data.counts.DONE}</strong>
        </article>
        {hasDetails ? (
          <button
            type="button"
            className="ui-btn ui-btn-ghost"
            onClick={() => setExpanded((value) => !value)}
            aria-expanded={expanded}
          >
            {expanded ? 'Hide details' : 'Show details'}
          </button>
        ) : null}
      </div>

      {expanded ? (
        <div className="ui-summary-details">
          {data.overdue.length > 0 ? (
            <section className="ui-summary-section ui-summary-overdue">
              <h3>Overdue</h3>
              <ul>
                {data.overdue.map((item, index) => (
                  <li key={`overdue-${index}`}>{item}</li>
                ))}
              </ul>
            </section>
          ) : null}

          {data.near_deadline.length > 0 ? (
            <section className="ui-summary-section ui-summary-near">
              <h3>Near Deadline</h3>
              <ul>
                {data.near_deadline.map((item, index) => (
                  <li key={`near-${index}`}>{item}</li>
                ))}
              </ul>
            </section>
          ) : null}

          {data.next_up.length > 0 ? (
            <section className="ui-summary-section ui-summary-next">
              <h3>Next Up</h3>
              <ul>
                {data.next_up.map((item, index) => (
                  <li key={`next-${index}`}>
                    <span>{item.title}</span>
                    <em>{item.reason}</em>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
        </div>
      ) : null}
    </section>
  );
};
