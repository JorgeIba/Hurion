document.addEventListener('DOMContentLoaded', () => {
    const params = new URLSearchParams(window.location.search);
    const classCode = params.get('class') || 'basicos_1';

    fetch(`/exports/${classCode}_score.json`)
        .then(res => res.json())
        .then(data => renderScoreboard(data))
        .catch(err => {
            document.getElementById('leaderboard').innerHTML = `<tr><td colspan="4">Error loading scoreboard: ${err.message}</td></tr>`;
        });

    function renderScoreboard(data) {
        document.getElementById('title').textContent = `Scoreboard: ${data.className || data.classCode}`;
        const users = (data.users || []).sort((a, b) => b.solvedCount - a.solvedCount);
        const tbody = document.getElementById('leaderboard');

        if (users.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4">No students found</td></tr>';
            return;
        }

        tbody.innerHTML = users.map((u, i) => `
            <tr>
                <td>${i + 1}</td>
                <td>${u.displayName} (${u.userId})</td>
                <td>${u.solvedCount}</td>
                <td>${(u.solved || []).join(', ') || 'None'}</td>
            </tr>
        `).join('');
    }
});
