import { UserScore, Problem, ActiveTab } from '@/domain/scoreboard/types';
import { SCOREBOARD_LABELS, SCOREBOARD_CONFIG } from '@/domain/scoreboard/constants';
import { NavigationTabs } from '@/components/NavigationTabs';

interface HeaderBarProps {
  title?: string;
  users: UserScore[];
  problems: Problem[];
  activeTab: ActiveTab;
  onTabChange: (tab: ActiveTab) => void;
}

export function HeaderBar({
  title,
  users,
  problems,
  activeTab,
  onTabChange,
}: HeaderBarProps) {
  const topUser = users.length > 0 ? users[0] : null;
  const leaderName =
    (topUser ? topUser.displayName || topUser.userId : null) ||
    SCOREBOARD_CONFIG.DEFAULT_LEADER_NAME;

  return (
    <header className="header-bar">
      <div className="brand-section">
        <span className="brand-logo">{SCOREBOARD_LABELS.BRAND_LOGO}</span>
        <h1 className="header-title" id="title">
          {title || SCOREBOARD_CONFIG.DEFAULT_PAGE_TITLE}
        </h1>
      </div>

      {/* Center Segmented View Switcher */}
      <NavigationTabs
        activeTab={activeTab}
        onTabChange={onTabChange}
      />

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
}
