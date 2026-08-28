import React, { useRef } from 'react';
import { ActiveTab, CLASS_LABELS } from '@/domain/class';

interface NavigationTabsProps {
  activeTab: ActiveTab;
  onTabChange: (tab: ActiveTab) => void;
}

export function NavigationTabs({
  activeTab,
  onTabChange,
}: NavigationTabsProps) {
  const tabs: { id: ActiveTab; label: string; icon: string }[] = [
    {
      id: 'standings',
      label: CLASS_LABELS.TABS.STANDINGS,
      icon: '🏆',
    },
    {
      id: 'problems',
      label: CLASS_LABELS.TABS.PROBLEMS,
      icon: '🧩',
    },
  ];

  const tabButtonsRef = useRef<(HTMLButtonElement | null)[]>([]);

  const handleKeyDown = (e: React.KeyboardEvent, index: number) => {
    let targetIndex = -1;
    if (e.key === 'ArrowRight') {
      targetIndex = (index + 1) % tabs.length;
    } else if (e.key === 'ArrowLeft') {
      targetIndex = (index - 1 + tabs.length) % tabs.length;
    } else if (e.key === 'Home') {
      targetIndex = 0;
    } else if (e.key === 'End') {
      targetIndex = tabs.length - 1;
    }

    if (targetIndex !== -1) {
      e.preventDefault();
      const nextTab = tabs[targetIndex];
      onTabChange(nextTab.id);
      tabButtonsRef.current[targetIndex]?.focus();
    }
  };

  return (
    <nav className="nav-tabs-container" role="tablist" aria-label="Class Views">
      {tabs.map((tab, idx) => {
        const isActive = activeTab === tab.id;
        return (
          <button
            key={tab.id}
            ref={(el) => {
              tabButtonsRef.current[idx] = el;
            }}
            id={`tab-${tab.id}`}
            role="tab"
            type="button"
            className={`nav-tab-btn ${isActive ? 'nav-tab-active' : ''}`}
            aria-selected={isActive}
            aria-controls={`tabpanel-${tab.id}`}
            tabIndex={isActive ? 0 : -1}
            onClick={() => onTabChange(tab.id)}
            onKeyDown={(e) => handleKeyDown(e, idx)}
          >
            <span className="nav-tab-icon" aria-hidden="true">
              {tab.icon}
            </span>
            <span className="nav-tab-label">{tab.label}</span>
          </button>
        );
      })}
    </nav>
  );
}
