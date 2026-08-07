document.addEventListener('DOMContentLoaded', () => {
    const params = new URLSearchParams(window.location.search);
    const classCode = params.get('class') || 'basicos_1';

    fetch(`/exports/${encodeURIComponent(classCode)}_score.json`)
        .then(res => res.json())
        .then(data => renderScoreboard(data))
        .catch(err => {
            const tbody = document.getElementById('leaderboard');
            tbody.replaceChildren();
            const tr = document.createElement('tr');
            const td = document.createElement('td');
            td.colSpan = 4;
            // Using textContent treats inputs strictly as plain text, preventing XSS attacks
            td.textContent = `Error loading scoreboard: ${err.message}`;
            tr.appendChild(td);
            tbody.appendChild(tr);
        });

    function renderScoreboard(data) {
        document.getElementById('title').textContent = `Scoreboard: ${data.className || data.classCode}`;
        const users = (data.users || []).sort((a, b) => b.solvedCount - a.solvedCount);
        const tbody = document.getElementById('leaderboard');
        tbody.replaceChildren();

        if (users.length === 0) {
            const tr = document.createElement('tr');
            const td = document.createElement('td');
            td.colSpan = 4;
            td.textContent = 'No students found';
            tr.appendChild(td);
            tbody.appendChild(tr);
            return;
        }

        const fragment = document.createDocumentFragment();
        users.forEach((u, i) => {
            const tr = document.createElement('tr');

            const tdRank = document.createElement('td');
            tdRank.textContent = i + 1;

            // OWASP XSS Prevention: Building DOM nodes and assigning textContent directly prevents
            // XSS vulnerabilities when displaying untrusted user inputs (such as Discord nicknames).
            const tdName = document.createElement('td');
            tdName.textContent = `${u.displayName} (${u.userId})`;

            const tdCount = document.createElement('td');
            tdCount.textContent = u.solvedCount;

            const tdSolved = document.createElement('td');
            tdSolved.textContent = (u.solved || []).join(', ') || 'None';

            tr.append(tdRank, tdName, tdCount, tdSolved);
            fragment.appendChild(tr);
        });

        tbody.appendChild(fragment);
    }
});
