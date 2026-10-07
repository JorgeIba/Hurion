import { describe, expect, it } from 'vitest';
import { parseScoreboardSearch } from '@/domain/class/parser';
import { evaluateScoreboardFilter } from '@/domain/class/filters';
import type { Problem, UserScore } from '@/domain/class/types';

const problems: Problem[] = [
  { id: 1, name: 'Two Sum', topic: 'arreglos', createdAt: '2026-01-01' },
  { id: 104, name: 'Dijkstra Shortest Path', topic: 'grafos', createdAt: '2026-01-01' },
  { id: 257, name: 'Max Flow', topic: 'max_flow', createdAt: '2026-01-01' },
  { id: 300, name: 'Dijkstra DP', topic: 'dp', createdAt: '2026-01-01' },
  { id: 400, name: '3 Sum', topic: 'arreglos', createdAt: '2026-01-01' },
  { id: 500, name: 'Sum of Two Numbers', createdAt: '2026-01-01' },
];
const users: UserScore[] = [
  { userId: 'jorgeiba', displayName: 'Jorge Ibarra' },
  { userId: 'usuario', displayName: 'Luis Gómez' },
  { userId: 'luis_dev' },
];

function search(query: string, problemCatalog: Problem[] = problems) {
  const parsed = parseScoreboardSearch(query);
  const result = evaluateScoreboardFilter({
    problems: problemCatalog,
    users,
    filters: parsed.filters,
  });
  return {
    problemIds: [...result.problemIds],
    userIds: [...result.userIds],
    issues: parsed.issues,
  };
}

describe('parseScoreboardSearch', () => {
  it.each(['', '   ', '+', ' + + ', '@', 't:', 'topic:', '#', '? ! / :::', 't:___'])(
    'omits restrictions for empty or incomplete input %j',
    (query) => {
      expect(parseScoreboardSearch(query)).toEqual({ filters: {}, issues: [] });
    }
  );

  it('produces a single condition for a title phrase', () => {
    expect(parseScoreboardSearch('  Two   Sum  ')).toEqual({
      filters: { rows: { condition: { by: 'name', value: 'Two Sum' } } },
      issues: [],
    });
  });

  it('constructs global name OR handle filters without a row restriction', () => {
    expect(parseScoreboardSearch('@jorge')).toEqual({
      filters: {
        students: {
          anyOf: [
            { condition: { by: 'displayName', value: 'jorge' } },
            { condition: { by: 'userId', value: 'jorge' } },
          ],
        },
      },
      issues: [],
    });
  });

  it.each([
    ['@jorge', ['jorgeiba']],
    ['@jorgeiba', ['jorgeiba']],
    ['@GOMEZ', ['usuario']],
    ['@lu', ['usuario', 'luis_dev']],
    ['@luis_dev', ['luis_dev']],
    ['@jorge + @usuario', ['jorgeiba', 'usuario']],
    ['@nadie', []],
  ])('selects students independently for %s', (query, expected) => {
    const result = search(query);
    expect(result.userIds).toEqual(expected);
    expect(result.problemIds).toEqual(problems.map((problem) => problem.id));
  });

  it.each([
    ['#1', [1]],
    ['#10', []],
    ['#0104', [104]],
    ['#257', [257]],
    ['two sum', [1]],
    ['3 sum', [400]],
    ['257', []],
    ['T:grafos', [104]],
    ['Topic:DP', [300]],
    ['t:maxflow', [257]],
    ['t:grafos dijkstra', [104]],
    ['t:grafos + dijkstra', [104, 300]],
    ['t:grafos t:dp dijkstra', [104, 300]],
    ['t:grafos t:dp shortest path', [104]],
    ['t:grafos #257', []],
    ['t:grafos + #257', [104, 257]],
    ['#104 #257', []],
    ['#104 + #257', [104, 257]],
    ['t:grafos + dijkstra + #104', [104, 300]],
    ['dijkstra +', [104, 300]],
    ['+ + dijkstra ++', [104, 300]],
    ['dijkstra t:', [104, 300]],
    ['t:___ #257', [257]],
  ])('selects rows independently for %s', (query, expected) => {
    const result = search(query);
    expect(result.problemIds).toEqual(expected);
    expect(result.userIds).toEqual(users.map((user) => user.userId));
    expect(result.issues).toEqual([]);
  });

  it.each(['@usuario + #257', '#257 + @usuario', '@usuario #257', '@usuario + #257 +'])(
    'never creates an unrestricted row branch from student terms in %s',
    (query) => {
      expect(search(query)).toEqual({
        userIds: ['usuario'],
        problemIds: [257],
        issues: [],
      });
    }
  );

  it('extracts students globally while preserving row AND/OR meaning', () => {
    expect(search('@jorge t:grafos shortest path + @luis_dev #257')).toEqual({
      userIds: ['jorgeiba', 'luis_dev'],
      problemIds: [104, 257],
      issues: [],
    });
  });

  it('distinguishes topic/title intersection from their inclusive union', () => {
    const problemCatalog: Problem[] = [
      ...problems,
      { id: 600, name: 'BFS', topic: 'grafos', createdAt: '2026-01-01' },
    ];

    // AND excludes BFS and the Dijkstra problem outside Grafos.
    expect(search('t:grafos dijkstra', problemCatalog)).toEqual({
      userIds: users.map((user) => user.userId),
      problemIds: [104],
      issues: [],
    });
    // OR includes Grafos-only, title-only and overlapping matches exactly once.
    expect(search('t:grafos + dijkstra', problemCatalog)).toEqual({
      userIds: users.map((user) => user.userId),
      problemIds: [104, 300, 600],
      issues: [],
    });
  });

  it.each([
    '#abc',
    '#!!!',
    '#___',
    '#1.5',
    '#-1',
    '#1e2',
    '#9007199254740992',
    '#Infinity',
  ])('invalidates the row branch and reports malformed ID %s', (token) => {
    expect(search(`t:grafos ${token}`)).toEqual({
      userIds: users.map((user) => user.userId),
      problemIds: [],
      issues: [{ code: 'invalidProblemId', token }],
    });
    expect(parseScoreboardSearch(token).filters.rows).toEqual({ anyOf: [] });
  });

  it('keeps valid OR branches when another branch is invalid', () => {
    expect(search('t:grafos #abc + #257')).toEqual({
      userIds: users.map((user) => user.userId),
      problemIds: [257],
      issues: [{ code: 'invalidProblemId', token: '#abc' }],
    });
  });

  it('retains global student terms even in invalid row branches', () => {
    expect(search('@jorge #abc + @usuario #257')).toEqual({
      userIds: ['jorgeiba', 'usuario'],
      problemIds: [257],
      issues: [{ code: 'invalidProblemId', token: '#abc' }],
    });
  });

  it('reports every invalid ID in input order', () => {
    expect(parseScoreboardSearch('#abc #1.5 + #-1').issues).toEqual([
      { code: 'invalidProblemId', token: '#abc' },
      { code: 'invalidProblemId', token: '#1.5' },
      { code: 'invalidProblemId', token: '#-1' },
    ]);
  });
});
