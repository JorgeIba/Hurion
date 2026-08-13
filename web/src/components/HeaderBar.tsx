import React from 'react';
import { UserScore, Problem } from '@/domain/scoreboard/types';

interface HeaderBarProps {
  title?: string;
  users: UserScore[];
  problems: Problem[];
  searchQuery?: string;
  onSearchChange?: (query: string) => void;
}

export const HeaderBar: React.FC<HeaderBarProps> = ({
  title,
  users,
  problems,
}) => {
  const leaderName = users.length > 0 ? (users[0].displayName || users[0].userId) : '--';

  return (
    <header className="header-bar">
      <div className="brand-section">
        <span className="brand-logo">📊</span>
        <h1 className="header-title" id="title">
          {title || 'Scoreboard'}
        </h1>
      </div>

      {/* Dynamic Class Health Stats Bar */}
      <div className="class-stats-pill" id="class-stats">
        <span className="stat-item">👥 {users.length} Students</span>
        <span className="stat-divider">•</span>
        <span className="stat-item">🧩 {problems.length} Problems</span>
        <span className="stat-divider">•</span>
        <span className="stat-item">
          👑 Leader: <span className="stat-highlight">{leaderName}</span>
        </span>
      </div>

    </header>
  );
};
