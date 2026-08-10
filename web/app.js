document.addEventListener('DOMContentLoaded', () => {
    const params = new URLSearchParams(window.location.search);
    const classCode = params.get('class') || 'basicos';

    fetch(`/exports/${encodeURIComponent(classCode)}_score.json`)
        .then(res => {
            if (!res.ok) {
                throw new Error(`Scoreboard for class "${classCode}" not found (${res.status})`);
            }
            return res.json();
        })
        .then(data => renderScoreboard(data))
        .catch(err => {
            const tbody = document.getElementById('leaderboard');
            tbody.replaceChildren();
            const tr = document.createElement('tr');
            const td = document.createElement('td');
            td.colSpan = 10;
            td.className = 'state-row';
            td.textContent = `Error loading scoreboard: ${err.message}`;
            tr.appendChild(td);
            tbody.appendChild(tr);
        });

    function renderScoreboard(data) {
        const titleEl = document.getElementById('title');
        if (titleEl) {
            titleEl.textContent = data.className || data.classCode || 'Scoreboard';
        }

        const problems = data.problems || [];
        const getSolvedCount = u => u.solvedCount ?? (u.solved ? u.solved.length : 0);
        const users = (data.users || []).sort((a, b) => getSolvedCount(b) - getSolvedCount(a));

        // 1. Update Class Health Summary Bar (/layout)
        const statStudents = document.getElementById('stat-students');
        const statProblems = document.getElementById('stat-problems');
        const statLeader = document.getElementById('stat-leader');

        if (statStudents) statStudents.textContent = `👥 ${users.length} Students`;
        if (statProblems) statProblems.textContent = `🧩 ${problems.length} Problems`;
        if (statLeader) {
            const leaderName = users.length > 0 ? (users[0].displayName || users[0].userId) : '--';
            statLeader.innerHTML = `👑 Leader: <span class="stat-highlight">${escapeHTML(leaderName)}</span>`;
        }

        const headerRow = document.getElementById('table-header-row');
        const subHeaderRow = document.getElementById('table-sub-header-row');

        // 2. Build Student Column Headers & Total ACs Sub-Header Row using DocumentFragments for zero reflows
        if (headerRow) {
            // Keep first 2 columns: (ID) Problem, Difficulty
            while (headerRow.children.length > 2) {
                headerRow.removeChild(headerRow.lastChild);
            }
            if (subHeaderRow) {
                while (subHeaderRow.children.length > 2) {
                    subHeaderRow.removeChild(subHeaderRow.lastChild);
                }
            }

            const headerFragment = document.createDocumentFragment();
            const subHeaderFragment = document.createDocumentFragment();

            users.forEach((u, index) => {
                // Header Row 1: Student Handle
                const th = document.createElement('th');
                th.className = 'col-student-header';
                th.setAttribute('scope', 'col');
                
                if (index === 0) th.classList.add('rank-1-header');
                else if (index === 1) th.classList.add('rank-2-header');
                else if (index === 2) th.classList.add('rank-3-header');

                const wrapper = document.createElement('div');
                wrapper.className = 'student-header-inner';

                if (index === 0) {
                    const crown = document.createElement('span');
                    crown.className = 'student-crown';
                    crown.textContent = '👑';
                    wrapper.appendChild(crown);
                } else if (index === 1) {
                    const crown = document.createElement('span');
                    crown.className = 'student-crown';
                    crown.textContent = '🥈';
                    wrapper.appendChild(crown);
                } else if (index === 2) {
                    const crown = document.createElement('span');
                    crown.className = 'student-crown';
                    crown.textContent = '🥉';
                    wrapper.appendChild(crown);
                }

                const nameSpan = document.createElement('span');
                nameSpan.textContent = u.displayName || u.userId;
                wrapper.appendChild(nameSpan);

                if (problems.length > 0 && getSolvedCount(u) >= problems.length) {
                    const mastered = document.createElement('span');
                    mastered.className = 'mastered-badge';
                    mastered.textContent = '✨ 100%';
                    wrapper.appendChild(mastered);
                }

                th.appendChild(wrapper);
                th.title = u.userId ? `@${u.userId} (${getSolvedCount(u)} / ${problems.length} solved)` : '';
                headerFragment.appendChild(th);

                // Header Row 2 (Sub-header): Total AC Count per Student
                if (subHeaderRow) {
                    const thCount = document.createElement('th');
                    thCount.className = 'total-ac-count-cell';
                    thCount.setAttribute('scope', 'col');
                    if (index === 0) thCount.classList.add('rank-1-ac');
                    thCount.textContent = String(getSolvedCount(u));
                    subHeaderFragment.appendChild(thCount);
                }
            });

            headerRow.appendChild(headerFragment);
            if (subHeaderRow) subHeaderRow.appendChild(subHeaderFragment);
        }

        const tbody = document.getElementById('leaderboard');
        tbody.replaceChildren();

        const totalCols = 2 + users.length;

        if (problems.length === 0) {
            const tr = document.createElement('tr');
            const td = document.createElement('td');
            td.colSpan = totalCols;
            td.className = 'state-row';
            td.textContent = 'No problems recorded for this class';
            tr.appendChild(td);
            tbody.appendChild(tr);
            return;
        }

        // Group problems by topic
        const topicGroups = new Map();
        problems.forEach(p => {
            const topicName = p.topic || 'General';
            if (!topicGroups.has(topicName)) {
                topicGroups.set(topicName, []);
            }
            topicGroups.get(topicName).push(p);
        });

        const userSolvedMaps = users.map(u => new Set(u.solved || []));
        const fragment = document.createDocumentFragment();

        // 3. Build Transposed Rows (/colorize difficulty row tints & /colorize code badges)
        topicGroups.forEach((probList, topicName) => {
            // Topic Section Header Row (/bolder + /typeset)
            const topicRow = document.createElement('tr');
            topicRow.className = 'topic-header-row';
            const topicTd = document.createElement('td');
            topicTd.colSpan = totalCols;
            topicTd.textContent = `// TOPIC: ${topicName.toUpperCase()}`;
            topicRow.appendChild(topicTd);
            fragment.appendChild(topicRow);

            probList.forEach(p => {
                const tr = document.createElement('tr');
                tr.className = 'problem-row';

                const diffLower = (p.difficulty || '').toLowerCase();
                let diffClass = 'problem-row-medium';
                let badgeClass = 'badge-medium';
                let diffDotClass = 'diff-medium';
                let diffLabel = 'Medium';

                if (diffLower.includes('easy')) {
                    diffClass = 'problem-row-easy';
                    badgeClass = 'badge-easy';
                    diffDotClass = 'diff-easy';
                    diffLabel = 'Easy';
                } else if (diffLower.includes('hard')) {
                    diffClass = 'problem-row-hard';
                    badgeClass = 'badge-hard';
                    diffDotClass = 'diff-hard';
                    diffLabel = 'Hard';
                } else if (diffLower.includes('not specified') || diffLower === '-' || !diffLower) {
                    diffClass = 'problem-row-unspecified';
                    badgeClass = 'badge-unspecified';
                    diffDotClass = 'diff-unspecified';
                    diffLabel = 'Not specified';
                }

                tr.classList.add(diffClass);

                // Col 1: Colorized Monospace Code Badge + Problem Name (/colorize + /bolder)
                const tdProblem = document.createElement('td');
                tdProblem.className = 'problem-title-cell';

                const codeBadge = document.createElement('span');
                codeBadge.className = 'problem-code-badge ' + badgeClass;
                codeBadge.textContent = `[${String(p.id).padStart(2, '0')}]`;

                const nameText = document.createTextNode(p.name || 'Problem');
                tdProblem.append(codeBadge, nameText);

                // Col 2: Difficulty Pure Color Dot with WCAG Accessible ARIA Label
                const tdDiff = document.createElement('td');
                tdDiff.className = 'diff-cell';
                const diffDot = document.createElement('span');
                diffDot.className = 'diff-dot ' + diffDotClass;
                diffDot.title = diffLabel;
                diffDot.setAttribute('aria-label', 'Difficulty: ' + diffLabel);

                tdDiff.appendChild(diffDot);

                tr.append(tdProblem, tdDiff);

                // Cols 3..N: Student Solved Cells (/delight 3D glass AC pills)
                userSolvedMaps.forEach(solvedSet => {
                    const tdStatus = document.createElement('td');
                    tdStatus.className = 'status-cell';

                    if (solvedSet.has(p.id)) {
                        const acPill = document.createElement('span');
                        acPill.className = 'status-ac-pill';
                        acPill.textContent = '✓';
                        tdStatus.appendChild(acPill);
                    }

                    tr.appendChild(tdStatus);
                });

                fragment.appendChild(tr);
            });
        });

        tbody.appendChild(fragment);
    }

    function escapeHTML(str) {
        return String(str)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }
});
