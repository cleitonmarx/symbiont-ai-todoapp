import { useEffect, useMemo, useRef, useState } from 'react';
import { DayPicker, type DateRange } from 'react-day-picker';
import 'react-day-picker/style.css';

interface DateRangePickerProps {
  id: string;
  startDate: string;
  endDate: string;
  onChange: (startDate: string, endDate: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  align?: 'left' | 'right';
  showClearButton?: boolean;
}

const DEFAULT_PLACEHOLDER = 'Select date range';
const PICKER_START_MONTH = new Date(2000, 0, 1);
const PICKER_END_MONTH = new Date(2100, 11, 31);

const parseDate = (value: string): Date | undefined => {
  if (!value) {
    return undefined;
  }
  const [year, month, day] = value.split('-').map(Number);
  if (!year || !month || !day) {
    return undefined;
  }
  return new Date(year, month - 1, day);
};

const toIsoDate = (value?: Date): string => {
  if (!value) {
    return '';
  }
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, '0');
  const day = String(value.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const toDisplayDate = (value?: Date): string => {
  if (!value) {
    return '';
  }
  return value.toLocaleDateString('en-US', {
    month: '2-digit',
    day: '2-digit',
    year: 'numeric',
  });
};

const getStartOfMonth = (value: Date): Date => new Date(value.getFullYear(), value.getMonth(), 1);
const addMonths = (value: Date, months: number): Date => new Date(value.getFullYear(), value.getMonth() + months, 1);
const isSameDay = (left?: Date, right?: Date): boolean => {
  if (!left || !right) {
    return false;
  }
  return (
    left.getFullYear() === right.getFullYear()
    && left.getMonth() === right.getMonth()
    && left.getDate() === right.getDate()
  );
};

export const DateRangePicker = ({
  id,
  startDate,
  endDate,
  onChange,
  placeholder = DEFAULT_PLACEHOLDER,
  disabled = false,
  className = '',
  align = 'left',
  showClearButton = true,
}: DateRangePickerProps) => {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement | null>(null);
  const hasValue = Boolean(startDate || endDate);

  const selectedRange: DateRange | undefined = useMemo(() => {
    const from = parseDate(startDate);
    const to = parseDate(endDate);
    if (!from && !to) {
      return undefined;
    }
    return { from, to };
  }, [startDate, endDate]);

  const [leftMonth, setLeftMonth] = useState<Date>(() => {
    const baseMonth = getStartOfMonth(parseDate(startDate) ?? new Date());
    return baseMonth;
  });
  const [rightMonth, setRightMonth] = useState<Date>(() => addMonths(leftMonth, 1));

  useEffect(() => {
    if (!open) {
      return;
    }

    const handleMouseDown = (event: MouseEvent) => {
      if (!rootRef.current) {
        return;
      }
      if (!rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };

    document.addEventListener('mousedown', handleMouseDown);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handleMouseDown);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const startValue = parseDate(startDate);
    const endValue = parseDate(endDate);
    const startMonth = getStartOfMonth(startValue ?? endValue ?? new Date());
    const endMonth = endValue ? getStartOfMonth(endValue) : addMonths(startMonth, 1);
    setLeftMonth(startMonth);
    setRightMonth(endMonth);
  }, [open]);

  const textValue = useMemo(() => {
    const from = parseDate(startDate);
    const to = parseDate(endDate);
    if (from && to) {
      return `${toDisplayDate(from)} - ${toDisplayDate(to)}`;
    }
    if (from) {
      return `${toDisplayDate(from)} - ...`;
    }
    if (to) {
      return `... - ${toDisplayDate(to)}`;
    }
    return placeholder;
  }, [startDate, endDate, placeholder]);

  const handleDateRangeSelect = (range: DateRange | undefined) => {
    if (!range?.from) {
      onChange('', '');
      return;
    }

    // When selecting a new range, treat the first click as start-only and keep the popover open.
    if (!range.to || isSameDay(range.from, range.to)) {
      const start = toIsoDate(range.from);
      onChange(start, '');
      const baseMonth = getStartOfMonth(range.from);
      setLeftMonth(baseMonth);
      setRightMonth(addMonths(baseMonth, 1));
      return;
    }

    const start = range.from <= range.to ? range.from : range.to;
    const end = range.from <= range.to ? range.to : range.from;
    onChange(toIsoDate(start), toIsoDate(end));
    setOpen(false);
  };

  const handleClear = () => {
    onChange('', '');
    setOpen(false);
  };

  const handleLeftMonthChange = (month: Date) => {
    setLeftMonth(getStartOfMonth(month));
  };

  const handleRightMonthChange = (month: Date) => {
    setRightMonth(getStartOfMonth(month));
  };

  return (
    <div ref={rootRef} className={`ui-date-range ${align === 'right' ? 'right' : 'left'} ${className}`.trim()}>
      <button
        id={id}
        type="button"
        className={`ui-date-range-trigger ${hasValue ? '' : 'placeholder'}`.trim()}
        onClick={() => setOpen((current) => !current)}
        disabled={disabled}
        aria-haspopup="dialog"
        aria-expanded={open}
      >
        <span className="ui-date-range-value">{textValue}</span>
        <span className="ui-date-range-caret" aria-hidden="true">
          📅
        </span>
      </button>

      {open ? (
        <div className="ui-date-range-popover" role="dialog" aria-label="Date range picker">
          <div className="ui-date-range-calendars">
            <div className="ui-date-range-calendar-pane">
              <DayPicker
                mode="range"
                month={leftMonth}
                onMonthChange={handleLeftMonthChange}
                selected={selectedRange}
                onSelect={handleDateRangeSelect}
                showOutsideDays
                navLayout="around"
                captionLayout="dropdown"
                startMonth={PICKER_START_MONTH}
                endMonth={PICKER_END_MONTH}
                className="ui-date-range-calendar"
              />
            </div>
            <div className="ui-date-range-calendar-pane">
              <DayPicker
                mode="range"
                month={rightMonth}
                onMonthChange={handleRightMonthChange}
                selected={selectedRange}
                onSelect={handleDateRangeSelect}
                showOutsideDays
                navLayout="around"
                captionLayout="dropdown"
                startMonth={PICKER_START_MONTH}
                endMonth={PICKER_END_MONTH}
                className="ui-date-range-calendar"
              />
            </div>
          </div>

          {showClearButton ? (
            <div className="ui-date-range-actions">
              <button type="button" className="ui-btn ui-btn-secondary" onClick={handleClear} disabled={!hasValue}>
                Clear
              </button>
              <button type="button" className="ui-btn ui-btn-secondary" onClick={() => setOpen(false)}>
                Close
              </button>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
};
