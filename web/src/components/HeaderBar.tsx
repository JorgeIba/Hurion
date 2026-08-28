import { UserScore, Problem, ActiveTab, CLASS_LABELS, CLASS_CONFIG } from '@/domain/class';
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
    CLASS_CONFIG.DEFAULT_LEADER_NAME;

  return (
    <header className="header-bar">
      <div className="brand-section">
        <span className="brand-logo">{CLASS_LABELS.BRAND_LOGO}</span>
        <h1 className="header-title" id="title">
          {title || CLASS_CONFIG.DEFAULT_PAGE_TITLE}
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
          👥 {users.length} {CLASS_LABELS.STATS.STUDENTS_SUFFIX}
        </span>
        <span className="stat-divider">•</span>
        <span className="stat-item">
          🧩 {problems.length} {CLASS_LABELS.STATS.PROBLEMS_SUFFIX}
        </span>
        <span className="stat-divider">•</span>
        <span className="stat-item">
          {CLASS_LABELS.MEDALS.GOLD} {CLASS_LABELS.STATS.LEADER_PREFIX}{' '}
          <span className="stat-highlight">{leaderName}</span>
        </span>
      </div>
    </header>
  );
}
