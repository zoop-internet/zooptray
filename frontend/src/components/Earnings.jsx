import React from 'react';
import { Wallet, ExternalLink } from 'lucide-react';
import * as runtime from "../../wailsjs/runtime/runtime";

export default function Earnings() {
  const handleLearnMore = () => {
    runtime.BrowserOpenURL("https://zoopnetwork.com");
  };

  return (
    <div className="earnings-card">
      <div className="earnings-icon-container">
        <Wallet size={32} />
      </div>
      <h2>Earnings coming soon</h2>
      <p>
        We're working on bringing you detailed earnings and payment insights directly inside the desktop dashboard.
      </p>
      <button 
        type="button" 
        className="btn btn-primary"
        onClick={handleLearnMore}
      >
        <span>Learn more</span>
        <ExternalLink size={14} />
      </button>
    </div>
  );
}
