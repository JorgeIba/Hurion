CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    CONSTRAINT users_display_name_nonempty CHECK (trim(display_name) <> '')
);

CREATE TABLE IF NOT EXISTS classes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_code TEXT UNIQUE,
    name TEXT
);

CREATE TABLE IF NOT EXISTS classrooms (
    room_id TEXT PRIMARY KEY,
    class_code TEXT NOT NULL,
    FOREIGN KEY (class_code) REFERENCES classes(class_code) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS class_access_roles (
    class_code TEXT NOT NULL,
    role_id TEXT NOT NULL,
    PRIMARY KEY (class_code, role_id),
    FOREIGN KEY (class_code) REFERENCES classes(class_code) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS problems (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    url TEXT,
    class_id INTEGER,
    difficulty TEXT,
    topic TEXT,
    UNIQUE(name, class_id),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS enrollments (
    user_id TEXT,
    class_id INTEGER,
    PRIMARY KEY (user_id, class_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS solved_problems (
    user_id TEXT,
    problem_id INTEGER,
    solved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, problem_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pending_confirmations (
    message_id TEXT,
    user_id TEXT,
    problem_id INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, problem_id)
);
