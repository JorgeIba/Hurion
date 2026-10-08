import type {
  Filter,
  RowFilter,
  ScoreboardFilter,
  StudentFilter,
} from '@/domain/class/filters';

export interface SearchIssue {
  code: 'invalidProblemId' | 'invalidSolvedProblemId';
  token: string;
}

export interface ParsedScoreboardSearch {
  filters: ScoreboardFilter;
  issues: readonly SearchIssue[];
}

// A Unicode-aware regex for any letter or number, used to ignore empty tokens and prefixes.
const HAS_SEARCH_TEXT = /[\p{L}\p{N}]/u;

/**
 * Parses scoreboard search text into typed filters without evaluating class data.
 * Splits the query on +, then classifies whitespace-separated tokens in each branch:
 * @ selects students globally by display name OR handle; t:/topic: selects topics;
 * # selects an exact problem ID; solved: selects students who solved that ID;
 * remaining words form a continuous title phrase.
 * Within a row branch, repeated topics use OR and topic/title/ID restrictions use
 * AND. Repeated solved: conditions also use AND, with + joining their branches
 * by OR. Global @ matches and solved restrictions intersect. Students and rows
 * are independent axes: solved: never hides rows, and row terms never broaden
 * the student selection. An invalid ID makes its branch false only on its axis
 * and emits an issue; incomplete prefixes and branches without conditions on
 * that axis are omitted.
 *
 * Example: @jorge t:grafos t:dp shortest path + #257 produces:
 * - students: displayName contains "jorge" OR userId contains "jorge";
 * - rows: ((topicName contains "grafos" OR "dp") AND name contains "shortest path")
 *   OR id equals 257.
 *
 * Example: solved:304 + solved:305 selects students who solved either problem,
 * keeping all rows. solved:304 solved:305 requires both; @jorge solved:304
 * additionally requires a matching name or handle.
 *
 * The filter model supports recursive composition, but this syntax has no nested
 * groups or parentheses, so parsing only needs branch/token loops, not a stack.
 */
export function parseScoreboardSearch(query: string): ParsedScoreboardSearch {
  const students: Filter<StudentFilter>[] = [];
  const solvedBranches: Filter<StudentFilter>[] = [];
  const rows: Filter<RowFilter>[] = [];
  const issues: SearchIssue[] = [];

  for (const branch of query.split('+')) {
    const topics: Filter<RowFilter>[] = [];
    const conditions: Filter<RowFilter>[] = [];
    const titleWords: string[] = [];
    const solvedConditions: Filter<StudentFilter>[] = [];
    let invalidRow = false;
    let invalidStudent = false;

    for (const token of branch.trim().split(/\s+/).filter(Boolean)) {
      if (token.startsWith('@')) {
        const value = token.slice(1);
        if (HAS_SEARCH_TEXT.test(value)) {
          students.push(
            { condition: { by: 'displayName', value } },
            { condition: { by: 'userId', value } }
          );
        }
      } else if (/^(?:t|topic):/i.test(token)) {
        const value = token.slice(token.indexOf(':') + 1);
        if (HAS_SEARCH_TEXT.test(value)) {
          topics.push({ condition: { by: 'topicName', value } });
        }
      } else if (/^solved:/i.test(token)) {
        const value = token.slice(token.indexOf(':') + 1);
        if (!value) continue;
        const id = parseProblemId(value);
        if (id !== undefined) {
          solvedConditions.push({ condition: { by: 'solvedProblem', value: id } });
        } else {
          issues.push({ code: 'invalidSolvedProblemId', token });
          invalidStudent = true;
        }
      } else if (token.startsWith('#')) {
        const value = token.slice(1);
        if (!value) continue;
        const id = parseProblemId(value);
        if (id !== undefined) {
          conditions.push({ condition: { by: 'id', value: id } });
        } else {
          issues.push({ code: 'invalidProblemId', token });
          invalidRow = true;
        }
      } else if (HAS_SEARCH_TEXT.test(token)) {
        titleWords.push(token);
      }
    }

    if (invalidStudent) {
      solvedBranches.push({ anyOf: [] });
    } else if (solvedConditions.length > 0) {
      solvedBranches.push(
        solvedConditions.length === 1 ? solvedConditions[0] : { allOf: solvedConditions }
      );
    }

    if (invalidRow) {
      // An invalid AND condition makes its entire branch false, never unrestricted.
      rows.push({ anyOf: [] });
      continue;
    }
    if (topics.length > 0) {
      conditions.push(topics.length === 1 ? topics[0] : { anyOf: topics });
    }
    if (titleWords.length > 0) {
      conditions.push({ condition: { by: 'name', value: titleWords.join(' ') } });
    }
    if (conditions.length > 0) {
      rows.push(conditions.length === 1 ? conditions[0] : { allOf: conditions });
    }
  }

  const filters: ScoreboardFilter = {};
  const studentRestrictions: Filter<StudentFilter>[] = [];
  if (students.length > 0) studentRestrictions.push({ anyOf: students });
  if (solvedBranches.length > 0) {
    studentRestrictions.push(
      solvedBranches.length === 1 ? solvedBranches[0] : { anyOf: solvedBranches }
    );
  }
  if (studentRestrictions.length > 0) {
    filters.students =
      studentRestrictions.length === 1
        ? studentRestrictions[0]
        : { allOf: studentRestrictions };
  }
  if (rows.length > 0) filters.rows = rows.length === 1 ? rows[0] : { anyOf: rows };
  return { filters, issues };
}

function parseProblemId(value: string): number | undefined {
  const id = Number(value);
  return /^\d+$/.test(value) && Number.isSafeInteger(id) ? id : undefined;
}
