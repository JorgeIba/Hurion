import { getProblemPriorityLabel } from '@/domain/class/problems';

export function PriorityLabel({ priority }: { priority?: string | null }) {
  return (
    <span className="problem-priority-label">
      Priority: {getProblemPriorityLabel(priority)}
    </span>
  );
}
