CREATE TABLE IF NOT EXISTS user_names (
    user_name TEXT PRIMARY KEY,
    name TEXT
);

CREATE TABLE IF NOT EXISTS users (
    discord_id TEXT PRIMARY KEY,
    user_name TEXT,
    FOREIGN KEY (user_name) REFERENCES user_names(user_name)
);

CREATE TABLE IF NOT EXISTS classes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_code TEXT UNIQUE,
    name TEXT
);

CREATE TABLE IF NOT EXISTS problems (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    class_id INTEGER,
    FOREIGN KEY (class_id) REFERENCES classes(id)
);

CREATE TABLE IF NOT EXISTS enrollments (
    user_id TEXT,
    class_id INTEGER,
    registered_name TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, class_id),
    FOREIGN KEY (user_id) REFERENCES users(discord_id),
    FOREIGN KEY (class_id) REFERENCES classes(id)
);

CREATE TABLE IF NOT EXISTS solved_problems (
    user_id TEXT,
    problem_id INTEGER,
    class_id INTEGER,
    solved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, problem_id, class_id),
    FOREIGN KEY (user_id) REFERENCES users(discord_id),
    FOREIGN KEY (problem_id) REFERENCES problems(id),
    FOREIGN KEY (class_id) REFERENCES classes(id)
);