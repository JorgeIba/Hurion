# Scoreboard User Guide

Open a class's interactive scoreboard with `!score [class_code]`. If `class_code` is omitted, Hurion uses the class linked to the current Discord channel. The scoreboard has two views: **Standings** and **Problems**.

## Standings

Standings lists students by solved-problem count, with problem columns grouped by topic. Click a student's name to cycle through that student's view:

1. **Top ACs** puts solved problems first within each topic.
2. **Backlog first** puts unsolved problems first.
3. Click the same name again to return to the default order.

The order selector controls the problem order within each topic and breaks ties within the solved or pending groups.

### Search

Use the search control above the standings. Press **Enter** to apply the query. A query stays visible as a compact chip when the input closes; clear it to remove the search. The search affects students and problem columns independently.

| Search text | What it matches |
|---|---|
| `@jorge` | Students whose display name or Discord ID contains `jorge`. |
| `t:grafos` or `topic:grafos` | Problems whose topic contains `grafos`. |
| `#12` | The problem with exactly ID `12`. |
| `solved:12` | Students who solved problem `12`; this does not hide problem columns. |
| `shortest path` | Problems whose name contains the phrase `shortest path`. |
| `solved:12 + solved:18` | Students who solved problem `12` or `18`. |

Search ignores letter case and accents. Topic search also ignores underscores. Repeated topic terms are alternatives; a topic, title, and ID restriction in the same branch must all match. For students, `@name` matches display names or Discord IDs and narrows matching solves; repeated `@name` terms match any listed name, while repeated `solved:` terms require all listed problems. Use `+` for alternatives; an `@name` filter applies to each `solved:` alternative. For example, `@jorge solved:12` matches Jorge only if they solved problem 12, while `t:grafos #12` shows problem 12 only if its topic also matches `grafos`.

Invalid problem IDs are reported below the search control. Use a positive numeric ID after `#` or `solved:`.

## Problems

Problems shows the class catalog grouped by topic. The difficulty and priority filters apply in both views. The order selector is shared when switching between Standings and Problems; changing classes resets these controls.

The difficulty filter offers Easy, Medium, and Hard. The priority filter offers Essential, Recommended, Supplementary, Not Required, and Discouraged. The sort menu offers highest or lowest ID first, easy or hard difficulty first, and priority from most recommended to least recommended or in reverse. Problems without difficulty or priority sort last. In the default Standings order and in Problems, topic groups follow the highest ID they contain; choosing lowest ID first orders topics by their lowest ID. When sorting a student's Standings, topics instead follow that student's solved or pending counts.

Use **Expand All** or **Collapse All** to control every visible topic, or toggle a topic by its heading. Filters expand matching topics temporarily and restore their prior collapsed state when the filters are cleared. The count shows how many problems match the current filters.

## Narrow screens

The problem column starts narrower on mobile and can still be resized with its handle. Double-click the handle to reset its width. When the handle is focused, use the arrow keys to adjust the width.
