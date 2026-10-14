import { useCallback, useState } from 'react';

export interface UseTopicCollapseResult {
  collapsedTopics: Set<string>;
  isCollapsed: (topicName: string) => boolean;
  toggleTopic: (topicName: string) => void;
  expandAll: (allTopics: string[]) => void;
  collapseAll: (allTopics: string[]) => void;
  isAllCollapsed: (allTopics: string[]) => boolean;
}

/**
 * Custom hook to manage collapsible topic sections across scoreboard views.
 */
export function useTopicCollapse(
  initialCollapsed: string[] = []
): UseTopicCollapseResult {
  const [collapsedTopics, setCollapsedTopics] = useState<Set<string>>(
    () => new Set(initialCollapsed)
  );

  const isCollapsed = useCallback(
    (topicName: string) => collapsedTopics.has(topicName),
    [collapsedTopics]
  );

  const toggleTopic = useCallback((topicName: string) => {
    setCollapsedTopics((prev) => {
      const next = new Set(prev);
      if (next.has(topicName)) {
        next.delete(topicName);
      } else {
        next.add(topicName);
      }
      return next;
    });
  }, []);

  const expandAll = useCallback((allTopics: string[]) => {
    setCollapsedTopics((prev) => {
      const next = new Set(prev);
      for (const topic of allTopics) next.delete(topic);
      return next;
    });
  }, []);

  const collapseAll = useCallback((allTopics: string[]) => {
    setCollapsedTopics((prev) => new Set([...prev, ...allTopics]));
  }, []);

  const isAllCollapsed = useCallback(
    (allTopics: string[]) =>
      allTopics.length > 0 && allTopics.every((t) => collapsedTopics.has(t)),
    [collapsedTopics]
  );

  return {
    collapsedTopics,
    isCollapsed,
    toggleTopic,
    expandAll,
    collapseAll,
    isAllCollapsed,
  };
}
