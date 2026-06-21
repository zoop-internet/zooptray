import React, { useState, useEffect, useCallback } from 'react';
import { CheckCircle2, AlertTriangle, Info, X } from 'lucide-react';

// Global toast emitter — call window.__toast(message, type) from anywhere
let _emit = null;
export const toast = {
  success: (msg) => _emit?.(msg, 'success'),
  warning: (msg) => _emit?.(msg, 'warning'),
  info:    (msg) => _emit?.(msg, 'info'),
  error:   (msg) => _emit?.(msg, 'error'),
};

export default function ToastContainer() {
  const [toasts, setToasts] = useState([]);

  // Register the emitter
  useEffect(() => {
    _emit = (message, type = 'info') => {
      const id = Date.now() + Math.random();
      setToasts(prev => [...prev, { id, message, type }]);
      setTimeout(() => {
        setToasts(prev => prev.filter(t => t.id !== id));
      }, 3500);
    };
    return () => { _emit = null; };
  }, []);

  const dismiss = useCallback((id) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  const icons = {
    success: <CheckCircle2 size={17} />,
    warning: <AlertTriangle size={17} />,
    info:    <Info size={17} />,
    error:   <X size={17} />,
  };

  return (
    <div className="toast-container">
      {toasts.map((t) => (
        <div key={t.id} className={`toast toast-${t.type}`}>
          <span className="toast-icon">{icons[t.type]}</span>
          <span className="toast-msg">{t.message}</span>
          <button className="toast-close" onClick={() => dismiss(t.id)}>
            <X size={13} />
          </button>
        </div>
      ))}
    </div>
  );
}
