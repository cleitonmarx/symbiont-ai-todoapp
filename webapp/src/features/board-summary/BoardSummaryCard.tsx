import { useState } from 'react';
import type { BoardSummary } from '../../services/boardApi';

interface BoardSummaryCardProps {
  data: BoardSummary;
}

export const BoardSummaryCard = ({ data }: BoardSummaryCardProps) => {
  const [expanded, setExpanded] = useState(false);

  const hasDetails = data.next_up.length > 0 || data.overdue.length > 0 || data.near_deadline.length > 0;

  return (
    <section className="ui-summary-card">
      <header className="ui-summary-header">
        <div>
          <h2>Board Summary</h2>
          <p>AI generated board insights</p>
        </div>
        <span className="ui-summary-badge">AI</span>
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
