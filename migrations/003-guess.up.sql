CREATE TABLE guesses (
    id INTEGER PRIMARY KEY,
    guild_id INTEGER NOT NULL,
    value INTEGER NOT NULL,
    username TEXT NOT NULL
);
