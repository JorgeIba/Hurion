import { describe, expect, it } from 'vitest';
import {
  evaluateScoreboardFilter,
  matchesFilterText,
  normalizeFilterText,
} from './filters';
import type {
  Filter,
  RowFilter,
  ScoreboardFilter,
  StudentFilter,
  MatchingTarget,
} from './filters';
import type { Problem, UserScore } from './types';
import { CLASS_CONFIG } from './constants';

const problems: Problem[] = [
  { id: 1, name: 'Two Sum', topic: 'Arreglos', createdAt: '2026-01-01' },
  { id: 104, name: 'Dijkstra Shortest Path', topic: ' Grafos ', createdAt: '2026-01-01' },
  { id: 257, name: 'max_flow', topic: 'Grafos', createdAt: '2026-01-01' },
  { id: 300, name: 'Mochila', topic: 'Programación Dinámica', createdAt: '2026-01-01' },
  { id: 400, name: 'Sum of Two Numbers', createdAt: '2026-01-01' },
];
const users: UserScore[] = [
  { userId: 'jorgeiba', displayName: 'Jorge Ibarra', solved: [1, 104] },
  { userId: 'usuario', displayName: 'Luis Gómez', solved: [257, 257], solvedCount: 99 },
  { userId: 'luis_dev', solvedCount: 99 },
];

function evaluate(filters?: ScoreboardFilter) {
  return evaluateScoreboardFilter({ problems, users, filters });
}

describe('matchesFilterText', () => {
  const fields: MatchingTarget[] = ['displayName', 'userId', 'problemName', 'topicName'];

  it.each(fields)('uses the same normalized substring policy for %s', (field) => {
    expect(matchesFilterText(' Programación Dinámica ', '  DINAMICA ', field)).toBe(true);
    expect(matchesFilterText('jorgeiba', 'jorge', field)).toBe(true);
    expect(matchesFilterText('jorgeiba', 'iba', field)).toBe(true);
    expect(matchesFilterText('jorge', 'jogre', field)).toBe(false);
  });

  it('ignores underscores on both sides only when matching topics', () => {
    expect(matchesFilterText('MAX_FLOW', 'maxflow', 'topicName')).toBe(true);
    expect(matchesFilterText('MaxFlow', 'max_flow', 'topicName')).toBe(true);
    expect(matchesFilterText('MÁX_FLOW', 'maxflow', 'topicName')).toBe(true);
    for (const field of ['displayName', 'userId', 'problemName'] as const) {
      expect(matchesFilterText('max_flow', 'maxflow', field)).toBe(false);
      expect(matchesFilterText('maxflow', 'max_flow', field)).toBe(false);
    }
  });

  it('matches continuous phrases rather than separate words', () => {
    expect(matchesFilterText('Two Sum II', 'two sum', 'problemName')).toBe(true);
    expect(matchesFilterText('Sum of Two Numbers', 'two sum', 'problemName')).toBe(false);
  });

  it('treats empty text as an unrestricted substring, including an empty candidate', () => {
    expect(matchesFilterText('Jorge', '  ', 'displayName')).toBe(true);
    expect(matchesFilterText('', '', 'displayName')).toBe(true);
    expect(matchesFilterText('', 'jorge', 'displayName')).toBe(false);
  });

  it('normalizes both precomposed and decomposed Unicode accents', () => {
    expect(normalizeFilterText('  A\u0301RBOL  ')).toBe('arbol');
    expect(matchesFilterText('Árbol', 'a\u0301rbol', 'topicName')).toBe(true);
  });
});

describe('evaluateScoreboardFilter', () => {
  it('keeps complete axes when filters are absent or empty', () => {
    for (const filter of [undefined, {}]) {
      const result = evaluate(filter);
      expect([...result.userIds]).toEqual(users.map((user) => user.userId));
      expect([...result.problemIds]).toEqual(problems.map((problem) => problem.id));
    }
  });

  it('matches display name OR handle even when the handle is masked by a display name', () => {
    const result = evaluate({
      students: {
        anyOf: [
          { condition: { by: 'displayName', value: 'gomez' } },
          { condition: { by: 'userId', value: 'jorgeiba' } },
          { condition: { by: 'userId', value: 'usuario' } },
        ],
      },
    });
    expect([...result.userIds]).toEqual(['jorgeiba', 'usuario']);
    expect(result.problemIds.size).toBe(problems.length);
  });

  it('matches a handle without a display name', () => {
    expect([
      ...evaluate({ students: { condition: { by: 'userId', value: 'lu' } } }).userIds,
    ]).toEqual(['luis_dev']);
  });

  it('does not treat unmatched student filters as absent', () => {
    const result = evaluate({
      students: { condition: { by: 'displayName', value: 'nadie' } },
    });
    expect(result.userIds.size).toBe(0);
    expect(result.problemIds.size).toBe(problems.length);
  });

  it('intersects topics and problem titles inside an OR expression', () => {
    const result = evaluate({
      rows: {
        anyOf: [
          {
            allOf: [
              { condition: { by: 'topicName', value: 'grafo' } },
              { condition: { by: 'name', value: 'shortest path' } },
            ],
          },
          { condition: { by: 'id', value: 1 } },
        ],
      },
    });
    expect([...result.problemIds]).toEqual([1, 104]);
    expect(result.userIds.size).toBe(users.length);
  });

  it('supports AND around nested OR without duplicating matched IDs', () => {
    const result = evaluate({
      rows: {
        allOf: [
          {
            anyOf: [
              { condition: { by: 'topicName', value: 'grafos' } },
              { condition: { by: 'id', value: 104 } },
            ],
          },
          { condition: { by: 'name', value: 'dijkstra' } },
        ],
      },
    });
    expect([...result.problemIds]).toEqual([104]);
  });

  it('matches IDs exactly, never by substring or prefix', () => {
    expect([
      ...evaluate({ rows: { condition: { by: 'id', value: 1 } } }).problemIds,
    ]).toEqual([1]);
    expect(
      evaluate({ rows: { condition: { by: 'id', value: 10 } } }).problemIds.size
    ).toBe(0);
  });

  it('uses the effective topic shown in the table, including its fallback', () => {
    expect([
      ...evaluate({ rows: { condition: { by: 'topicName', value: 'grafos' } } })
        .problemIds,
    ]).toEqual([104, 257]);
    expect([
      ...evaluate({ rows: { condition: { by: 'topicName', value: 'dinamica' } } })
        .problemIds,
    ]).toEqual([300]);
    expect([
      ...evaluate({
        rows: { condition: { by: 'topicName', value: CLASS_CONFIG.DEFAULT_TOPIC_NAME } },
      }).problemIds,
    ]).toEqual([400]);
  });

  it('uses solved IDs rather than solvedCount and keeps both axes independent', () => {
    const result = evaluate({
      students: { condition: { by: 'solvedProblem', value: 257 } },
      rows: { condition: { by: 'id', value: 1 } },
    });
    expect([...result.userIds]).toEqual(['usuario']);
    expect([...result.problemIds]).toEqual([1]);
  });

  it('supports a programmatic solver selection for one problem', () => {
    const result = evaluate({
      students: { condition: { by: 'solvedProblem', value: 104 } },
      rows: { condition: { by: 'id', value: 104 } },
    });
    expect([...result.userIds]).toEqual(['jorgeiba']);
    expect([...result.problemIds]).toEqual([104]);
  });

  it('uses false for empty OR and true for empty AND on either axis', () => {
    expect(evaluate({ students: { anyOf: [] }, rows: { allOf: [] } }).userIds.size).toBe(
      0
    );
    expect(
      evaluate({ students: { anyOf: [] }, rows: { allOf: [] } }).problemIds.size
    ).toBe(problems.length);
    expect(evaluate({ students: { allOf: [] }, rows: { anyOf: [] } }).userIds.size).toBe(
      users.length
    );
    expect(
      evaluate({ students: { allOf: [] }, rows: { anyOf: [] } }).problemIds.size
    ).toBe(0);
  });

  it('handles empty class data', () => {
    const result = evaluateScoreboardFilter({ problems: [], users: [] });
    expect(result.userIds.size).toBe(0);
    expect(result.problemIds.size).toBe(0);
  });

  it('does not mutate class data or the filter tree', () => {
    const filter: ScoreboardFilter = {
      students: { condition: { by: 'solvedProblem', value: 104 } },
      rows: { anyOf: [{ condition: { by: 'id', value: 104 } }] },
    };
    const before = JSON.stringify({ problems, users, filter });
    evaluate(filter);
    expect(JSON.stringify({ problems, users, filter })).toBe(before);
  });

  it('rejects ambiguous filter nodes and conditions from another axis at typecheck', () => {
    // @ts-expect-error A node cannot combine a condition and an operator.
    const mixed: Filter<RowFilter> = { condition: { by: 'id', value: 1 }, anyOf: [] };
    // @ts-expect-error Student conditions cannot be used in the row axis.
    const wrongAxis: Filter<RowFilter> = { condition: { by: 'userId', value: 'jorge' } };
    // @ts-expect-error Problem IDs must be numeric.
    const wrongValue: Filter<RowFilter> = { condition: { by: 'id', value: '1' } };
    const wrongStudent: Filter<StudentFilter> = {
      // @ts-expect-error Solver conditions must reference a numeric problem ID.
      condition: { by: 'solvedProblem', value: '104' },
    };
    void [mixed, wrongAxis, wrongValue, wrongStudent];
  });
});
