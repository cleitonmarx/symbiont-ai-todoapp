import { useEffect, useState } from 'react';
import type { TodoStatus } from '../../types';
import type { TodoSearchType, TodoSort } from '../../services/todosApi';
import { DateRangePicker } from '../../components/ui/DateRangePicker';

type TodoControlsStatus = TodoStatus | 'ALL';

interface TodoControlsBarProps<TStatus extends string = TodoControlsStatus> {
  statusFilter: TStatus;
  onStatusFilterChange: (status: TStatus) => void;
  statusOptions?: readonly TStatus[];
  disabled?: boolean;
  idPrefix?: string;
  sortBy: TodoSort;
  onSortByChange: (sort: TodoSort) => void;
  pageSize: number;
  onPageSizeChange: (size: number) => void;
  searchQuery: string;
  onSearchQueryChange: (query: string) => void;
  searchType: TodoSearchType;
  onSearchTypeChange: (searchType: TodoSearchType) => void;
  dueAfter: string;
  onDueAfterChange: (date: string) => void;
  dueBefore: string;
  onDueBeforeChange: (date: string) => void;
  onClearDateRange: () => void;
}

export const TodoControlsBar = <TStatus extends string = TodoControlsStatus>({
  statusFilter,
  onStatusFilterChange,
  statusOptions,
  disabled = false,
  idPrefix = 'todo',
  sortBy,
  onSortByChange,
  pageSize,
  onPageSizeChange,
  searchQuery,
  onSearchQueryChange,
  searchType,
  onSearchTypeChange,
  dueAfter,
  onDueAfterChange,
  dueBefore,
  onDueBeforeChange,
  onClearDateRange,
}: TodoControlsBarProps<TStatus>) => {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const hasDateRange = Boolean(dueAfter || dueBefore);
  const availableStatusOptions = (statusOptions ?? (['ALL', 'OPEN', 'DONE'] as const)) as readonly TStatus[];
  const sortSelectId = `${idPrefix}-sort-select`;
  const pageSizeSelectId = `${idPrefix}-page-size-select`;
  const searchInputId = `${idPrefix}-search-input`;
  const searchTypeSelectId = `${idPrefix}-search-type-select`;
  const advancedFiltersId = `${idPrefix}-advanced-filters`;
  const dateRangeId = `${idPrefix}-date-range-picker`;
  const handleDateRangeChange = (startDate: string, endDate: string) => {
    onDueAfterChange(startDate);
    onDueBeforeChange(endDate);
  };

  useEffect(() => {
    if (hasDateRange) {
      setShowAdvanced(true);
    }
  }, [hasDateRange]);

  return (
    <section className="ui-controls" aria-label="Todo filters and search">
      <div className="ui-controls-group">
        <span className="ui-controls-label">Status</span>
        <div className="ui-segmented">
          {availableStatusOptions.map((status) => (
            <button
              key={String(status)}
              type="button"
              className={`ui-segmented-item ${statusFilter === status ? 'active' : ''}`}
              onClick={() => onStatusFilterChange(status)}
              disabled={disabled}
            >
              {status}
            </button>
          ))}
        </div>
      </div>

      <div className="ui-controls-group ui-controls-sort">
        <label className="ui-controls-label" htmlFor={sortSelectId}>
          Sort
        </label>
        <select
          id={sortSelectId}
          className="ui-select"
          value={sortBy}
          disabled={disabled}
          onChange={(event) => onSortByChange(event.target.value as TodoSort)}
        >
          <option value="createdAtAsc">Created At</option>
          <option value="createdAtDesc">Created At Desc</option>
          <option value="dueDateAsc">Due Date Asc</option>
          <option value="dueDateDesc">Due Date Desc</option>
          {searchQuery && searchType === 'SIMILARITY' ? (
            <>
              <option value="similarityAsc">Similarity Asc</option>
              <option value="similarityDesc">Similarity Desc</option>
            </>
          ) : null}
        </select>
      </div>

      <div className="ui-controls-group ui-controls-size">
        <label className="ui-controls-label" htmlFor={pageSizeSelectId}>
          Page Size
        </label>
        <select
          id={pageSizeSelectId}
          className="ui-select"
          value={pageSize}
          disabled={disabled}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
        >
          <option value={25}>25</option>
          <option value={50}>50</option>
          <option value={100}>100</option>
        </select>
      </div>

      <div className="ui-controls-group ui-controls-search">
        <label className="ui-controls-label" htmlFor={searchInputId}>
          Search
        </label>
        <input
          id={searchInputId}
          className="ui-input"
          type="text"
          value={searchQuery}
          placeholder="Search todos"
          disabled={disabled}
          onChange={(event) => onSearchQueryChange(event.target.value)}
        />
      </div>

      <div className="ui-controls-group ui-controls-search-type">
        <label className="ui-controls-label" htmlFor={searchTypeSelectId}>
          Search Type
        </label>
        <select
          id={searchTypeSelectId}
          className="ui-select"
          value={searchType}
          disabled={disabled}
          onChange={(event) => onSearchTypeChange(event.target.value as TodoSearchType)}
        >
          <option value="TITLE">Title</option>
          <option value="SIMILARITY">Similarity</option>
        </select>
      </div>

      <div className="ui-controls-group ui-controls-advanced-toggle-group">
        <span className="ui-controls-label">Filters</span>
        <button
          type="button"
          className={`ui-btn ui-btn-secondary ui-controls-advanced-toggle ${showAdvanced ? 'active' : ''}`}
          onClick={() => setShowAdvanced((prev) => !prev)}
          aria-expanded={showAdvanced}
          aria-controls={advancedFiltersId}
          disabled={disabled}
        >
          {showAdvanced ? 'Hide dates' : 'Dates'}
        </button>
      </div>

      {showAdvanced ? (
        <div id={advancedFiltersId} className="ui-controls-advanced">
          <div className="ui-controls-group ui-controls-date-range">
            <label className="ui-controls-label" htmlFor={dateRangeId}>
              Due Range
            </label>
            <DateRangePicker
              id={dateRangeId}
              startDate={dueAfter}
              endDate={dueBefore}
              onChange={handleDateRangeChange}
              placeholder="Select date range"
              disabled={disabled}
              showClearButton={false}
            />
          </div>

          <div className="ui-controls-group ui-controls-advanced-actions">
            <button
              type="button"
              className="ui-btn ui-btn-secondary"
              onClick={onClearDateRange}
              disabled={disabled || !hasDateRange}
            >
              Clear range
            </button>
          </div>
        </div>
      ) : null}
    </section>
  );
};
