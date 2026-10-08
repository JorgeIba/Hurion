import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { Problem, UserScore } from '@/domain/class/types';
import { useScoreboardSearch } from '@/hooks/useScoreboardSearch';

const problems: Problem[] = [
  { id: 12, name: 'BFS', topic: 'grafos', createdAt: '2026-01-01' },
  { id: 120, name: 'Grafo implícito', topic: 'dp', createdAt: '2026-01-01' },
  { id: 3, name: 'Mochila', topic: 'dp', createdAt: '2026-01-01' },
];
const users: UserScore[] = [
  { userId: 'jorgeiba', displayName: 'Jorge', solved: [12] },
  { userId: 'luis', displayName: 'Luis', solved: [3] },
];

function resolve(query: string, classProblems = problems, classUsers = users) {
  let result!: ReturnType<typeof useScoreboardSearch>;
  function Probe() {
    result = useScoreboardSearch(query, classProblems, classUsers);
    return null;
  }
  renderToStaticMarkup(<Probe />);
  return result;
}

describe('scoreboard search resolution', () => {
  it.each(['', '   ', '@ t: #', '+ +'])('keeps %j unrestricted', (query) => {
    const result = resolve(query);
    expect(result.isFiltering).toBe(false);
    expect(result.selection).toBeUndefined();
    expect(result.issues).toEqual([]);
  });

  it('combines a global student restriction with alternative row matches', () => {
    const result = resolve('@jorge t:grafos + grafo');
    expect(result.selection?.userIds).toEqual(new Set(['jorgeiba']));
    expect(result.selection?.problemIds).toEqual(new Set([12, 120]));
    expect(result.isFiltering).toBe(true);
  });

  it('keeps the unrequested axis complete and problem IDs exact', () => {
    const result = resolve('#12');
    expect(result.selection?.userIds).toEqual(new Set(['jorgeiba', 'luis']));
    expect(result.selection?.problemIds).toEqual(new Set([12]));
    expect(resolve('@luis').selection?.problemIds).toEqual(new Set([12, 120, 3]));
  });

  it('retains active empty matches instead of treating them as no restriction', () => {
    const result = resolve('@unknown #999');
    expect(result.isFiltering).toBe(true);
    expect(result.selection?.userIds.size).toBe(0);
    expect(result.selection?.problemIds.size).toBe(0);
  });

  it('reports invalid IDs while retaining valid alternatives', () => {
    const result = resolve('#wrong + #3');
    expect(result.selection?.problemIds).toEqual(new Set([3]));
    expect(result.issues).toEqual([{ code: 'invalidProblemId', token: '#wrong' }]);
    expect(resolve('#wrong').selection?.problemIds.size).toBe(0);
  });

  it('evaluates against the supplied class data without changing it', () => {
    const before = JSON.stringify({ problems, users });
    expect(resolve('#12', [problems[2]], [users[1]]).selection).toEqual({
      userIds: new Set(['luis']),
      problemIds: new Set(),
    });
    expect(JSON.stringify({ problems, users })).toBe(before);
  });
});
