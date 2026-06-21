import React, { useState, useRef, useEffect } from 'react';
import { ChevronDown, Check } from 'lucide-react';

/**
 * Custom Dropdown component — replaces native <select> with an animated popup.
 * Props:
 *   value     - currently selected value (string)
 *   onChange  - callback(value) when an option is selected
 *   options   - array of { label: string, value: string }
 *   label     - optional placeholder label
 */
export default function Dropdown({ value, onChange, options, label }) {
  const [open, setOpen] = useState(false);
  const ref = useRef(null);

  const selected = options.find(o => o.value === value) || options[0];

  // Close on outside click
  useEffect(() => {
    const handler = (e) => {
      if (ref.current && !ref.current.contains(e.target)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const handleSelect = (optValue) => {
    onChange(optValue);
    setOpen(false);
  };

  return (
    <div className="zoop-dropdown" ref={ref}>
      <button
        type="button"
        className={`zoop-dropdown-trigger ${open ? 'open' : ''}`}
        onClick={() => setOpen(prev => !prev)}
      >
        <span>{selected?.label || label || 'Select...'}</span>
        <ChevronDown size={14} className={`zoop-dropdown-chevron ${open ? 'rotated' : ''}`} />
      </button>

      {open && (
        <div className="zoop-dropdown-menu">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              className={`zoop-dropdown-item ${opt.value === value ? 'selected' : ''}`}
              onClick={() => handleSelect(opt.value)}
            >
              <span>{opt.label}</span>
              {opt.value === value && <Check size={13} />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
