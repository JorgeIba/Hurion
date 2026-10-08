import type {
  Filter,
  RowFilter,
  ScoreboardFilter,
  StudentFilter,
} from '@/domain/class/filters';

export interface SearchIssue {
  code: 'invalidProblemId';
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
 * # selects an exact problem ID; remaining words form a continuous title phrase.
 * Within a row branch, repeated topics use OR and topic/title/ID restrictions use
 * AND. Row branches use OR. An invalid ID makes its branch false and emits an issue;
 * incomplete prefixes and branches without row conditions are omitted.
 *
 * Example: @jorge t:grafos t:dp shortest path + #257 produces:
 * - students: displayName contains "jorge" OR userId contains "jorge";
 * - rows: ((topicName contains "grafos" OR "dp") AND name contains "shortest path")
 *   OR id equals 257.
 *
 * The filter model supports recursive composition, but this syntax has no nested
 * groups or parentheses, so parsing only needs branch/token loops, not a stack.
 */
export function parseScoreboardSearch(query: string): ParsedScoreboardSearch {
  const students: Filter<StudentFilter>[] = [];
  const rows: Filter<RowFilter>[] = [];
  const issues: SearchIssue[] = [];

  for (const branch of query.split('+')) {
    const topics: Filter<RowFilter>[] = [];
    const conditions: Filter<RowFilter>[] = [];
    const titleWords: string[] = [];
    let invalid = false;

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
      } else if (token.startsWith('#')) {
        const value = token.slice(1);
        if (!value) continue;
        const id = Number(value);
        if (/^\d+$/.test(value) && Number.isSafeInteger(id)) {
          conditions.push({ condition: { by: 'id', value: id } });
        } else {
          issues.push({ code: 'invalidProblemId', token });
          invalid = true;
        }
      } else if (HAS_SEARCH_TEXT.test(token)) {
        titleWords.push(token);
      }
    }

    if (invalid) {
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
  if (students.length > 0) filters.students = { anyOf: students };
  if (rows.length > 0) filters.rows = rows.length === 1 ? rows[0] : { anyOf: rows };
  return { filters, issues };
}
