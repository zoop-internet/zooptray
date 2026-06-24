import React, { useState, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
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
  const [menuStyle, setMenuStyle] = useState({ top: 0, left: 0, width: 0 });
  const triggerRef = useRef(null);
  const menuRef = useRef(null);

  const selected = options.find(o => o.value === value) || options[0] || { label: label || 'Select...', value: '' };

  useEffect(() => {
    const handleClickOutside = (e) => {
      if (triggerRef.current?.contains(e.target)) return;
      if (menuRef.current?.contains(e.target)) return;
      setOpen(false);
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    if (!open) return;

    const updatePosition = () => {
      const rect = triggerRef.current?.getBoundingClientRect();
      if (!rect) return;
      setMenuStyle({
        top: rect.bottom + 6,
        left: rect.left,
        width: rect.width,
      });
    };

    updatePosition();
    window.addEventListener('resize', updatePosition);
    window.addEventListener('scroll', updatePosition, true);
    return () => {
      window.removeEventListener('resize', updatePosition);
      window.removeEventListener('scroll', updatePosition, true);
    };
  }, [open]);

  const handleSelect = (optValue) => {
    onChange(optValue);
    setOpen(false);
  };

  const menu = open && createPortal(
    <div className="zoop-dropdown-menu" ref={menuRef} style={menuStyle}>
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
    </div>,
    document.body,
  );

  return (
    <div className="zoop-dropdown" ref={triggerRef}>
      <button
        type="button"
        className={`zoop-dropdown-trigger ${open ? 'open' : ''}`}
        onClick={() => setOpen(prev => !prev)}
      >
        <span>{selected?.label || label || 'Select...'}</span>
        <ChevronDown size={14} className={`zoop-dropdown-chevron ${open ? 'rotated' : ''}`} />
      </button>
      {menu}
    </div>
  );
}
