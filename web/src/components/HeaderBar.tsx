import React from 'react';
import { UserScore, Problem } from '@/domain/scoreboard/types';
import { SCOREBOARD_LABELS, SCOREBOARD_CONFIG } from '@/domain/scoreboard/constants';

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
  const leaderName =
    users.length > 0 ? users[0].displayName || users[0].userId : SCOREBOARD_CONFIG.DEFAULT_LEADER_NAME;

  return (
    <header className="header-bar">
      <div className="brand-section">
        <span className="brand-logo">{SCOREBOARD_LABELS.BRAND_LOGO}</span>
        <h1 className="header-title" id="title">
          {title || SCOREBOARD_CONFIG.DEFAULT_PAGE_TITLE}
        </h1>
      </div>

      {/* Dynamic Class Health Stats Bar */}
      <div className="class-stats-pill" id="class-stats">
        <span className="stat-item">
          👥 {users.length} {SCOREBOARD_LABELS.STATS.STUDENTS_SUFFIX}
        </span>
        <span className="stat-divider">•</span>
        <span className="stat-item">
          🧩 {problems.length} {SCOREBOARD_LABELS.STATS.PROBLEMS_SUFFIX}
        </span>
        <span className="stat-divider">•</span>
        <span className="stat-item">
          {SCOREBOARD_LABELS.MEDALS.GOLD} {SCOREBOARD_LABELS.STATS.LEADER_PREFIX}{' '}
          <span className="stat-highlight">{leaderName}</span>
        </span>
      </div>
    </header>
  );
};
